package engine

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// ── Test helpers ──────────────────────────────────────────────────────

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testLogger writes structured logs through t.Logf so failures show what
// the engine was doing in the moments before timeout.
func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(b []byte) (int, error) {
	w.t.Log(string(b))
	return len(b), nil
}

// testSpec returns a Spec with sane defaults for unit tests.
// Durations are short so any real timer that does fire doesn't slow tests.
func testSpec(name string, numprocs int) Spec {
	return Spec{
		Name:         name,
		Bin:          "/bin/true",
		Args:         nil,
		Numprocs:     numprocs,
		Workingdir:   "/tmp",
		Env:          os.Environ(),
		Autostart:    false,
		Autorestart:  RestartNever,
		Exitcodes:    []int{0},
		Startretries: 3,
		Starttime:    10 * time.Millisecond,
		Stopsignal:   syscall.SIGTERM,
		Stoptime:     50 * time.Millisecond,
		StdoutPath:   "/dev/null",
		StderrPath:   "/dev/null",
	}
}

// newTestManager builds a Manager with the fake runtime + a noop logger
// and gives it a shutdownCtx without running the loop. Tests drive the
// loop by calling stepOnce or invoking handlers directly.
func newTestManager(t *testing.T, specs []Spec) (*Manager, *fakeRuntime) {
	t.Helper()
	fr := newFakeRuntime()
	m, err := NewManager(specs, fr, quietLogger())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.shutdownCtx, m.shutdownCancel = ctx, cancel
	t.Cleanup(cancel)
	return m, fr
}

// stepOnce reads exactly one event from m.events and runs its handler.
// Uses a short timeout so a missing event surfaces as a test failure rather
// than a hang.
func stepOnce(t *testing.T, m *Manager) {
	t.Helper()
	select {
	case e := <-m.events:
		e.handle(m)
	case <-time.After(time.Second):
		t.Fatal("stepOnce: no event arrived")
	}
}

// drain processes all currently-pending events.
func drain(m *Manager) {
	for {
		select {
		case e := <-m.events:
			e.handle(m)
		default:
			return
		}
	}
}

func mustGetProc(t *testing.T, m *Manager, name string, index int) *Process {
	t.Helper()
	p, _ := m.lookup(name, index)
	if p == nil {
		t.Fatalf("lookup(%q,%d) returned nil", name, index)
	}
	return p
}

// procExitedWithCode synthesizes a ProcExited as if the child exited with
// the given code. Status==nil for code 0.
func procExitedWithCode(name string, index, code int) ProcExited {
	if code == 0 {
		return ProcExited{Name: name, Index: index, Status: nil}
	}
	return ProcExited{
		Name:   name,
		Index:  index,
		Status: makeExitErr(code, false),
	}
}

// procKilledBy synthesizes a ProcExited as if the child was killed by sig.
func procKilledBy(name string, index int, sig syscall.Signal) ProcExited {
	return ProcExited{
		Name:   name,
		Index:  index,
		Status: makeExitErr(int(sig), true),
	}
}

// makeExitErr fabricates an *exec.ExitError carrying a WaitStatus that
// parseProcessStatus will read. We do this by running /bin/true (always
// available enough — /run/current-system/sw/bin/true on NixOS, /bin/true
// elsewhere) and then overlaying a synthetic ProcessState would require
// unexported access. Instead we wrap an error type that parseProcessStatus
// recognizes by Status pattern. The simplest portable approach: run a real
// /bin/sh -c "exit N" and capture the resulting *exec.ExitError.
//
// This is the one piece of "real" cmd execution remaining in unit tests,
// but it runs in ~5ms per call and is fully deterministic.
func makeExitErr(code int, bySignal bool) error {
	bin := findShell()
	var cmd *exec.Cmd
	if bySignal {
		cmd = exec.Command(bin, "-c", "kill -"+itoa(code)+" $$")
	} else {
		cmd = exec.Command(bin, "-c", "exit "+itoa(code))
	}
	return cmd.Run()
}

