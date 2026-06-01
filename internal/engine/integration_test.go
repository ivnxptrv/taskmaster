package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// These tests exercise the real osRuntime: real fork/exec, real signals,
// real file IO. They use small portable binaries (sh, sleep, echo, true).
// All tests are wrapped in a deadline-bounded ctx so a stuck child can't
// hang the suite forever.

func TestOsRuntime_SpawnEchoCapturesOutputAndReportsExit(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	stdoutPath := filepath.Join(tmp, "out.log")
	stderrPath := filepath.Join(tmp, "err.log")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := make(chan event, 4)
	rt := NewOsRuntime(quietLogger())

	res, err := rt.SpawnProcess(ctx, SpawnRequest{
		Name:       "echoer",
		Index:      0,
		Bin:        findEcho(),
		Args:       []string{"hello world"},
		Umask:      022,
		Workingdir: tmp,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
		Env:        os.Environ(),
	}, events)
	if err != nil {
		t.Fatalf("SpawnProcess: %v", err)
	}
	if res.Cmd == nil {
		t.Fatal("SpawnResult.Cmd is nil")
	}

	select {
	case e := <-events:
		pe, ok := e.(ProcExited)
		if !ok {
			t.Fatalf("got %T, want ProcExited", e)
		}
		if pe.Status != nil {
			t.Fatalf("Status = %v, want nil (exit 0)", pe.Status)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for ProcExited")
	}

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world\n" {
		t.Fatalf("stdout = %q", string(data))
	}
}

func TestOsRuntime_SignalActuallyDelivers(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := make(chan event, 4)
	rt := NewOsRuntime(quietLogger())

	res, err := rt.SpawnProcess(ctx, SpawnRequest{
		Name:       "sleeper",
		Bin:        findSleep(),
		Args:       []string{"30"},
		Workingdir: tmp,
		StdoutPath: filepath.Join(tmp, "o"),
		StderrPath: filepath.Join(tmp, "e"),
		Env:        os.Environ(),
	}, events)
	if err != nil {
		t.Fatal(err)
	}

	if err := rt.Signal(res.Cmd, syscall.SIGTERM); err != nil {
		t.Fatalf("Signal: %v", err)
	}

	select {
	case e := <-events:
		pe := e.(ProcExited)
		if pe.Status == nil {
			t.Fatal("Status = nil — process exited cleanly somehow; SIGTERM was expected to kill it")
		}
	case <-ctx.Done():
		t.Fatal("timeout — process did not exit after SIGTERM")
	}
}

