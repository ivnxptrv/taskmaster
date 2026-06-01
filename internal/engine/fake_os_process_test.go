package engine

import "os"

// fakeOSProcess wraps an os.Process so tests can construct one with a
// chosen Pid without going through real OS fork/exec. Direct construction
// of os.Process via &os.Process{Pid: ...} is valid Go — the zero value of
// its other fields is harmless as long as we never call .Signal/.Wait on it
// (we don't; Runtime.Signal is faked).
type fakeProcess struct {
	os.Process
}

func fakeOSProcess(pid int) *fakeProcess {
	return &fakeProcess{Process: os.Process{Pid: pid}}
}
