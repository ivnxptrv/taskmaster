package engine

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestOsRuntimeSpawnsAndReportsExit(t *testing.T) {
	tmp := t.TempDir()
	stdoutPath := filepath.Join(tmp, "out.log")
	stderrPath := filepath.Join(tmp, "err.log")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := make(chan event, 1)
	rt := &osRuntime{}

	res, err := rt.SpawnProcess(ctx, events, SpawnRequest{
		name:       "test",
		index:      0,
		bin:        "/bin/echo",
		args:       []string{"hello world"},
		umask:      022,
		workingdir: tmp,
		stdoutPath: stdoutPath,
		stderrPath: stderrPath,
		env:        os.Environ(),
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if res.cmd == nil {
		t.Fatal("nil cmd in result")
	}

	// Wait for the ProcExited event the wait-goroutine sends.
	select {
	case e := <-events:
		pe, ok := e.(ProcExited)
		if !ok {
			t.Fatalf("got %T, want ProcExited", e)
		}
		if pe.Status != nil {
			t.Fatalf("status = %v, want nil", pe.Status)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for ProcExited")
	}

	// Verify echo wrote to the file.
	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world\n" {
		t.Fatalf("stdout = %q", string(data))
	}
}
