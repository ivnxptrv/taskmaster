package engine

import (
	"context"
	"os/exec"
	"sync"
	"syscall"
)

// fakeRuntime is a Runtime implementation for unit tests. It records every
// SpawnProcess / Signal call and lets tests drive process lifecycle manually
// by enqueueing ProcExited events themselves.
//
// SpawnProcess returns a SpawnResult with a dummy *exec.Cmd (just enough so
// downstream code that touches cmd.Process.Pid doesn't nil-panic).
type fakeRuntime struct {
	mu sync.Mutex

	spawns  []SpawnRequest
	signals []signalCall

	// nextPid is incremented on each SpawnProcess so each fake process has
	// a distinct pid like real ones would.
	nextPid int

	// Injectable errors
	SpawnErr  error
	SignalErr error
}

type signalCall struct {
	Pid int
	Sig syscall.Signal
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{nextPid: 1000}
}

func (f *fakeRuntime) SpawnProcess(_ context.Context, req SpawnRequest, _ chan<- event) (SpawnResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.SpawnErr != nil {
		return SpawnResult{}, f.SpawnErr
	}
	f.spawns = append(f.spawns, req)
	f.nextPid++
	pid := f.nextPid

	// Build a *exec.Cmd that downstream code can interrogate without
	// actually running anything. We hand it a fake Process whose Pid field
	// is the one we just generated. Process.Signal will fail because there's
	// no real process, but we never call cmd.Process.Signal in tests — Signal
	// goes through Runtime.Signal which is intercepted by this fake.
	cmd := exec.Command("/bin/true")
	cmd.Process = &fakeOSProcess(pid).Process
	return SpawnResult{Cmd: cmd}, nil
}

func (f *fakeRuntime) Signal(cmd *exec.Cmd, sig syscall.Signal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	pid := 0
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	f.signals = append(f.signals, signalCall{Pid: pid, Sig: sig})
	return f.SignalErr
}

// Spawns returns the recorded SpawnRequests (defensive copy).
func (f *fakeRuntime) Spawns() []SpawnRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]SpawnRequest, len(f.spawns))
	copy(out, f.spawns)
	return out
}

// Signals returns the recorded Signal calls (defensive copy).
func (f *fakeRuntime) Signals() []signalCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]signalCall, len(f.signals))
	copy(out, f.signals)
	return out
}

func (f *fakeRuntime) lastSignal() (signalCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.signals) == 0 {
		return signalCall{}, false
	}
	return f.signals[len(f.signals)-1], true
}
