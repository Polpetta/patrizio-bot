// Package main is the entrypoint for the Patrizio Delta Chat bot.
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/polpetta/patrizio/internal/bot"
	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// #nosec G301 - Directory needs to be readable by deltachat-rpc-server
	if err := os.MkdirAll(cfg.MediaPath(), 0755); err != nil {
		panic(err)
	}

	dbDir := filepath.Dir(cfg.DBPath())
	// #nosec G301 - Directory needs to be readable by deltachat-rpc-server
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panic(err)
	}

	db, err := bot.InitDatabase(cfg, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	deps, chatExec := bot.BuildDependencies(cfg, db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*1e9)
		defer cancel()
		_ = chatExec.Shutdown(shutdownCtx) //nolint:errcheck // best-effort shutdown in signal handler
	}()

	cli := bot.Setup(deps)

	if err := cli.Start(); err != nil {
		cli.Logger.Error(err)
		os.Exit(1)
	}
}
