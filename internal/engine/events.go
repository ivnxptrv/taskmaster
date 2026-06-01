package engine

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
	"syscall"
	"taskmaster/internal/config"
	"time"
)

// -----------------------------------------------

type event interface {
	handle(m *Manager) error
}

// -----------------------------------------------

// event handles finished probe for a fresh process to consider it running
// state = Running
type StartTimerFired struct {
	Name  string
	Index int
}

func (e StartTimerFired) handle(m *Manager) error {
	proc, ok := m.getProcess(e.Name, e.Index)
	if !ok {
		m.log.Debug("proc not found", "event", "StartTimerFired")
	}
	proc.state = Running
	return nil
}

// -----------------------------------------------

// delay set to wait in backoff state before new spawn
// state = not changed
type BackoffElapsed struct {
	Name  string
	Index int
}

func (e BackoffElapsed) handle(m *Manager) error {
	prg, ok := m.getProgram(e.Name)
	if !ok {
		m.log.Debug("program not found", "event", "BackoffElapsed")
	}
	proc, ok := prg.getProcess(e.Index)
	if !ok {
		m.log.Debug("proc not found", "event", "BackoffElapsed")
	}
	err := m.spawn(prg.spec, proc)
	proc.state = Starting

	return err
}

// -----------------------------------------------

// time to wait before sending uncatchable kill signal
// state = Stopped
type StopTimerFired struct {
	Name  string
	Index int
}

func (e StopTimerFired) handle(m *Manager) error {
	proc, ok := m.getProcess(e.Name, e.Index)
	if !ok {
		m.log.Debug("program not found", "event", "StopTimerFired")
	}
	err := m.runtime.Signal(proc.cmd, syscall.SIGKILL)
	if err != nil {
		m.log.Error("sigkill failed", "event", "StopTimerFired")
	}

	proc.wipe()
	proc.state = Stopped
	return err
}

// -----------------------------------------------

type ProcExited struct {
	Name   string
	Index  int
	Status error // from cmd.Wait() — nil if exit 0; *exec.ExitError otherwise
}

func (e ProcExited) handle(m *Manager) error {
	var err error

	prg, ok := m.getProgram(e.Name)
	if !ok {
		m.log.Debug("program not found", "event", "ProcExited")
	}

	proc, ok := prg.getProcess(e.Index)
	if !ok {
		m.log.Debug("process not found", "event", "ProcExited")
	}

	fatal, signal, code := parseProcessStatus(e.Status)
	if fatal {
		proc.state = Unknown
		proc.wipe()
		return fmt.Errorf("process failed")
	}

	// restart requested
	if proc.restartPending {
		proc.wipe()
		proc.state = Starting
		return m.spawn(prg.spec, proc)
	}

	// exited because of signal / Stopping
	if signal {
		signal := syscall.Signal(code)
		if signal == prg.spec.stopsignal {
			proc.state = Stopped
		} else {
			proc.state = Exited
		}
		proc.wipe()
		return nil
	}

	// retry
	if proc.state == Starting {
		if proc.startTimer != nil {
			proc.startTimer.Stop()
		}
		if proc.retries < prg.spec.startretries {
			proc.state = Backoff
			proc.retries += 1
			delay := time.Second * 5
			proc.backoffTimer = time.AfterFunc(delay, func() {
				m.events <- BackoffElapsed{Name: e.Name, Index: e.Index}
			})
		} else {
			proc.state = Fatal
			proc.wipe()
		}
	}

	// autorestart
	if proc.state == Running {
		switch prg.spec.autorestart {
		case RestartAlways:
			proc.wipe()
			proc.state = Starting
			return m.spawn(prg.spec, proc)

		case RestartNever:
			proc.wipe()
			proc.state = Exited
			return nil

		case RestartUnexpected:
			if !slices.Contains(prg.spec.exitcodes, code) {
				proc.wipe()
				proc.state = Starting
				return m.spawn(prg.spec, proc)
			}
			proc.wipe()
			proc.state = Exited
			return nil

		default:
			return nil
		}
	}

	return err
}

// -----------------------------------------------

