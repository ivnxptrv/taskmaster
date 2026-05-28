package engine

import (
	"fmt"
)

// -----------------------------------------------

type command interface {
	handle() error
}

// -----------------------------------------------

type Status struct {
	name string
}

func (c *Status) handle() error {
	// print content
	return nil
}

// -----------------------------------------------

type Start struct {
	name string
}

func (c *Start) handle() error {
	// search process obj
	// spawn
	return nil
}

// -----------------------------------------------

type Stop struct {
	name string
}

func (c *Stop) handle() error {
	// sends stopsignal to a process
	// start timer for stoptime
	// set state STOPPING
	return nil
}

// -----------------------------------------------

type Restart struct {
	name string
}

func (c *Restart) handle() error {
	return nil
}

// -----------------------------------------------

type Reload struct{}

func (c *Reload) handle() error {
	return nil
}

// -----------------------------------------------

type Shutdown struct{}

func (c *Shutdown) handle() error {
	return nil
}

// -----------------------------------------------
