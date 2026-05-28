package engine

import (
	"os"
	"os/exec"
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
	Uknown
)

// -----------------------------------------------

type Process struct {
	parent *Program

	index   uint8
	state   State
	retries uint16

	cmd *exec.Cmd

	pid        int
	startedAt  *time.Time
	startTimer *time.Timer
	stopTimer  *time.Timer
	stdoutFile *os.File
	stderrFile *os.File
}

func newProcess(prg *Program, index uint8) *Process {
	return &Process{
		parent:     prg,
		index:      index,
		state:      Stopped,
		retries:    0,
		pid:        -1,
		cmd:        nil,
		startedAt:  nil,
		stopTimer:  nil,
		stdoutFile: nil,
		stderrFile: nil,
	}
}
