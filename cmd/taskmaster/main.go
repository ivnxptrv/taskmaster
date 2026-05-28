package main

import (
	"fmt"
	"os"
	"taskmaster/internal/client"
	"taskmaster/internal/config"
	"taskmaster/internal/engine"
	"taskmaster/internal/logger"
)

func main() {

	// init logger
	log, err := logger.NewDefaultLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// parse args
	args, err := parseArgs()
	if err != nil {
		log.Error("failed to parse arguments", "error", err)
		os.Exit(1)
	}

	// load and init config
	loader := config.NewLoader(config.Options{})
	cfg, err := loader.Load(args.Filepath)
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(2)
	}
	// validate config
	err = cfg.Validate()
	if err != nil {
		log.Error("failed to validate config", "error", err)
		os.Exit(3)
	}

	// local daemon to recevive cmds to start processes
	s := engine.NewSpawner(log)

	// create manager
	m := engine.NewManager(s, cfg, log)
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
