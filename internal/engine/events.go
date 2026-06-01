package engine

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"syscall"
	"time"
)

// event is anything the Manager event loop can process. Handlers run on the
// single loop goroutine and MUST NOT send into m.events themselves. Follow-up
// work is a direct function call; future work uses time.AfterFunc whose
// callback runs in its own goroutine and CAN send.
//
// ─────────────────── State × event transition table ───────────────────
//
//                    StartCmd      StopCmd         RestartCmd        ProcExited                       StartTimerFired   StopTimerFired   BackoffElapsed
// NeverStarted/Stopped  spawn→Starting    no-op            spawn→Starting    n/a                              ignore            ignore           ignore
// Starting              no-op             stopProc→Stopping flag+stopProc    retry/Backoff or Fatal           →Running          ignore           ignore
// Backoff               no-op             cancelB→Stopped   cancelB+spawn    n/a                              ignore            ignore           spawn→Starting
// Running               no-op             stopProc→Stopping flag+stopProc    autorestart policy decides       ignore (stale)    ignore           ignore
// Stopping              flag (resp later) no-op             flag             →Stopped/Exited (or resp if flag)ignore (stale)    SIGKILL          ignore
// Exited/Fatal          spawn→Starting    no-op            spawn→Starting    n/a                              ignore            ignore           ignore
//
// "flag" = set proc.restartPending=true; ProcExited handler checks it first
// and respawns regardless of autorestart policy.
type event interface {
	handle(m *Manager)
}

// ────────────────────── Internal lifecycle events ──────────────────────

// ProcExited is sent by the wait-goroutine after cmd.Wait() returns.
// Status is the cmd.Wait() error: nil for exit 0, *exec.ExitError otherwise.
type ProcExited struct {
	Name   string
	Index  int
	Status error
}

func (e ProcExited) handle(m *Manager) {
	p, prg := m.lookup(e.Name, e.Index)
	if p == nil {
		m.log.Warn("ProcExited for unknown process", "program", e.Name, "index", e.Index)
		return
	}

	fatal, bySignal, code := parseProcessStatus(e.Status)
	prev := p.state
	pid := p.pid

	m.log.Info("process exited",
		"program", e.Name, "index", e.Index, "pid", pid,
		"from_state", prev.String(),
		"by_signal", bySignal, "code", code, "status", e.Status)

	// Reload requested this slot's removal — short-circuit everything else.
	if p.removeAfterExit {
		p.wipe()
		p.state = Stopped
		pruneAfterRemoval(m, prg, p)
		return
	}

	// Pending restart wins over everything except a fatal exec error.
	if p.restartPending && !fatal {
		p.wipe()
		if err := m.spawn(&prg.Spec, p); err != nil {
			m.log.Error("restart respawn failed", "program", e.Name, "index", e.Index, "err", err)
		} else {
			m.log.Info("restart respawned", "program", e.Name, "index", e.Index, "pid", p.pid)
		}
		return
	}

	// Exec itself failed (binary missing, etc.) — permanent.
	if fatal {
		p.wipe()
		p.state = Fatal
		return
	}

	// We requested this stop and the process complied (or got SIGKILL'd
	// after stoptime). Don't apply autorestart — that's the whole point of
	// the user-initiated stop.
	if prev == Stopping {
		p.wipe()
		p.state = Stopped
		return
	}

	// Death during Starting — whether by code or signal — is a failed start.
	// Retry until startretries, then give up.
	if prev == Starting {
		if p.startTimer != nil {
			p.startTimer.Stop()
			p.startTimer = nil
		}
		p.retries++
		if p.retries <= prg.Spec.Startretries {
			p.state = Backoff
			delay := backoffDelay(p.retries)
			m.log.Info("entering backoff",
				"program", e.Name, "index", e.Index,
				"retry", p.retries, "max_retries", prg.Spec.Startretries, "delay", delay)
			name, idx := e.Name, e.Index
			ctx := m.shutdownCtx
			p.backoffTimer = time.AfterFunc(delay, func() {
				select {
				case m.events <- BackoffElapsed{Name: name, Index: idx}:
				case <-ctx.Done():
				}
			})
			return
		}
		p.wipe()
		p.state = Fatal
		m.log.Warn("process gave up after retries",
			"program", e.Name, "index", e.Index, "retries", p.retries)
		return
	}

	// Exit while Running: apply autorestart policy. Signaled exits without
	// us asking (external SIGKILL etc.) are treated as "unexpected" — they
	// trigger respawn under always and under unexpected (signal codes are
	// not in spec.Exitcodes which is exit-code-only).
	if prev == Running {
		shouldRestart := false
		switch prg.Spec.Autorestart {
		case RestartAlways:
			shouldRestart = true
		case RestartNever:
			shouldRestart = false
		case RestartUnexpected:
			// Signaled exits are by definition unexpected from the policy's
			// point of view. Code-based exits are unexpected iff not in the
			// configured success-code set.
			if bySignal {
				shouldRestart = true
			} else {
				shouldRestart = !slices.Contains(prg.Spec.Exitcodes, code)
			}
		}
		p.wipe()
		if shouldRestart {
			if err := m.spawn(&prg.Spec, p); err != nil {
				m.log.Error("autorestart spawn failed", "program", e.Name, "index", e.Index, "err", err)
			}
		} else {
			p.state = Exited
		}
		return
	}

	// Fall-through: log and mark Exited so we don't lose track.
	p.wipe()
	p.state = Exited
}

