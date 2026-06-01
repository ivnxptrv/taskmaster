package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"taskmaster/internal/config"
	"taskmaster/internal/engine"
	"taskmaster/internal/logger"
)

func run() error {

	// parse args
	args, err := parseArgs()
	if err != nil {
		fmt.Errorf("failed to parse arguments: %w", err)
	}

	// init logger
	log, err := logger.NewDefaultLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// load and init config
	loader := config.NewLoader(config.Options{})
	cfg, err := loader.Load(args.Filepath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// create runtime
	runtime := engine.NewOsRuntime(log.With("runtime"))

	// create manager
	m, err := engine.NewManager(cfg, runtime, log)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// create context with signals attached
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// autostart and listen for events
	err = m.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// shell := client.NewShell(m)
	// // if shell == nil {
	// // }
	// // w := NewSignalWatcher(f)
	// // go w.listen()

	// // enter shell
	// shell.Enter()
	stop()

}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taskmaster:", err)
		os.Exit(1)
	}
}
