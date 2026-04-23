package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrRegistryStopped is returned when Run is called after Shutdown.
var ErrRegistryStopped = errors.New("chat worker registry is stopped")

type job struct {
	ctx    context.Context
	fn     func(context.Context) error
	result chan error
}

type chatWorker struct {
	ch       chan job
	done     chan struct{}
	lastUsed atomic.Int64 // unix seconds; reserved for future idle-eviction
}

func newChatWorker(bufSize int) *chatWorker {
	w := &chatWorker{
		ch:   make(chan job, bufSize),
		done: make(chan struct{}),
	}
	return w
}

func (w *chatWorker) loop(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-w.done:
			return
		case j := <-w.ch:
			w.lastUsed.Store(time.Now().Unix())
			func() {
				defer func() {
					if r := recover(); r != nil {
						j.result <- fmt.Errorf("chat worker panic: %v", r)
					}
				}()
				if err := j.ctx.Err(); err != nil {
					j.result <- err
					return
				}
				j.result <- j.fn(j.ctx)
			}()
		}
	}
}

// ChatWorkerRegistry implements domain.ChatExecutor using one goroutine per active chat.
// Same-chat operations are serialized; different chats run in parallel.
// Workers are process-lifetime (chat count is small; lastUsed is in place for future idle-eviction).
type ChatWorkerRegistry struct {
	mu      sync.Mutex
	workers map[int64]*chatWorker
	wg      sync.WaitGroup
	bufSize int
	stopped atomic.Bool
}

// NewChatWorkerRegistry creates a registry with the given job queue buffer size per worker.
func NewChatWorkerRegistry(bufSize int) *ChatWorkerRegistry {
	return &ChatWorkerRegistry{
		workers: make(map[int64]*chatWorker),
		bufSize: bufSize,
	}
}

func (r *ChatWorkerRegistry) getOrCreate(chatID int64) *chatWorker {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.workers[chatID]; ok {
		return w
	}
	w := newChatWorker(r.bufSize)
	r.workers[chatID] = w
	r.wg.Add(1)
	go w.loop(&r.wg)
	return w
}

// Run enqueues fn for chatID and blocks until it returns or ctx is cancelled.
func (r *ChatWorkerRegistry) Run(ctx context.Context, chatID int64, fn func(context.Context) error) error {
	if r.stopped.Load() {
		return ErrRegistryStopped
	}
	w := r.getOrCreate(chatID)
	j := job{ctx: ctx, fn: fn, result: make(chan error, 1)}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.ch <- j:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-j.result:
		return err
	}
}

// Shutdown stops all workers and waits for in-flight jobs to complete.
func (r *ChatWorkerRegistry) Shutdown(ctx context.Context) error {
	r.stopped.Store(true)
	r.mu.Lock()
	for _, w := range r.workers {
		close(w.done)
	}
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