// StartTimerFired promotes Starting → Running if still in Starting; otherwise stale.
type StartTimerFired struct {
	Name  string
	Index int
}

func (e StartTimerFired) handle(m *Manager) {
	p, _ := m.lookup(e.Name, e.Index)
	if p == nil || p.state != Starting {
		return
	}
	p.state = Running
	p.retries = 0
	m.log.Info("process running", "program", e.Name, "index", e.Index, "pid", p.pid)
}

// StopTimerFired sends SIGKILL if the process is still Stopping.
type StopTimerFired struct {
	Name  string
	Index int
}

func (e StopTimerFired) handle(m *Manager) {
	p, _ := m.lookup(e.Name, e.Index)
	if p == nil || p.state != Stopping {
		return
	}
	m.log.Warn("stoptime elapsed, sending SIGKILL",
		"program", e.Name, "index", e.Index, "pid", p.pid)
	if err := m.runtime.Signal(p.cmd, syscall.SIGKILL); err != nil {
		m.log.Error("SIGKILL failed", "program", e.Name, "index", e.Index, "err", err)
	}
}

// BackoffElapsed respawns the process after the backoff delay.
type BackoffElapsed struct {
	Name  string
	Index int
}

func (e BackoffElapsed) handle(m *Manager) {
	p, prg := m.lookup(e.Name, e.Index)
	if p == nil || p.state != Backoff {
		return // stale (e.g. user issued Stop while in backoff)
	}
	if err := m.spawn(&prg.Spec, p); err != nil {
		m.log.Error("backoff respawn failed", "program", e.Name, "index", e.Index, "err", err)
		p.state = Fatal
	}
}

// ─────────────────────── External command events ───────────────────────
//
// Each command carries a Reply chan: the public method on Manager (api.go)
// enqueues the command and blocks on Reply. Always reply on every exit path.

type startCmd struct {
	Name  string
	Index int  // -1 means "all processes for this program"
	Reply chan error
}

func (e startCmd) handle(m *Manager) {
	prg, ok := m.programs[e.Name]
	if !ok {
		e.Reply <- fmt.Errorf("unknown program %q", e.Name)
		return
	}
	targets := selectProcs(prg, e.Index)
	if targets == nil {
		e.Reply <- fmt.Errorf("process %q[%d] not found", e.Name, e.Index)
		return
	}
	var firstErr error
	for _, p := range targets {
		if !canStart(p.state) {
			continue
		}
		if err := m.spawn(&prg.Spec, p); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s[%d]: %w", e.Name, p.index, err)
		}
	}
	e.Reply <- firstErr
}

func canStart(s State) bool {
	switch s {
	case NeverStarted, Stopped, Exited, Fatal:
		return true
	}
	return false
}

type stopCmd struct {
	Name  string
	Index int
	Reply chan error
}