func findShell() string {
	for _, p := range []string{"/bin/sh", "/run/current-system/sw/bin/sh", "/usr/bin/sh"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "sh"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ── Tests ─────────────────────────────────────────────────────────────

func TestStart_SpawnsAndTransitionsToStarting(t *testing.T) {
	m, fr := newTestManager(t, []Spec{testSpec("foo", 1)})

	// Tests bypass the public API (which blocks on the loop's reply); they
	// invoke handlers directly, which is equivalent at the actor boundary.
	reply := make(chan error, 1)
	startCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("startCmd: %v", err)
	}

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Starting {
		t.Fatalf("state = %v, want Starting", p.state)
	}
	if len(fr.Spawns()) != 1 {
		t.Fatalf("spawns = %d, want 1", len(fr.Spawns()))
	}
}

func TestStartTimerFired_PromotesStartingToRunning(t *testing.T) {
	m, _ := newTestManager(t, []Spec{testSpec("foo", 1)})
	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Running {
		t.Fatalf("state = %v, want Running", p.state)
	}
	if p.retries != 0 {
		t.Fatalf("retries = %d, want 0", p.retries)
	}
}

func TestStartTimerFired_IgnoresWhenNotStarting(t *testing.T) {
	m, _ := newTestManager(t, []Spec{testSpec("foo", 1)})
	p := mustGetProc(t, m, "foo", 0)
	p.state = Exited
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	if p.state != Exited {
		t.Fatalf("state = %v, want Exited (stale event ignored)", p.state)
	}
}

func TestProcExited_StartingBelowRetries_GoesBackoff(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Startretries = 3
	m, _ := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	procExitedWithCode("foo", 0, 1).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Backoff {
		t.Fatalf("state = %v, want Backoff", p.state)
	}
	if p.retries != 1 {
		t.Fatalf("retries = %d, want 1", p.retries)
	}
}

func TestProcExited_StartingExhaustsRetries_GoesFatal(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Startretries = 2
	m, _ := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	for i := 0; i < 3; i++ {
		// simulate: process keeps exiting before StartTimerFired.
		procExitedWithCode("foo", 0, 1).handle(m)
		if i < 2 {
			BackoffElapsed{Name: "foo", Index: 0}.handle(m) // respawns; state=Starting
		}
	}

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Fatal {
		t.Fatalf("state = %v, want Fatal after %d retries", p.state, spec.Startretries)
	}
}

func TestProcExited_RunningWithRestartAlways_Respawns(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Autorestart = RestartAlways
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	procExitedWithCode("foo", 0, 7).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Starting {
		t.Fatalf("state = %v, want Starting (autorestart=always)", p.state)
	}
	if len(fr.Spawns()) != 2 {
		t.Fatalf("spawns = %d, want 2", len(fr.Spawns()))
	}
}

func TestProcExited_RunningWithRestartNever_Exited(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Autorestart = RestartNever
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	procExitedWithCode("foo", 0, 7).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Exited {
		t.Fatalf("state = %v, want Exited", p.state)
	}
	if len(fr.Spawns()) != 1 {
		t.Fatalf("spawns = %d, want 1 (no respawn)", len(fr.Spawns()))
	}
}

func TestProcExited_RunningUnexpected_ExpectedCode_Exited(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Autorestart = RestartUnexpected
	spec.Exitcodes = []int{0, 2}
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	procExitedWithCode("foo", 0, 2).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Exited {
		t.Fatalf("state = %v, want Exited (code is in expected set)", p.state)
	}
	if len(fr.Spawns()) != 1 {
		t.Fatalf("spawns = %d, want 1 (no respawn)", len(fr.Spawns()))
	}
}

func TestProcExited_RunningUnexpected_UnexpectedCode_Respawns(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Autorestart = RestartUnexpected
	spec.Exitcodes = []int{0}
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	procExitedWithCode("foo", 0, 7).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Starting {
		t.Fatalf("state = %v, want Starting (unexpected exit)", p.state)
	}
	if len(fr.Spawns()) != 2 {
		t.Fatalf("spawns = %d, want 2", len(fr.Spawns()))
	}
}

func TestStop_SendsConfiguredSignalAndTransitionsToStopping(t *testing.T) {
	spec := testSpec("foo", 1)
	spec.Stopsignal = syscall.SIGUSR1
	m, fr := newTestManager(t, []Spec{spec})

	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	reply := make(chan error, 1)
	stopCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("stopCmd: %v", err)
	}

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Stopping {
		t.Fatalf("state = %v, want Stopping", p.state)
	}
	sigs := fr.Signals()
	if len(sigs) != 1 || sigs[0].Sig != syscall.SIGUSR1 {
		t.Fatalf("signals = %+v, want [SIGUSR1]", sigs)
	}
}

