package manager

import (
	"fmt"
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

	cmd *exec.Cmd

	index      uint32
	state      State
	pid        int
	StartedAt  *time.Time
	Retries    int
	StartTimer *time.Timer
	StopTimer  *time.Timer

	stdoutFile *os.File
	stderrFile *os.File
}

func spawnProcess(prg *Program) (*Process, error) {
	return &Process
}