// fatal, signal, code
func parseProcessStatus(status error) (bool, bool, int) {

	if status == nil {
		// exitcode 0
		return false, false, 0
	}

	var exitErr *exec.ExitError
	if errors.As(status, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			// signal number in code slot
			return false, true, int(ws.Signal()) // signal triggered exit
		}

		code := exitErr.ExitCode()
		if code >= 1 && code <= 255 {
			// exitcode 1-255
			return false, false, code // 1–255
		}
		return false, true, 0
	}

	// non-exit error
	return true, false, 0 // command never ran / non-exit error

}

// -----------------------------------------------

type Status struct {
	Name  string
	Index int
	Reply chan<- ProcInfo
}

func (e Status) handle(m *Manager) error {
	proc, ok := m.getProcess(e.Name, e.Index)
	if !ok {
		e.Reply <- ProcInfo{}
		return fmt.Errorf("process not found")
	}
	e.Reply <- ProcInfo{
		name:      e.Name,
		index:     e.Index,
		state:     string(proc.state),
		pid:       proc.pid,
		startedAt: proc.startedAt,
	}
	return nil
}

// -----------------------------------------------

type Start struct {
	Name  string
	Index int
	Reply chan struct{}
}

func (e Start) handle(m *Manager) error {
	prg, ok := m.getProgram(e.Name)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("program not found")
	}
	proc, ok := prg.getProcess(e.Index)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("process not found")
	}
	if proc.state == Starting || proc.state == Backoff || proc.state == Running || proc.state == Stopping {
		e.Reply <- struct{}{}
		m.log.Debug("process can not be started")
		return nil
	}
	err := m.spawn(prg.spec, proc)
	if err != nil {
		e.Reply <- struct{}{}
		return err
	}
	proc.state = Starting
	e.Reply <- struct{}{}
	return nil
}

// -----------------------------------------------

type Stop struct {
	Name  string
	Index int
	Reply chan struct{}
}

func (e Stop) handle(m *Manager) error {
	prg, ok := m.getProgram(e.Name)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("program not found")
	}
	proc, ok := prg.getProcess(e.Index)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("process not found")
	}

	err := m.runtime.Signal(proc.cmd, prg.spec.stopsignal)
	if err != nil {
		return fmt.Errorf("stop event: %w", err)
	}

	proc.stopTimer = time.AfterFunc(prg.spec.starttime, func() {
		m.events <- StopTimerFired{Name: prg.spec.name, Index: proc.index}
	})

	proc.state = Stopping
	e.Reply <- struct{}{}
	return nil
}

// -----------------------------------------------

// send stop event and change state to be restarted on exit event
// state = Restarting
type Restart struct {
	Name  string
	Index int
	Reply chan struct{}
}

func (e Restart) handle(m *Manager) error {
	// send stop event to event queue
	// change state to restart to be restarted at exit
	prg, ok := m.getProgram(e.Name)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("program not found")
	}
	proc, ok := prg.getProcess(e.Index)
	if !ok {
		e.Reply <- struct{}{}
		return fmt.Errorf("process not found")
	}
	proc.restartPending = true
	m.events <- Stop{Name: prg.spec.name, Index: proc.index, Reply: make(chan struct{})}

	return nil
}

// -----------------------------------------------

type Reload struct {
	reply chan struct{}
}

func (e Reload) handle(m *Manager) error {
	// loader := config.NewLoader(config.Options{})
	// cfg, err := loader.Load(e.m.cfg.Filepath)
	// if err != nil {
	// 	// log.Error("failed to load config", "error", err)
	// }

	return nil
}

// -----------------------------------------------

type Shutdown struct {
	Reply chan struct{}
}

func (e Shutdown) handle(m *Manager) error {
	m.log.Debug("shutdown event received")
	for _, prg := range m.programs {
		for _, proc := range prg.procs {
			if proc.state != NeverStarted || proc.state != Stopped || proc.state != Stopping || proc.state != Exited || proc.state != Fatal || proc.state != Unknown {
				err := m.runtime.Signal(proc.cmd, syscall.SIGKILL)
				if err != nil {
					m.log.Error("sigkill failed")
				}
				proc.wipe()
				proc.state = Stopped
			}
		}
	}
	e.Reply <- struct{}{}
	return nil
}

// -----------------------------------------------
