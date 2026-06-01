package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"taskmaster/internal/client"
	"taskmaster/internal/engine"
	"taskmaster/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taskmaster:", err)
		os.Exit(1)
	}
}

func run() error {
	args, err := parseArgs()
	if err != nil {
		return fmt.Errorf("parse args: %w", err)
	}

	log, err := logger.NewDefaultLogger()
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	specs, err := loadSpecs(args.Filepath)
	if err != nil {
		return err
	}

	runtime := engine.NewOsRuntime(log)
	mgr, err := engine.NewManager(specs, runtime, log)
	if err != nil {
		return fmt.Errorf("build manager: %w", err)
	}

	backend := &engineBackend{m: mgr, cfgPath: args.Filepath}

	// Root ctx cancels on SIGINT/SIGTERM. After stop() is called subsequent
	// signals fall through to Go's default (instant exit) — the escape
	// hatch for "graceful shutdown hangs".
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go watchReload(ctx, backend, log)

	engineErr := make(chan error, 1)
	go func() { engineErr <- mgr.Run(ctx) }()

	shell := client.NewShell(backend, log)
	shellErr := shell.Run(ctx)

	stop()         // release signal handlers + cancel ctx
	mgr.Shutdown() // idempotent: ensures serve() exits if not already
	eErr := <-engineErr

	if eErr != nil && !errors.Is(eErr, context.Canceled) && !errors.Is(eErr, context.DeadlineExceeded) {
		return errors.Join(shellErr, eErr)
	}
	log.Info("taskmaster exiting cleanly")
	return shellErr
}

// watchReload listens for SIGHUP and triggers a config reload through the
// backend (which re-reads the file). SIGHUP is intentionally separate from
// shutdown signals — it's a "re-read config" instruction, not "stop".
func watchReload(ctx context.Context, backend *engineBackend, log *slog.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	defer signal.Stop(ch)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			log.Info("SIGHUP received; reloading")
			if err := backend.Reload(); err != nil {
				log.Error("reload failed", "err", err)
			}
		}
	}
}
