package engine

import (
// "fmt"
)

// -----------------------------------------------

type event interface {
	handle() error
}

// -----------------------------------------------

type StartTimerFired struct {
	proc *Process
}

func (e StartTimerFired) handle() error {
	return nil
}

// -----------------------------------------------

type BackoffElapsed struct {
	proc *Process
}

func (e BackoffElapsed) handle() error {
	return nil
}

// -----------------------------------------------

type StopTimerFired struct {
	proc *Process
}

func (e StopTimerFired) handle() error {
	return nil
}

// -----------------------------------------------

type ProcExited struct {
	proc *Process
	err  error // from cmd.Wait() — nil if exit 0; *exec.ExitError otherwise
}

func (e ProcExited) handle() error {
	return nil
}

// -----------------------------------------------
