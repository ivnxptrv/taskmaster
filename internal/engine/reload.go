package engine

import (
	"fmt"
)

// reloadCmd is the event a SIGHUP / shell `reload` triggers. It carries the
// freshly-parsed specs. Handler runs the diff vs current m.programs and
// applies actions. Reply is non-nil error if anything went wrong while
// processing the diff itself (not when individual programs fail to restart —
// those are logged, not propagated).
type reloadCmd struct {
	NewSpecs []Spec
	Reply    chan error
}

func (e reloadCmd) handle(m *Manager) {
	newByName := make(map[string]Spec, len(e.NewSpecs))
	for _, s := range e.NewSpecs {
		if _, dup := newByName[s.Name]; dup {
			e.Reply <- fmt.Errorf("duplicate program name in new config: %q", s.Name)
			return
		}
		newByName[s.Name] = s
	}

	// 1. Programs that exist in BOTH old and new: equal → leave alone;
	//    different → in-place restart (preserves pids of unchanged programs).
	for name, prg := range m.programs {
		ns, present := newByName[name]
		if !present { // brand new program
			continue // handled in step 3
		}
		if prg.Spec.Equal(ns) { // same program unchanged
			m.log.Info("reload: program unchanged", "program", name)
			continue
		}
		if prg.Spec.equalIgnoringNumprocs(ns) {
			m.log.Info("reload: program numprocs changed; resizing",
				"program", name,
				"old_numprocs", prg.Spec.Numprocs, "new_numprocs", ns.Numprocs)
		} else {
			m.log.Info("reload: program changed; restarting", "program", name,
				"old_numprocs", prg.Spec.Numprocs, "new_numprocs", ns.Numprocs)
		}

		// same program but changed spec
		applyChangedSpec(m, prg, ns)
	}

	// 2. Programs in new but not in old → ADDED.
	for name, ns := range newByName {
		if _, present := m.programs[name]; present {
			continue
		}
		m.log.Info("reload: program added", "program", name, "numprocs", ns.Numprocs)
		prg := newProgram(ns)
		m.programs[name] = prg
		if ns.Autostart {
			for _, p := range prg.procs {
				if err := m.spawn(&prg.Spec, p); err != nil {
					m.log.Error("reload-autostart failed",
						"program", name, "index", p.index, "err", err)
				}
			}
		}
	}

	// 3. Programs in old but not in new → REMOVED.
	for name, prg := range m.programs {
		if _, present := newByName[name]; present {
			continue
		}
		m.log.Info("reload: program removed", "program", name)
		prg.removeWhenEmpty = true
		for _, p := range prg.procs {
			p.removeAfterExit = true
			p.restartPending = false
		}
		liveCount := 0
		for _, p := range prg.procs {
			if isLive(p.state) {
				stopProcess(m, &prg.Spec, p)
				liveCount++
			}
		}
		if liveCount == 0 {
			// Nothing to stop — prune immediately.
			pruneProgram(m, name, prg)
		}
	}

	e.Reply <- nil
}

// applyChangedSpec updates the Program's spec and reconciles processes:
//   - Procs at indices common to old/new numprocs: if only Numprocs differs,
//     leave them alone; otherwise live → flag restart and stopProcess
//     (ProcExited will respawn with the new spec); dead → leave (next
//     user-issued Start uses new spec).
//   - Surplus old procs (index >= new.Numprocs): mark removeAfterExit and stop.
//   - New procs (index >= old.Numprocs): allocate; spawn if Autostart.
func applyChangedSpec(m *Manager, prg *Program, ns Spec) {
	oldN := len(prg.procs)
	newN := ns.Numprocs
	oldSpec := prg.Spec
	prg.Spec = ns

	// Indices that exist on both sides → in-place restart, unless the only
	// change is Numprocs (then the surviving slots keep running untouched).
	common := oldN
	if newN < common {
		common = newN
	}
	numprocsOnly := oldSpec.equalIgnoringNumprocs(ns)
	if !numprocsOnly {
		for i := 0; i < common; i++ {
			p := prg.procs[i]
			switch p.state {
			case Starting, Running, Backoff:
				// set restart flag and kill have it then restarted on exit with new spec
				p.restartPending = true
				if p.state == Backoff && p.backoffTimer != nil {
					p.backoffTimer.Stop()
					p.backoffTimer = nil
					p.wipe()
					if err := m.spawn(&prg.Spec, p); err != nil {
						m.log.Error("reload spawn failed", "program", ns.Name, "index", i, "err", err)
					}
				} else {
					stopProcess(m, &prg.Spec, p)
				}
			case Stopping:
				p.restartPending = true
			case NeverStarted, Stopped, Exited, Fatal:
				// Dead — leave; user can Start to bring it up with new spec.
			}
		}
	}

	// Surplus old procs (newN < oldN): mark for removal, stop if live.
	for i := newN; i < oldN; i++ {
		p := prg.procs[i]
		p.removeAfterExit = true
		p.restartPending = false
		if isLive(p.state) {
			stopProcess(m, &prg.Spec, p)
		}
	}

	// New slots (newN > oldN): allocate; autostart if configured.
	for i := oldN; i < newN; i++ {
		p := newProcess(i)
		prg.procs = append(prg.procs, p)
		if ns.Autostart {
			if err := m.spawn(&prg.Spec, p); err != nil {
				m.log.Error("reload-grow spawn failed",
					"program", ns.Name, "index", i, "err", err)
			}
		}
	}

	// Surplus dead procs: prune immediately (no exit event coming).
	pruneSurplusDead(prg, newN)
	if prg.Spec.Numprocs == 0 {
		// Edge case: shrunk to zero. The map entry remains (it's not a
		// "removed program") but has no procs. Next reload can grow it.
	}
}

// pruneSurplusDead drops trailing procs at indices >= newN that are not live.
// Live ones stay until their ProcExited handles the removal.
func pruneSurplusDead(prg *Program, newN int) {
	for len(prg.procs) > newN {
		p := prg.procs[len(prg.procs)-1]
		if isLive(p.state) {
			return
		}
		prg.procs = prg.procs[:len(prg.procs)-1]
	}
}

// pruneProgram removes the Program from Manager.programs if no procs remain.
// Called from ProcExited when removeAfterExit completes.
func pruneProgram(m *Manager, name string, prg *Program) {
	if !prg.removeWhenEmpty {
		return
	}
	for _, p := range prg.procs {
		if isLive(p.state) {
			return
		}
	}
	delete(m.programs, name)
	m.log.Info("reload: program fully removed", "program", name)
}

// pruneAfterRemoval is called from ProcExited when p.removeAfterExit is true.
// Drops p from prg.procs (preserving the .index of every survivor) and, if
// the whole program is being removed and now has no procs, drops the program
// from Manager.programs.
//
// Survivor .index values are NEVER rewritten. Wait-goroutines and stop/start
// timers capture their proc's index by value into closures at spawn/stop time
// and re-numbering would mis-target every captured event still in flight.
// Program.process(index) does a linear scan so lookups remain correct even
// when the slice has holes in its old [0..numprocs) range.
func pruneAfterRemoval(m *Manager, prg *Program, p *Process) {
	for i, q := range prg.procs {
		if q == p {
			prg.procs = append(prg.procs[:i], prg.procs[i+1:]...)
			break
		}
	}
	if prg.removeWhenEmpty {
		// Find this prg in m.programs and check if we can drop it.
		for name, candidate := range m.programs {
			if candidate == prg {
				pruneProgram(m, name, prg)
				break
			}
		}
	}
}