func (e stopCmd) handle(m *Manager) {
	prg, ok := m.programs[e.Name]
	if !ok {
		e.Reply <- fmt.Errorf("unknown program %q", e.Name)
		return
	}
	targets := selectProcs(prg, e.Index)
	if targets == nil {
		e.Reply <- fmt.Errorf("process %q[%d] not found", e.Name, e.Index)
		return
	}
	for _, p := range targets {
		stopProcess(m, &prg.Spec, p)
	}
	e.Reply <- nil
}

type restartCmd struct {
	Name  string
	Index int
	Reply chan error
}

func (e restartCmd) handle(m *Manager) {
	prg, ok := m.programs[e.Name]
	if !ok {
		e.Reply <- fmt.Errorf("unknown program %q", e.Name)
		return
	}
	targets := selectProcs(prg, e.Index)
	if targets == nil {
		e.Reply <- fmt.Errorf("process %q[%d] not found", e.Name, e.Index)
		return
	}
	var firstErr error
	for _, p := range targets {
		switch p.state {
		case Starting, Running:
			p.restartPending = true
			stopProcess(m, &prg.Spec, p)
		case Stopping:
			p.restartPending = true
		case Backoff:
			if p.backoffTimer != nil {
				p.backoffTimer.Stop()
				p.backoffTimer = nil
			}
			p.wipe()
			if err := m.spawn(&prg.Spec, p); err != nil && firstErr == nil {
				firstErr = err
			}
		case NeverStarted, Stopped, Exited, Fatal:
			if err := m.spawn(&prg.Spec, p); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	e.Reply <- firstErr
}

// ProcInfo is the snapshot returned by Status. Plain value type.
type ProcInfo struct {
	Name      string
	Index     int
	State     string
	Pid       int
	StartedAt time.Time
	Retries   int
}

type statusCmd struct {
	Reply chan []ProcInfo
}

func (e statusCmd) handle(m *Manager) {
	out := make([]ProcInfo, 0)
	for name, prg := range m.programs {
		for _, p := range prg.procs {
			out = append(out, ProcInfo{
				Name:      name,
				Index:     p.index,
				State:     p.state.String(),
				Pid:       p.pid,
				StartedAt: p.startedAt,
				Retries:   p.retries,
			})
		}
	}
	e.Reply <- out
}

// shutdownCmd signals every live process and cancels the Manager's
// shutdown context once everyone has reported exit (or the deadline hits).
type shutdownCmd struct {
	Reply chan struct{}
}

func (e shutdownCmd) handle(m *Manager) {
	m.log.Info("shutdown requested")
	// signalExit makes serve() return; Run's gracefulDrain then sends
	// SIGTERM, waits for ProcExited drainage, and SIGKILLs stragglers.
	m.signalExit()
	e.Reply <- struct{}{}
}

// ───────────────────────────── helpers ─────────────────────────────

func isLive(s State) bool {
	switch s {
	case Starting, Running, Stopping, Backoff:
		return true
	}
	return false
}

// selectProcs returns the slice of processes a command targets.
// index == -1 means "all processes of the program".
func selectProcs(prg *Program, index int) []*Process {
	if index < 0 {
		return prg.procs
	}
	p := prg.process(index)
	if p == nil {
		return nil
	}
	return []*Process{p}
}

// backoffDelay grows quickly to cap restart storms but stays small enough
// for tests to advance through with a brief sleep. Phase 9 can tune.
func backoffDelay(retries int) time.Duration {
	d := time.Duration(retries) * time.Second
	if d < time.Second {
		d = time.Second
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// parseProcessStatus dissects the error returned by cmd.Wait().
// Returns:
//
//	fatal    — exec itself failed (binary not found, etc.); process never
//	           really ran. State should go to Fatal.
//	bySignal — child died from a signal; "code" is the signal number.
//	code     — exit code (1-255) for normal exits; signal number for signaled exits.
func parseProcessStatus(status error) (fatal, bySignal bool, code int) {
	if status == nil {
		return false, false, 0
	}
	var exitErr *exec.ExitError
	if errors.As(status, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return false, true, int(ws.Signal())
		}
		ec := exitErr.ExitCode()
		if ec >= 1 && ec <= 255 {
			return false, false, ec
		}
		return false, true, 0
	}
	// Anything else from Wait — e.g. fork/exec failure — is fatal.
	return true, false, 0
}
