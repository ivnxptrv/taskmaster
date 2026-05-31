package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// -----------------------------------------------

type State int

const (
	NeverStarted State = iota
	Stopped
	Starting
	Backoff
	Running
	Stopping
	Exited
	Fatal
	Unknown
)

func (s State) String() string {
	switch s {
	case NeverStarted:
		return "NeverStarted"
	case Stopped:
		return "Stopped"
	case Starting:
		return "Starting"
	case Backoff:
		return "Backoff"
	case Running:
		return "Running"
	case Stopping:
		return "Stopping"
	case Exited:
		return "Exited"
	case Fatal:
		return "Fatal"
	case Unknown:
		return "Unknown"
	default:
		return "UnknownState"
	}
}

// -----------------------------------------------

type Process struct {
	index   int
	state   State
	retries int

	cmd *exec.Cmd

	pid          int
	startedAt    time.Time
	startTimer   *time.Timer
	backoffTimer *time.Timer
	stopTimer    *time.Timer
	stdoutFile   *os.File
	stderrFile   *os.File
}

func newProcess(index int) *Process {
	return &Process{
		index:        index,
		state:        NeverStarted,
		retries:      0,
		pid:          -1,
		cmd:          nil,
		startedAt:    time.Time{},
		startTimer:   nil,
		backoffTimer: nil,
		stopTimer:    nil,
		stdoutFile:   nil,
		stderrFile:   nil,
	}
}

func (proc *Process) wipe() {
	if proc.startTimer != nil {
		proc.startTimer.Stop()
	}
	if proc.backoffTimer != nil {
		proc.backoffTimer.Stop()
	}
	if proc.stopTimer != nil {
		proc.stopTimer.Stop()
	}

	proc.startTimer = nil
	proc.backoffTimer = nil
	proc.stopTimer = nil

	proc.cmd = nil
	proc.pid = -1
	proc.startedAt = time.Time{}
	proc.retries = 0
	proc.stdoutFile = nil
	proc.stderrFile = nil
}
