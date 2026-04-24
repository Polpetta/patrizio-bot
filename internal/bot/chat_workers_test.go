package bot

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChatWorkerRegistry_SameChatSerializes(t *testing.T) {
	r := NewChatWorkerRegistry(4)
	ctx := context.Background()

	var order []int
	var mu sync.Mutex
	var inFlight atomic.Int32

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Run(ctx, 42, func(_ context.Context) error {
				if n := inFlight.Add(1); n > 1 {
					t.Errorf("concurrent execution detected: %d jobs in flight at once", n)
				}
				mu.Lock()
				order = append(order, i)
				mu.Unlock()
				inFlight.Add(-1)
				return nil
			})
		}()
	}
	wg.Wait()

	if len(order) != 3 {
		t.Errorf("expected 3 jobs executed, got %d: %v", len(order), order)
	}
}

func TestChatWorkerRegistry_DifferentChatsParallel(t *testing.T) {
	r := NewChatWorkerRegistry(4)
	ctx := context.Background()

	var started atomic.Int32
	barrier := make(chan struct{})

	var wg sync.WaitGroup
	for chatID := int64(1); chatID <= 3; chatID++ {
		chatID := chatID
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Run(ctx, chatID, func(_ context.Context) error {
				started.Add(1)
				<-barrier
				return nil
			})
		}()
	}

	// Wait until all 3 workers are executing simultaneously
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if started.Load() == 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if started.Load() != 3 {
		close(barrier)
		t.Fatalf("expected 3 parallel workers, only %d started", started.Load())
	}
	close(barrier)
	wg.Wait()
}

func TestChatWorkerRegistry_ContextCancelWhileQueued(t *testing.T) {
	r := NewChatWorkerRegistry(1)

	// Block the worker so the second job stays queued.
	block := make(chan struct{})
	bgCtx := context.Background()
	go func() {
		_ = r.Run(bgCtx, 1, func(_ context.Context) error {
			<-block
			return nil
		})
	}()
	time.Sleep(10 * time.Millisecond) // give the first job time to start

	cancelCtx, cancel := context.WithCancel(context.Background())
	var executed atomic.Bool

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(cancelCtx, 1, func(_ context.Context) error {
			executed.Store(true)
			return nil
		})
	}()
	cancel()
	close(block)

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	time.Sleep(10 * time.Millisecond) // allow any incorrectly-retained job to run
	if executed.Load() {
		t.Error("cancelled queued job must not execute")
	}
}

func TestChatWorkerRegistry_PanicInFnReturnsError(t *testing.T) {
	r := NewChatWorkerRegistry(4)
	ctx := context.Background()

	err := r.Run(ctx, 1, func(_ context.Context) error {
		panic("deliberate panic")
	})
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}

	// Worker must still process next job after the panic.
	err = r.Run(ctx, 1, func(_ context.Context) error { return nil })
	if err != nil {
		t.Errorf("expected nil from next job after panic, got %v", err)
	}
}

func TestChatWorkerRegistry_ShutdownWaitsForInFlight(t *testing.T) {
	r := NewChatWorkerRegistry(4)
	ctx := context.Background()

	block := make(chan struct{})
	var done atomic.Bool

	go func() {
		_ = r.Run(ctx, 1, func(_ context.Context) error {
			<-block
			done.Store(true)
			return nil
		})
	}()
	time.Sleep(10 * time.Millisecond) // ensure job is running

	shutdownCh := make(chan error, 1)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownCh <- r.Shutdown(shutdownCtx)
	}()

	time.Sleep(20 * time.Millisecond)
	close(block)

	if err := <-shutdownCh; err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if !done.Load() {
		t.Error("in-flight job did not complete before Shutdown returned")
	}
}

func TestChatWorkerRegistry_RunAfterShutdown(t *testing.T) {
	r := NewChatWorkerRegistry(4)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = r.Shutdown(shutdownCtx)

	err := r.Run(context.Background(), 1, func(_ context.Context) error { return nil })
	if !errors.Is(err, ErrRegistryStopped) {
		t.Errorf("expected ErrRegistryStopped, got %v", err)
	}
}
