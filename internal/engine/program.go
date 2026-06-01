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

// process returns the process at index, or nil if out of range.
func (p *Program) process(index int) *Process {
	if index < 0 || index >= len(p.procs) {
		return nil
	}
	return p.procs[index]
}
