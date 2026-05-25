package main

import (
	"fmt"
	"os"
	// "taskmaster/internal/config"
)

func main() {
	args, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(args.Filepath)

	// config, err := config.Load(args.Filepath)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	// 	os.Exit(1)
	// }

	// config
}
