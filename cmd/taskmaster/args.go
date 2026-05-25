package main

import (
	"flag"
	"fmt"
	"os"
)

type Args struct {
	Filepath string
}

func argExists(arg string, msg string) error {
	if arg == "" {
		return fmt.Errorf("the -c flag is required")
	}
	return nil
}

func fileExists(filepath string) error {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filepath)
	}
	return nil
}

func fileReadable(filepath string) error {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot read file: %s (permission denied)", filepath)
	}
	file.Close()
	return nil
}

func filepathValid(filepath string) error {
	if err := argExists(filepath, "the -c flag is required"); err != nil {
		return err
	}

	if err := fileExists(filepath); err != nil {
		return err
	}

	if err := fileReadable(filepath); err != nil {
		return err
	}
	return nil
}

func argsValid(args Args) error {
	if err := filepathValid(args.Filepath); err != nil {
		return err
	}
	return nil
}

func initArgs() (Args, error) {
	filepath := flag.String("c", "", "path to config file")
	flag.Parse()

	return Args{Filepath: *filepath}, nil
}

func parseArgs() (Args, error) {
	args, err := initArgs()
	if err != nil {
		return Args{}, err
	}
	err = argsValid(args)
	if err != nil {
		return Args{}, err
	}
	return args, nil
}
