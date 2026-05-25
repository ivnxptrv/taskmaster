package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Name string

type RestartPolicy string

const (
	RestartUnexpected RestartPolicy = "unexpected"
	RestartAlways     RestartPolicy = "always"
	RestartNever      RestartPolicy = "never"
)

type Signal string

const (
	SigTerm Signal = "TERM"
)

type Program struct {
	// Name of the process
	Name string `yaml:"name" hint:"Unique identifier for the process"`

	// Command to execute
	Cmd string `yaml:"cmd" hint:"The full command line to run"`

	// Number of instances to keep running
	Numprocs uint32 `yaml:"numprocs" hint:"Number of parallel instances"`

	// File mode creation mask
	Umask uint32 `yaml:"umask" hint:"Permissions mask (e.g., 0022)"`

	// Working directory for the process
	Workingdir string `yaml:"workingdir" hint:"Path to the directory where process runs"`

	// Deprecated: Use Autorestart instead
	Austostart bool `yaml:"austostart" hint:"Legacy flag"`

	// Policy for restarting the process
	Autorestart RestartPolicy `yaml:"autorestart" hint:"Options: unexpected | always | never"`

	// List of valid exit codes for successful execution
	Exitcodes []uint8 `yaml:"exitcodes" hint:"Codes considered 'successful'"`

	// Max retries on startup failure
	Startretries uint32 `yaml:"startretries" hint:"Max attempts to start before giving up"`

	// Time in seconds to wait for start confirmation
	Starttime uint32 `yaml:"starttime" hint:"Seconds to wait after starting"`

	// Signal to send for stopping the process
	Stopsignal Signal `yaml:"stopsignal" hint:"Signal to send (e.g., SIGTERM)"`

	// Time in seconds to wait for graceful shutdown
	Stoptime uint32 `yaml:"stoptime" hint:"Seconds to wait for graceful exit"`

	// Path to standard output log file
	Stdout string `yaml:"stdout" hint:"File path for stdout redirection"`

	// Path to standard error log file
	Stderr string `yaml:"stderr" hint:"File path for stderr redirection"`

	// Environment variables for the process
	Env map[string]string `yaml:"env" hint:"Key-value pairs for environment"`
}

type Config struct {
	Filepath string    // path to config file
	Programs []Program `yaml:"programs"`
}

func Load(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	config := &Config{Filepath: filepath}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Print() {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	encoder.Encode(c)
}
