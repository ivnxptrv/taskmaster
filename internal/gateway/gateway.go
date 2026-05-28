package gateway

import (
	"time"
)

// Main interface to communicate with Engine

// -----------------------------------------------

type Status struct {
	state  string
	pid    int
	uptime time.Time
}

type Gateway interface {
	Status(name string) Status
	Start(name string) error
	Stop(name string) error
	Restart(name string) error

	Reload() error
	Shutdown() error
}

// -----------------------------------------------
