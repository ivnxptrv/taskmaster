package engine

import (
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// -----------------------------------------------

type State uint8

const (
	Stopped State = iota
	Starting
	Backoff
	Running
	Stopping
	Exited
	Fatal
	Unknown
)

// -----------------------------------------------

type Process struct {
	index   uint8
	state   State
	retries uint16

	cmd *exec.Cmd

	pid          int
	startedAt    time.Time
	startTimer   *time.Timer
	backoffTimer *time.Timer
	stopTimer    *time.Timer
	stdoutFile   *os.File
	stderrFile   *os.File
}

func newProcess(index uint8) *Process {
	return &Process{
		index:        index,
		state:        Stopped,
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
