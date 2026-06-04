package engine

import (
	"syscall"
	"testing"
)

func TestReload_UnchangedSpec_KeepsRunningProcesses(t *testing.T) {
	spec := testSpec("foo", 1)
	m, fr := newTestManager(t, []Spec{spec})

	// Bring it up.
	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	pidBefore := mustGetProc(t, m, "foo", 0).pid
	if pidBefore <= 0 {
		t.Fatalf("expected pid > 0, got %d", pidBefore)
	}

	// Reload with same spec.
	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{spec}, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("reload: %v", err)
	}

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Running {
		t.Fatalf("state = %v, want Running (unchanged)", p.state)
	}
	if p.pid != pidBefore {
		t.Fatalf("pid changed: was %d, now %d", pidBefore, p.pid)
	}
	// No new spawn calls (still 1 from initial start).
	if got := len(fr.Spawns()); got != 1 {
		t.Fatalf("spawns = %d, want 1 (no restart)", got)
	}
	// No signals sent.
	if got := len(fr.Signals()); got != 0 {
		t.Fatalf("signals = %d, want 0", got)
	}
}

func TestReload_ChangedSpec_RestartsRunningProcesses(t *testing.T) {
	spec := testSpec("foo", 1)
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	newSpec := spec
	newSpec.Bin = "/bin/false" // any change

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{newSpec}, Reply: reply}.handle(m)
	<-reply

	p := mustGetProc(t, m, "foo", 0)
	if !p.restartPending {
		t.Fatal("expected restartPending=true")
	}
	if p.state != Stopping {
		t.Fatalf("state = %v, want Stopping (stop signal sent)", p.state)
	}
	// Simulate the process actually exiting.
	procKilledBy("foo", 0, syscall.SIGTERM).handle(m)
	p = mustGetProc(t, m, "foo", 0)
	if p.state != Starting {
		t.Fatalf("after exit, state = %v, want Starting (respawned with new spec)", p.state)
	}
	// Spec on the program should be the new one.
	prg := m.programs["foo"]
	if prg.Spec.Bin != "/bin/false" {
		t.Fatalf("program spec not updated: bin=%q", prg.Spec.Bin)
	}
	// Two spawn calls: original + post-reload respawn.
	if got := len(fr.Spawns()); got != 2 {
		t.Fatalf("spawns = %d, want 2", got)
	}
}

func TestReload_AddedProgram_AutostartsWhenConfigured(t *testing.T) {
	existing := testSpec("foo", 1)
	m, fr := newTestManager(t, []Spec{existing})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	added := testSpec("bar", 1)
	added.Autostart = true

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{existing, added}, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := m.programs["bar"]; !ok {
		t.Fatal("bar should have been added")
	}
	p := mustGetProc(t, m, "bar", 0)
	if p.state != Starting {
		t.Fatalf("bar state = %v, want Starting (autostart)", p.state)
	}
	// Initial foo spawn + bar autostart = 2 spawns.
	if got := len(fr.Spawns()); got != 2 {
		t.Fatalf("spawns = %d, want 2", got)
	}
}

func TestReload_RemovedProgram_StopsAndDropsFromMap(t *testing.T) {
	keep := testSpec("foo", 1)
	drop := testSpec("bar", 1)
	m, _ := newTestManager(t, []Spec{keep, drop})

	startSync(t, m, "bar", 0)
	StartTimerFired{Name: "bar", Index: 0}.handle(m)

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{keep}, Reply: reply}.handle(m)
	<-reply

	// bar still in map but marked for removal; proc was sent SIGTERM and
	// transitioned to Stopping.
	prg, ok := m.programs["bar"]
	if !ok {
		t.Fatal("bar still expected in map until proc exits")
	}
	if !prg.removeWhenEmpty {
		t.Fatal("bar should be marked for removal")
	}
	p := mustGetProc(t, m, "bar", 0)
	if !p.removeAfterExit {
		t.Fatal("bar's proc should be marked removeAfterExit")
	}
	if p.state != Stopping {
		t.Fatalf("state = %v, want Stopping", p.state)
	}

	// Simulate exit.
	procKilledBy("bar", 0, syscall.SIGTERM).handle(m)
	if _, ok := m.programs["bar"]; ok {
		t.Fatal("bar should be removed from map after its proc exited")
	}
}

func TestReload_NumprocsGrows_AddsAndAutostartsNewSlots(t *testing.T) {
	old := testSpec("foo", 2)
	old.Autostart = true
	m, fr := newTestManager(t, []Spec{old})

	// Bring up both.
	startSync(t, m, "foo", 0)
	startSync(t, m, "foo", 1)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	StartTimerFired{Name: "foo", Index: 1}.handle(m)
	pidsBefore := [2]int{
		mustGetProc(t, m, "foo", 0).pid,
		mustGetProc(t, m, "foo", 1).pid,
	}

	ns := old
	ns.Numprocs = 4

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{ns}, Reply: reply}.handle(m)
	<-reply

	prg := m.programs["foo"]
	if len(prg.procs) != 4 {
		t.Fatalf("len(procs) = %d, want 4", len(prg.procs))
	}
	// Numprocs-only change: existing slots keep running, no restart pending.
	for i := 0; i < 2; i++ {
		if prg.procs[i].restartPending {
			t.Errorf("slot %d should NOT be restartPending (numprocs-only reload)", i)
		}
		if prg.procs[i].state != Running {
			t.Errorf("slot %d state = %v, want Running (unchanged)", i, prg.procs[i].state)
		}
		if prg.procs[i].pid != pidsBefore[i] {
			t.Errorf("slot %d pid changed: was %d, now %d", i, pidsBefore[i], prg.procs[i].pid)
		}
	}
	// New slots are Starting (spawn called during reload).
	for i := 2; i < 4; i++ {
		if prg.procs[i].state != Starting {
			t.Errorf("slot %d state = %v, want Starting", i, prg.procs[i].state)
		}
	}
	// 2 original spawns + 2 new-slot spawns; no respawn of survivors.
	if got := len(fr.Spawns()); got != 4 {
		t.Fatalf("spawns = %d, want 4 (2 original + 2 new slots)", got)
	}
	// No signals: surviving slots were never asked to stop.
	if got := len(fr.Signals()); got != 0 {
		t.Fatalf("signals = %d, want 0 (survivors untouched)", got)
	}
}

