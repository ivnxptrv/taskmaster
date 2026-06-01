package engine

import (
	"time"
)

// stopProcess implements graceful stop: send Spec.Stopsignal, schedule a
// StopTimerFired event after Spec.Stoptime, set state to Stopping.
// No-op if the process is not currently live.
//
// Free function (not a method) because it spans Manager, Spec, and Process
// without belonging to any of them. Called by stopCmd, restartCmd, and
// shutdownCmd handlers — all of which run on the event-loop goroutine.
func stopProcess(m *Manager, spec *Spec, p *Process) {
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}
	if !isLive(p.state) {
		return
	}

	if err := m.runtime.Signal(p.cmd, spec.Stopsignal); err != nil {
		m.log.Error("stopsignal failed",
			"program", spec.Name, "index", p.index, "pid", p.pid, "err", err)
		return
	}
	p.state = Stopping
	m.log.Info("stopsignal sent",
		"program", spec.Name, "index", p.index, "pid", p.pid,
		"signal", spec.Stopsignal.String())

	// Schedule SIGKILL escalation.
	name, idx := spec.Name, p.index
	ctx := m.shutdownCtx
	delay := spec.Stoptime
	if delay <= 0 {
		delay = 5 * time.Second
	}
	p.stopTimer = time.AfterFunc(delay, func() {
		select {
		case m.events <- StopTimerFired{Name: name, Index: idx}:
		case <-ctx.Done():
		}
	})
}
