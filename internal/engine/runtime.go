package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"taskmaster/internal/fileutil"
	"time"
)

// Runtime is the OS boundary. Manager depends on this interface so the state
// machine can be unit-tested with a fake. Real implementation: osRuntime.
type Runtime interface {
	// SpawnProcess forks/execs a child according to req and returns a result
	// carrying the *exec.Cmd handle. It also starts a goroutine that blocks
	// on cmd.Wait() and, when the child exits, sends a ProcExited event on
	// the given channel (unless ctx is cancelled first).
	//
	// The goroutine owns the lifetime of the stdout/stderr files: it closes
	// them via defer when it exits.
	SpawnProcess(ctx context.Context, req SpawnRequest, events chan<- event) (SpawnResult, error)

	// Signal sends sig to the process represented by cmd. Goes through
	// (*os.Process).Signal so a reaped process safely returns
	// os.ErrProcessDone rather than racing pid reuse.
	Signal(cmd *exec.Cmd, sig syscall.Signal) error
}

// SpawnRequest captures everything the OS layer needs to fork a child.
// Plain value type — no pointers into engine state.
type SpawnRequest struct {
	Name       string
	Index      int
	Bin        string
	Args       []string
	Umask      int
	Workingdir string
	StdoutPath string
	StderrPath string
	Env        []string
}

// SpawnResult is what SpawnProcess hands back to Manager so it can populate
// the corresponding Process.
type SpawnResult struct {
	Cmd       *exec.Cmd
	StartedAt time.Time
}

// osRuntime is the production Runtime that actually forks processes.
type osRuntime struct {
	log *slog.Logger
}

// NewOsRuntime returns the production Runtime.
func NewOsRuntime(log *slog.Logger) Runtime {
	return &osRuntime{log: log}
}

func (r *osRuntime) SpawnProcess(ctx context.Context, req SpawnRequest, events chan<- event) (SpawnResult, error) {
	stdout, err := fileutil.OpenOutput(req.StdoutPath)
	if err != nil {
		return SpawnResult{}, fmt.Errorf("open stdout %q: %w", req.StdoutPath, err)
	}
	stderr, err := fileutil.OpenOutput(req.StderrPath)
	if err != nil {
		stdout.Close()
		return SpawnResult{}, fmt.Errorf("open stderr %q: %w", req.StderrPath, err)
	}

	cmd := exec.Command(req.Bin, req.Args...)
	cmd.Dir = req.Workingdir
	cmd.Env = req.Env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Apply umask around Start. umask is process-global on Linux but Start
	// is serialized by the event-loop goroutine (Manager.spawn runs in a
	// handler), so this is race-free in our design.
	oldUmask := syscall.Umask(req.Umask)
	err = cmd.Start()
	syscall.Umask(oldUmask)
	if err != nil {
		stdout.Close()
		stderr.Close()
		return SpawnResult{}, fmt.Errorf("exec %s: %w", req.Bin, err)
	}

	startedAt := time.Now()
	r.log.Info("process started",
		"program", req.Name, "index", req.Index, "pid", cmd.Process.Pid, "bin", req.Bin)

	// Wait-goroutine. Owns the file handles via defer. Sends ProcExited
	// when the child terminates; ctx guard prevents leak if Manager is gone.
	go func() {
		defer stdout.Close()
		defer stderr.Close()
		waitErr := cmd.Wait()
		select {
		case events <- ProcExited{Name: req.Name, Index: req.Index, Status: waitErr}:
		case <-ctx.Done():
		}
	}()

	return SpawnResult{Cmd: cmd, StartedAt: startedAt}, nil
}

func (r *osRuntime) Signal(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return errors.New("signal: no process")
	}
	if err := cmd.Process.Signal(sig); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil // already reaped — fine, target of signal is gone
		}
		return err
	}
	return nil
}
