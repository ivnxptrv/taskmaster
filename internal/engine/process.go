package engine

import (
	"os/exec"
	"time"
)

// State is the lifecycle state of a supervised process.
//
// Phase 1 only uses Stopped, Starting, Running, Exited.
// The full set (Backoff, Stopping, Fatal, Unknown) lands in Phase 3 with
// autorestart / stop / restart semantics.
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
	default:
		return "Unknown"
	}
}

// Process is the *observed* runtime state of one supervised child process.
// All fields are owned by the Manager event loop and must only be mutated
// from a handler running on that loop.
type Process struct {
	index int
	state State

	// retries counts failed start attempts (Starting → Exited before Running).
	// Resets to 0 on a successful start.
	retries int

	// restartPending records that a user-issued Restart should respawn
	// this process when the current instance exits. Phase 3.
	restartPending bool

	// removeAfterExit is set by reload when this slot is being deleted
	// (program removed, or numprocs shrunk). ProcExited won't respawn it
	// and the slot is dropped from the parent Program's procs slice.
	removeAfterExit bool

	cmd       *exec.Cmd
	pid       int
	startedAt time.Time

	startTimer   *time.Timer
	backoffTimer *time.Timer
	stopTimer    *time.Timer
}

func newProcess(index int) *Process {
	return &Process{index: index, state: NeverStarted, pid: -1}
}

// wipe resets transient runtime fields after a process exits. It does NOT
// touch state — the caller decides which terminal state to set (Exited,
// Stopped, Fatal). Timers are stopped best-effort; handlers must defend
// against stale events that fired before Stop took effect.
func (p *Process) wipe() {
	if p.startTimer != nil {
		p.startTimer.Stop()
	}
	if p.backoffTimer != nil {
		p.backoffTimer.Stop()
	}
	if p.stopTimer != nil {
		p.stopTimer.Stop()
	}
	p.startTimer = nil
	p.backoffTimer = nil
	p.stopTimer = nil
	p.cmd = nil
	p.pid = -1
	p.startedAt = time.Time{}
	p.restartPending = false
	// removeAfterExit is intentionally NOT cleared by wipe — ProcExited
	// reads it AFTER wipe to decide whether to drop the slot.
}
