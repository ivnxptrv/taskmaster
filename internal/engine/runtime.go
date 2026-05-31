package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"taskmaster/internal/fileutil"
)

type Runtime interface {
	SpawnProcess(ctx context.Context, exitedInbox chan<- event, req SpawnRequest) (SpawnResult, error)
	Signal(cmd *exec.Cmd, sig syscall.Signal) error
}

type SpawnRequest struct {
	name       string
	index      uint8
	bin        string
	args       []string
	umask      int
	workingdir string
	stdoutPath string
	stderrPath string
	env        []string
}

type SpawnResult struct {
	cmd        *exec.Cmd
	stdoutFile *os.File
	stderrFile *os.File
}

type osRuntime struct {
	log *slog.Logger
}

func NewOsRuntime(log *slog.Logger) *osRuntime {
	return &osRuntime{log: log}
}

func (r *osRuntime) SpawnProcess(ctx context.Context, exitedInbox chan<- event, req SpawnRequest) (SpawnResult, error) {
	var err error

	cmd := exec.Command(req.bin, req.args...)
	cmd.Dir = req.workingdir
	cmd.Env = req.env

	stdout, err := fileutil.OpenOutput(req.stdoutPath)
	if err != nil {
		return SpawnResult{}, err
	}
	cmd.Stdout = stdout

	stderr, err := fileutil.OpenOutput(req.stderrPath)
	if err != nil {
		stdout.Close()
		return SpawnResult{}, err
	}
	cmd.Stderr = stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// apply umask and spawn
	oldUmask := syscall.Umask(req.umask)
	err = cmd.Start() // fork
	syscall.Umask(oldUmask)
	if err != nil {
		stderr.Close()
		stdout.Close()
		return SpawnResult{}, err
	}

	// spawn listener for child exit
	go func() {
		defer stdout.Close()
		defer stderr.Close()
		err := cmd.Wait()
		select {
		case exitedInbox <- ProcExited{Name: req.name, Index: req.index, Status: err}:
		case <-ctx.Done():
			return
		}
	}()

	return SpawnResult{
		cmd:        cmd,
		stdoutFile: stdout,
		stderrFile: stderr,
	}, nil

}

func (r *osRuntime) Signal(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil {
		return fmt.Errorf("cmd is nil")
	}
	if cmd.Process == nil {
		return fmt.Errorf("process is not started")
	}
	if sig == syscall.SIGKILL {
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
	}
	err := cmd.Process.Signal(syscall.SIGKILL)

	if err != nil {
		return err
	}
}