// Defense scenario: process that ignores SIGTERM gets SIGKILL after stoptime.
// Drives the full Manager: stop command → stopProcess → StopTimerFired → SIGKILL.
func TestIntegration_SIGTERMIgnoredEscalatesToSIGKILL(t *testing.T) {
	t.Parallel()
	sh := findShell()
	tmp := t.TempDir()

	spec := Spec{
		Name:         "stubborn",
		Bin:          sh,
		Args:         []string{"-c", "trap '' TERM; sleep 30"},
		Numprocs:     1,
		Workingdir:   tmp,
		Env:          os.Environ(),
		Autostart:    true,
		Autorestart:  RestartNever,
		Exitcodes:    []int{0},
		Startretries: 0,
		Starttime:    100 * time.Millisecond,
		Stopsignal:   syscall.SIGTERM,
		Stoptime:     250 * time.Millisecond, // short so SIGKILL fires quickly
		StdoutPath:   filepath.Join(tmp, "o"),
		StderrPath:   filepath.Join(tmp, "e"),
	}

	mgr, err := NewManager([]Spec{spec}, NewOsRuntime(testLogger(t)), testLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	mgr.gracefulDeadline = 3 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- mgr.Run(ctx) }()

	t.Logf("waiting for stubborn[0] to reach Running…")
	require := requireRunning(t, mgr, "stubborn", 0, 3*time.Second)
	t.Logf("stubborn[0] running with pid=%d", require.Pid)

	// Issue stop; it'll send SIGTERM (ignored) then SIGKILL after stoptime.
	if err := mgr.Stop("stubborn", 0); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Wait until the process actually exits (state Stopped or Exited).
	// Capture the final state BEFORE shutting the engine down.
	deadline := time.Now().Add(2 * time.Second)
	var final ProcInfo
	for time.Now().Before(deadline) {
		infos := mgr.Status()
		if len(infos) == 1 && (infos[0].State == "Stopped" || infos[0].State == "Exited") {
			final = infos[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mgr.Shutdown()
	<-runErr

	if final.State != "Stopped" && final.State != "Exited" {
		t.Fatalf("final state = %q, want Stopped or Exited (SIGKILL escalation expected)", final.State)
	}
}

// Defense scenario: process that produces a lot of output. Verifies the
// pipe-less *os.File approach doesn't deadlock — supervisor doesn't read
// the file, the child writes directly.
func TestIntegration_LotsOfOutputDoesNotDeadlock(t *testing.T) {
	t.Parallel()
	sh := findShell()
	tmp := t.TempDir()
	stdoutPath := filepath.Join(tmp, "big.out")

	spec := Spec{
		Name:         "noisy",
		Bin:          sh,
		Args:         []string{"-c", "i=0; while [ $i -lt 5000 ]; do echo line-$i; i=$((i+1)); done"},
		Numprocs:     1,
		Workingdir:   tmp,
		Env:          os.Environ(),
		Autostart:    true,
		Autorestart:  RestartNever,
		Exitcodes:    []int{0},
		Startretries: 0,
		Starttime:    100 * time.Millisecond,
		Stopsignal:   syscall.SIGTERM,
		Stoptime:     2 * time.Second,
		StdoutPath:   stdoutPath,
		StderrPath:   "/dev/null",
	}

	mgr, err := NewManager([]Spec{spec}, NewOsRuntime(quietLogger()), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- mgr.Run(ctx) }()

	// Wait until the program finishes naturally.
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		infos := mgr.Status()
		if len(infos) == 1 && infos[0].State == "Exited" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mgr.Shutdown()
	<-runErr

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	lines := strings.Count(string(data), "\n")
	if lines < 5000 {
		t.Fatalf("only %d lines in output; expected ~5000 (deadlock or truncation?)", lines)
	}
}

// Defense scenario: external kill of a supervised process triggers autorestart.
func TestIntegration_ExternalKillTriggersAutorestart(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	spec := Spec{
		Name:         "respawner",
		Bin:          findSleep(),
		Args:         []string{"30"},
		Numprocs:     1,
		Workingdir:   tmp,
		Env:          os.Environ(),
		Autostart:    true,
		Autorestart:  RestartAlways,
		Exitcodes:    []int{0},
		Startretries: 3,
		Starttime:    100 * time.Millisecond,
		Stopsignal:   syscall.SIGTERM,
		Stoptime:     time.Second,
		StdoutPath:   "/dev/null",
		StderrPath:   "/dev/null",
	}

	mgr, err := NewManager([]Spec{spec}, NewOsRuntime(quietLogger()), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- mgr.Run(ctx) }()
	defer func() { mgr.Shutdown(); <-runErr }()

	// Get the pid once it's Running.
	requireRunning(t, mgr, "respawner", 0, 2*time.Second)
	firstPid := mgr.Status()[0].Pid
	if firstPid <= 0 {
		t.Fatal("invalid pid before kill")
	}

	// External SIGKILL — bypasses Manager.
	if err := exec.Command("kill", "-KILL", itoa(firstPid)).Run(); err != nil {
		t.Fatalf("external kill: %v", err)
	}

	// Wait for the autorestart to bring up a new pid.
	requireNewPid(t, mgr, "respawner", 0, firstPid, 3*time.Second)
}

// ── helpers for integration tests ─────────────────────────────────────

func requireRunning(t *testing.T, mgr *Manager, name string, index int, within time.Duration) ProcInfo {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		for _, p := range mgr.Status() {
			if p.Name == name && p.Index == index && p.State == "Running" {
				return p
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s[%d] did not reach Running within %v", name, index, within)
	return ProcInfo{}
}

func requireNewPid(t *testing.T, mgr *Manager, name string, index, oldPid int, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		for _, p := range mgr.Status() {
			if p.Name == name && p.Index == index && p.State == "Running" && p.Pid != oldPid && p.Pid > 0 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s[%d] did not get a new pid within %v (old=%d)", name, index, within, oldPid)
}

func findEcho() string {
	for _, p := range []string{"/bin/echo", "/run/current-system/sw/bin/echo", "/usr/bin/echo"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "echo"
}

func findSleep() string {
	for _, p := range []string{"/bin/sleep", "/run/current-system/sw/bin/sleep", "/usr/bin/sleep"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "sleep"
}