func TestStopTimerFired_EscalatesToSIGKILL(t *testing.T) {
	m, fr := newTestManager(t, []Spec{testSpec("foo", 1)})
	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	reply := make(chan error, 1)
	stopCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	<-reply

	StopTimerFired{Name: "foo", Index: 0}.handle(m)

	last, ok := fr.lastSignal()
	if !ok || last.Sig != syscall.SIGKILL {
		t.Fatalf("last signal = %+v, want SIGKILL", last)
	}
}

func TestProcExited_StoppedBySignal_GoesStopped(t *testing.T) {
	m, _ := newTestManager(t, []Spec{testSpec("foo", 1)})
	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)
	mustGetProc(t, m, "foo", 0).state = Stopping
	procKilledBy("foo", 0, syscall.SIGTERM).handle(m)

	p := mustGetProc(t, m, "foo", 0)
	if p.state != Stopped {
		t.Fatalf("state = %v, want Stopped", p.state)
	}
}

func TestRestart_OnRunningProcess_SetsFlagAndStops(t *testing.T) {
	m, fr := newTestManager(t, []Spec{testSpec("foo", 1)})
	startSync(t, m, "foo", 0)
	StartTimerFired{Name: "foo", Index: 0}.handle(m)

	reply := make(chan error, 1)
	restartCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("restartCmd: %v", err)
	}

	p := mustGetProc(t, m, "foo", 0)
	if !p.restartPending {
		t.Fatal("restartPending should be true")
	}
	if p.state != Stopping {
		t.Fatalf("state = %v, want Stopping", p.state)
	}

	// Simulate the process actually exiting → should respawn.
	procKilledBy("foo", 0, syscall.SIGTERM).handle(m)
	if p.state != Starting {
		t.Fatalf("after exit, state = %v, want Starting (restart respawned)", p.state)
	}
	if len(fr.Spawns()) != 2 {
		t.Fatalf("spawns = %d, want 2", len(fr.Spawns()))
	}
}

func TestRestart_OnStoppedProcess_SpawnsDirectly(t *testing.T) {
	m, fr := newTestManager(t, []Spec{testSpec("foo", 1)})
	p := mustGetProc(t, m, "foo", 0)
	p.state = Stopped

	reply := make(chan error, 1)
	restartCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("restartCmd: %v", err)
	}

	if p.state != Starting {
		t.Fatalf("state = %v, want Starting", p.state)
	}
	if len(fr.Spawns()) != 1 {
		t.Fatalf("spawns = %d, want 1", len(fr.Spawns()))
	}
}

func TestStatus_ReturnsSnapshotForAllProcesses(t *testing.T) {
	m, _ := newTestManager(t, []Spec{testSpec("foo", 2), testSpec("bar", 1)})
	infos := readStatus(t, m)
	if len(infos) != 3 {
		t.Fatalf("status entries = %d, want 3 (foo*2 + bar)", len(infos))
	}
}

func TestStart_UnknownProgram_ReturnsError(t *testing.T) {
	m, _ := newTestManager(t, []Spec{testSpec("foo", 1)})
	reply := make(chan error, 1)
	startCmd{Name: "ghost", Index: 0, Reply: reply}.handle(m)
	err := <-reply
	if err == nil {
		t.Fatal("expected error for unknown program, got nil")
	}
}

func TestStop_IdempotentOnAlreadyStopped(t *testing.T) {
	m, fr := newTestManager(t, []Spec{testSpec("foo", 1)})
	// Process is in NeverStarted; Stop should be no-op and not signal.
	reply := make(chan error, 1)
	stopCmd{Name: "foo", Index: 0, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("stopCmd on NeverStarted: %v", err)
	}
	if got := len(fr.Signals()); got != 0 {
		t.Fatalf("signals = %d, want 0", got)
	}
}

// ── small helpers ────────────────────────────────────────────────────

// startSync runs the startCmd handler synchronously (the public API would
// block waiting for reply via the loop; tests call handlers directly).
func startSync(t *testing.T, m *Manager, name string, index int) {
	t.Helper()
	reply := make(chan error, 1)
	startCmd{Name: name, Index: index, Reply: reply}.handle(m)
	if err := <-reply; err != nil {
		t.Fatalf("start %s[%d]: %v", name, index, err)
	}
}

func readStatus(t *testing.T, m *Manager) []ProcInfo {
	t.Helper()
	reply := make(chan []ProcInfo, 1)
	statusCmd{Reply: reply}.handle(m)
	return <-reply
}