func TestReload_NumprocsShrinks_MarksSurplusForRemoval(t *testing.T) {
	old := testSpec("foo", 3)
	m, _ := newTestManager(t, []Spec{old})

	// Bring up all three.
	for i := 0; i < 3; i++ {
		startSync(t, m, "foo", i)
		StartTimerFired{Name: "foo", Index: i}.handle(m)
	}
	pidKept := mustGetProc(t, m, "foo", 0).pid

	ns := old
	ns.Numprocs = 1

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{ns}, Reply: reply}.handle(m)
	<-reply

	prg := m.programs["foo"]
	if len(prg.procs) != 3 {
		t.Fatalf("len(procs) = %d, want 3 (still alive)", len(prg.procs))
	}
	// Numprocs-only shrink: surviving slot 0 is untouched.
	if prg.procs[0].restartPending {
		t.Error("slot 0 should NOT be restartPending (numprocs-only reload)")
	}
	if prg.procs[0].state != Running {
		t.Errorf("slot 0 state = %v, want Running (unchanged)", prg.procs[0].state)
	}
	if prg.procs[0].pid != pidKept {
		t.Errorf("slot 0 pid changed: was %d, now %d", pidKept, prg.procs[0].pid)
	}
	if !prg.procs[1].removeAfterExit || !prg.procs[2].removeAfterExit {
		t.Fatal("indices 1 and 2 should be removeAfterExit")
	}
	if prg.procs[1].state != Stopping || prg.procs[2].state != Stopping {
		t.Fatal("indices 1 and 2 should be Stopping")
	}

	// Simulate exits — slots are pruned.
	procKilledBy("foo", 2, syscall.SIGTERM).handle(m)
	procKilledBy("foo", 1, syscall.SIGTERM).handle(m)

	if got := len(prg.procs); got != 1 {
		t.Fatalf("after exits, len(procs) = %d, want 1", got)
	}
}

// Regression: pruneAfterRemoval used to re-number .index on survivors after
// every slot removal, which mis-targeted any ProcExited / StopTimerFired that
// captured an index by value at spawn or stop time. With shrinks of N>2 and
// non-tail-first exit order, surplus procs could be stranded in Stopping
// because their stopTimer's captured idx pointed at a slot that no longer
// existed. Drive an out-of-order shrink to lock the behavior in.
func TestReload_NumprocsShrinks_OutOfOrderExits_AllPruned(t *testing.T) {
	old := testSpec("foo", 5)
	m, _ := newTestManager(t, []Spec{old})

	for i := 0; i < 5; i++ {
		startSync(t, m, "foo", i)
		StartTimerFired{Name: "foo", Index: i}.handle(m)
	}
	pidKept := mustGetProc(t, m, "foo", 0).pid

	ns := old
	ns.Numprocs = 1

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{ns}, Reply: reply}.handle(m)
	<-reply

	// Surplus exits arrive out of order, including non-tail-first.
	for _, idx := range []int{2, 4, 1, 3} {
		procKilledBy("foo", idx, syscall.SIGTERM).handle(m)
	}

	prg := m.programs["foo"]
	if got := len(prg.procs); got != 1 {
		t.Fatalf("after exits, len(procs) = %d, want 1", got)
	}
	if prg.procs[0].index != 0 {
		t.Fatalf("survivor .index = %d, want 0 (stable)", prg.procs[0].index)
	}
	if prg.procs[0].state != Running {
		t.Fatalf("survivor state = %v, want Running", prg.procs[0].state)
	}
	if prg.procs[0].pid != pidKept {
		t.Fatalf("survivor pid changed: was %d, now %d", pidKept, prg.procs[0].pid)
	}
}

func TestReload_NumprocsAndOtherFieldChange_RestartsSurvivors(t *testing.T) {
	old := testSpec("foo", 2)
	old.Autostart = true
	m, _ := newTestManager(t, []Spec{old})

	startSync(t, m, "foo", 0)
	startSync(t, m, "foo", 1)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	StartTimerFired{Name: "foo", Index: 1}.handle(m)

	ns := old
	ns.Numprocs = 3
	ns.Bin = "/bin/false" // any non-numprocs change

	reply := make(chan error, 1)
	reloadCmd{NewSpecs: []Spec{ns}, Reply: reply}.handle(m)
	<-reply

	prg := m.programs["foo"]
	// Survivors are restarted because something other than numprocs changed.
	for i := 0; i < 2; i++ {
		if !prg.procs[i].restartPending {
			t.Errorf("slot %d should be restartPending", i)
		}
	}
	// The grown slot is up.
	if prg.procs[2].state != Starting {
		t.Errorf("slot 2 state = %v, want Starting", prg.procs[2].state)
	}
}
