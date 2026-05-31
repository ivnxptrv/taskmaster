package main

import (
	"fmt"
	"os"
	"taskmaster/internal/client"
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
	m := engine.NewManager(cfg, runtime, log)
	// validate manager
	err = m.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(4)
	}

	// cfg.Print()
	// m.Print()
	// print config
	// cfg.Print()

	// autostrat and listen for events
	err = m.Boot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(5)
	}

	// create entity to send events to manager
	f := engine.NewForeman(m)

	shell := client.NewShell(f)
	// if shell == nil {
	// }
	// w := NewSignalWatcher(f)
	// go w.listen()

	// enter shell
	shell.Enter()

}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taskmaster: ", err)
		os.Exit(1)
	}
}
