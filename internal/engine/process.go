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
	parent *Program
	log    *slog.Logger

	index   uint8
	state   State
	retries uint16

	cmd *exec.Cmd

	pid          int
	startedAt    *time.Time
	startTimer   *time.Timer
	backoffTimer *time.Timer
	stopTimer    *time.Timer
	stdoutFile   *os.File
	stderrFile   *os.File
}

func newProcess(prg *Program, index uint8, log *slog.Logger) *Process {
	return &Process{
		parent:       prg,
		log:          log,
		index:        index,
		state:        Stopped,
		retries:      0,
		pid:          -1,
		cmd:          nil,
		startedAt:    nil,
		startTimer:   nil,
		backoffTimer: nil,
		stopTimer:    nil,
		stdoutFile:   nil,
		stderrFile:   nil,
	}
}

func setStarting(p *Process) error {
	p.state = Starting
	return nil
}

func despawnP(p *Process) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil // Process isn't running
	}

	p.log.Info("sending SIGTERM to process", "pid", p.pid)
	err := p.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	// Schedule SIGKILL after 5 seconds
	p.stopTimer = time.AfterFunc(5*time.Second, func() {
		p.log.Warn("process did not exit, sending SIGKILL", "pid", p.pid)
		_ = p.cmd.Process.Kill()
	})

	return nil
}
