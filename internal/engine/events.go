package engine

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"
)

// -----------------------------------------------

type event interface {
	handle() error
}

// -----------------------------------------------

// event handles finished probe for a fresh process to consider it running
type StartTimerFired struct {
	proc *Process
}

func (e StartTimerFired) handle() error {
	e.proc.state = Running
	return nil
}

// -----------------------------------------------

// delay set to wait in backoff state before new spawn
type BackoffElapsed struct {
	proc *Process
}

func (e BackoffElapsed) handle() error {
	err := e.proc.parent.manager.spawner.spawn(e.proc)
	return err
}

// -----------------------------------------------

// time to wait before sending uncatchable kill signal
type StopTimerFired struct {
	proc *Process
}

func (e StopTimerFired) handle() error {
	p := e.proc
	err := p.cmd.Process.Kill()
	return err
}

// -----------------------------------------------

type ProcExited struct {
	proc   *Process
	status error // from cmd.Wait() — nil if exit 0; *exec.ExitError otherwise
}

func (e ProcExited) handle() error {
	var err error

	p := e.proc

	fatal, signal, code := parseProcessStatus(e.status)
	if fatal {
		p.state = Unknown
		// clean timers etc?
		return fmt.Errorf("process failed")
	}

	if signal {
		p.state = Exited
		// if killed by sigterm or p.signal requested
		return nil
	}

	// retry
	if p.state == Starting {
		handleRetry(e)
	}

	// autorestart
	if p.state == Running {
		err = handleRestart(e, code)
		if err != nil {
			return err
		}
	}

	log.Println("DEBUG: procexited")
	return err
}

// -----------------------------------------------

func handleRetry(e ProcExited) {
	p := e.proc
	if p.startTimer != nil {
		p.startTimer.Stop()
	}

	if p.retries < p.parent.startretries {
		p.state = Backoff
		p.retries += 1
		delay := time.Second * 5
		p.backoffTimer = time.AfterFunc(delay, func() {
			p.parent.manager.Events <- BackoffElapsed{proc: p}
		})
	} else {
		p.state = Fatal
	}
}

// -----------------------------------------------

func handleRestart(e ProcExited, code uint8) error {
	p := e.proc
	// change to switch
	if p.parent.autorestart == RestartAlways {
		err := p.parent.manager.spawner.spawn(p)
		return err
	} else if p.parent.autorestart == RestartNever {
		p.state = Exited
		return nil
	} else if p.parent.autorestart == RestartUnexpected {
		if !containsExitCode(p.parent.exitcodes, code) {
			err := p.parent.manager.spawner.spawn(p)
			return err
		} else {
			p.state = Exited
			return nil
		}
	}
	return nil
}

// fatal, signal, code
func parseProcessStatus(status error) (bool, bool, uint8) {

	if status == nil {
		// exitcode 0
		return false, false, 0
	}

	var exitErr *exec.ExitError
	if errors.As(status, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			// signal number in code slot
			return false, true, uint8(ws.Signal()) // signal triggered exit
		}

		code := exitErr.ExitCode()
		if code >= 1 && code <= 255 {
			// exitcode 1-255
			return false, false, uint8(code) // 1–255
		}
		return false, true, 0
	}

	// non-exit error
	return true, false, 0 // command never ran / non-exit error

}

func containsExitCode(slice []uint8, code uint8) bool {
	target := code

	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// -----------------------------------------------

type Status struct {
	Name string
}

func (e Status) handle() error {
	// print content
	return nil
}

// -----------------------------------------------

type Start struct {
	Name string
}

func (e Start) handle() error {
	// search process obj
	// spawn
	return nil
}

// -----------------------------------------------

type Stop struct {
	Name string
}

func (e Stop) handle() error {
	// sends stopsignal to a process
	// start timer for stoptime
	// set state STOPPING
	return nil
}

// -----------------------------------------------

type Restart struct {
	Name string
}

func (e Restart) handle() error {
	return nil
}

// -----------------------------------------------

type Reload struct {
	m   *Manager
	res chan struct{}
}

func (e Reload) handle() error {
	return nil
}

// -----------------------------------------------

type Shutdown struct {
	m   *Manager
	res chan struct{}
}

func (e Shutdown) handle() error {
	e.m.log.Debug("shutdown event received")
	// garefuly shutdown SIGTERM
	// inf 5s SIGKILL
	// close files?
	e.m.iter(despawnPrg)
	e.res <- struct{}{}
	return nil
}

// -----------------------------------------------
