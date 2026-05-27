package config

import (
	"gopkg.in/yaml.v3"
)

type Program struct {
	// Command to execute
	Cmd Cmd `yaml:"cmd" hint:"The full command line to run"`

	// Number of instances to keep running
	Numprocs Numprocs `yaml:"numprocs" hint:"Number of parallel instances"`

	// File mode creation mask
	Umask Umask `yaml:"umask" hint:"Permissions mask (e.g., 0022)"`

	// Working directory for the process
	Workingdir Workingdir `yaml:"workingdir" hint:"Path to the directory where process runs"`

	// Weather automatically start process just after config loaded
	Autostart Autostart `yaml:"autostart"`

	// Policy for restarting the process
	Autorestart Autorestart `yaml:"autorestart" hint:"Options: unexpected | always | never"`

	// List of valid exit codes for successful execution
	Exitcodes Exitcodes `yaml:"exitcodes" hint:"Codes considered 'successful'"`

	// Max retries on startup failure
	Startretries Startretries `yaml:"startretries" hint:"Max attempts to start before giving up"`

	// Time in seconds to wait for start confirmation
	Starttime Starttime `yaml:"starttime" hint:"Seconds to wait after starting"`

	// Signal to send for stopping the process
	Stopsignal Stopsignal `yaml:"stopsignal" hint:"Signal to send (e.g., SIGTERM)"`

	// Time in seconds to wait for graceful shutdown
	Stoptime Stoptime `yaml:"stoptime" hint:"Seconds to wait for graceful exit"`

	// Path to standard output log file
	Stdout Stdout `yaml:"stdout" hint:"File path for stdout redirection"`

	// Path to standard error log file
	Stderr Stderr `yaml:"stderr" hint:"File path for stderr redirection"`

	// Environment variables for the process
	Env Env `yaml:"env" hint:"Key-value pairs for environment"`
}

func (p Program) getFields() []Field {
	return []Field{
		&p.Cmd,
		&p.Numprocs,
		&p.Umask,
		&p.Workingdir,
		&p.Autostart,
		&p.Autorestart,
		&p.Exitcodes,
		&p.Startretries,
		&p.Starttime,
		&p.Stopsignal,
		&p.Stopsignal,
		&p.Stoptime,
		&p.Stdout,
		&p.Stderr,
		&p.Env,
	}
}

func (p *Program) setDefault() {
	fields := p.getFields()
	for _, field := range fields {
		field.setDefault()
	}
}

func (p Program) validate() error {
	fields := p.getFields()
	for _, field := range fields {
		err := field.validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Program) UnmarshalYAML(value *yaml.Node) error {
	p.setDefault()
	type Alias Program
	return value.Decode((*Alias)(p))
}
