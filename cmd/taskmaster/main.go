package main

import (
	"fmt"
	"os"
	"taskmaster/internal/config"
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

	// print config
	// cfg.Print()

}
