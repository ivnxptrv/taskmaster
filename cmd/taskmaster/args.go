package main

import (
	"flag"
	"fmt"
	"os"
)

type Args struct {
	Filepath string
}

func argExists(arg string, msg string) (bool, error) {
	if arg == "" {
		return false, fmt.Errorf("the -c flag is required")
	}
	return true, nil
}

func fileExists(filepath string) (bool, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, fmt.Errorf("file does not exist: %s", filepath)
	}
	return true, nil
}

func fileReadable(filepath string) (bool, error) {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		return false, fmt.Errorf("cannot read file: %s (permission denied)", filepath)
	}
	file.Close()
	return true, nil
}

func filepathValid(filepath string) (bool, error) {
	if _, err := argExists(filepath, "the -c flag is required"); err != nil {
		return false, err
	}

	if _, err := fileExists(filepath); err != nil {
		return false, err
	}

	if _, err := fileReadable(filepath); err != nil {
		return false, err
	}
	return true, nil
}

func argsValid(args Args) (bool, error) {
	if res, err := filepathValid(args.Filepath); res != true {
		return false, err
	}
	return true, nil
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
	res, err := argsValid(args)
	if res != true {
		return Args{}, err
	}
	return args, nil
}
