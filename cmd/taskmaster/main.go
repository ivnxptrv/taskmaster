package main

import (
	"fmt"
	"os"
	"taskmaster/internal/config"
	"taskmaster/internal/engine"
)

func main() {

	// parse args
	args, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// load and init config
	loader := config.NewLoader(config.Options{})
	cfg, err := loader.Load(args.Filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	// validate config
	err = cfg.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(3)
	}

	// local daemon to recevive cmds to start processes
	s := engine.NewSpawner()

	// create manager
	m := engine.NewManager(s, cfg)
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

	err = m.Boot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(5)
	}

}
