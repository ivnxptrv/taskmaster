package main

import (
	"flag"
	"fmt"
	"taskmaster/internal/fileutil"
)

type Args struct {
	Filepath string
}

func argExists(arg string, msg string) error {
	if arg == "" {
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func filepathArgValid(filepath string) error {
	if err := argExists(filepath, "the -c flag is required"); err != nil {
		return err
	}

	if err := fileutil.FileExists(filepath); err != nil {
		return err
	}

	if err := fileutil.FileReadable(filepath); err != nil {
		return err
	}
	return nil
}

func argsValid(args Args) error {
	if err := filepathArgValid(args.Filepath); err != nil {
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
