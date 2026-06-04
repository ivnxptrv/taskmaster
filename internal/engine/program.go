package engine

// Program bundles a Spec (desired) with its current Processes (observed).
// Manager owns a map[string]*Program keyed by spec name.
type Program struct {
	Spec  Spec
	procs []*Process

	// removeWhenEmpty is set by reload when the whole program was deleted.
	// Once all procs have exited and been pruned, the program is dropped
	// from Manager.programs.
	removeWhenEmpty bool
}

func newProgram(spec Spec) *Program {
	prg := &Program{Spec: spec, procs: make([]*Process, 0, spec.Numprocs)}
	for i := 0; i < spec.Numprocs; i++ {
		prg.procs = append(prg.procs, newProcess(i))
	}
	return prg
}

// process returns the proc whose .index matches, or nil if none exists.
// Lookup is by the proc's .index field, not by slice position: indices are
// stable for the proc's lifetime, so wait-goroutines and timers that
// captured an index at spawn/stop time always resolve back to the same
// proc even after siblings have been pruned out of the slice.
func (p *Program) process(index int) *Process {
	for _, proc := range p.procs {
		if proc.index == index {
			return proc
		}
	}
	return nil
}
