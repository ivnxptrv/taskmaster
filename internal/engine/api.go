package engine

import "errors"

// ErrEngineStopped is returned by API methods when the Manager has exited
// its event loop. Callers should treat this as "no longer available".
var ErrEngineStopped = errors.New("engine stopped")

// Public API surface used by clients (shell, signal watchers, future
// network server). Each method:
//
//   1. Builds a command event with a buffered reply channel.
//   2. Enqueues it on m.events guarded by m.shutdownCtx.Done().
//   3. Blocks on the reply, also guarded by m.shutdownCtx.Done().
//
// After Manager.Run returns, m.shutdownCtx is cancelled and every public
// method returns ErrEngineStopped instead of deadlocking.
//
// Index == -1 means "all processes of this program" for Start/Stop/Restart.

// Start spawns the targeted process(es) if currently not running.
func (m *Manager) Start(name string, index int) error {
	reply := make(chan error, 1)
	if err := m.submit(startCmd{Name: name, Index: index, Reply: reply}); err != nil {
		return err
	}
	return m.awaitErr(reply)
}

// Stop sends Spec.Stopsignal to the targeted process(es) and schedules
// SIGKILL escalation after Spec.Stoptime.
func (m *Manager) Stop(name string, index int) error {
	reply := make(chan error, 1)
	if err := m.submit(stopCmd{Name: name, Index: index, Reply: reply}); err != nil {
		return err
	}
	return m.awaitErr(reply)
}

// Restart stops then respawns the targeted process(es). For currently-stopped
// processes, equivalent to Start.
func (m *Manager) Restart(name string, index int) error {
	reply := make(chan error, 1)
	if err := m.submit(restartCmd{Name: name, Index: index, Reply: reply}); err != nil {
		return err
	}
	return m.awaitErr(reply)
}

// Status returns a snapshot of every process Manager knows about.
// Returns nil if the engine has stopped.
func (m *Manager) Status() []ProcInfo {
	reply := make(chan []ProcInfo, 1)
	if err := m.submit(statusCmd{Reply: reply}); err != nil {
		return nil
	}
	select {
	case r := <-reply:
		return r
	case <-m.shutdownCtx.Done():
		return nil
	}
}

// Shutdown signals every live process and cancels Manager.Run.
// Idempotent and safe to call after engine has already stopped.
func (m *Manager) Shutdown() {
	reply := make(chan struct{}, 1)
	if err := m.submit(shutdownCmd{Reply: reply}); err != nil {
		return
	}
	select {
	case <-reply:
	case <-m.shutdownCtx.Done():
	}
}

// Reload applies a diff between the current program set and newSpecs.
// See reload.go for the per-program semantics.
func (m *Manager) Reload(newSpecs []Spec) error {
	reply := make(chan error, 1)
	if err := m.submit(reloadCmd{NewSpecs: newSpecs, Reply: reply}); err != nil {
		return err
	}
	return m.awaitErr(reply)
}

// submit enqueues e on m.events, refusing if the engine has stopped.
// NewManager initializes m.shutdownCtx to a sentinel that never cancels,
// so calls before Run block on the send (eventually drained when Run starts)
// rather than nil-derefing.
func (m *Manager) submit(e event) error {
	select {
	case m.events <- e:
		return nil
	case <-m.shutdownCtx.Done():
		return ErrEngineStopped
	}
}

// awaitErr waits for an error-reply, returning ErrEngineStopped if the
// engine shuts down before the reply arrives.
func (m *Manager) awaitErr(reply <-chan error) error {
	select {
	case err := <-reply:
		return err
	case <-m.shutdownCtx.Done():
		return ErrEngineStopped
	}
}
