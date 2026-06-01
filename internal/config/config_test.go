package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_AppliesDefaultsForMissingFields(t *testing.T) {
	// Only `cmd` provided; every other field should fall back to its default.
	path := writeYAML(t, `
programs:
  echo:
    cmd: "/bin/echo hi"
`)
	loader := NewLoader(Options{})
	cfg, err := loader.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, ok := cfg.Programs["echo"]
	if !ok {
		t.Fatal("program 'echo' not found in loaded config")
	}
	if p.Numprocs != 1 {
		t.Errorf("Numprocs default = %d, want 1", p.Numprocs)
	}
	if p.Umask != 022 {
		t.Errorf("Umask default = %04o, want 0022", p.Umask)
	}
	if p.Autostart != true {
		t.Errorf("Autostart default = %v, want true", p.Autostart)
	}
	if p.Autorestart != RestartUnexpected {
		t.Errorf("Autorestart default = %q, want unexpected", p.Autorestart)
	}
	if p.Startretries != 3 {
		t.Errorf("Startretries default = %d, want 3", p.Startretries)
	}
	if p.Starttime != 5 {
		t.Errorf("Starttime default = %d, want 5", p.Starttime)
	}
	if p.Stopsignal != "TERM" {
		t.Errorf("Stopsignal default = %q, want TERM", p.Stopsignal)
	}
	if p.Stoptime != 10 {
		t.Errorf("Stoptime default = %d, want 10", p.Stoptime)
	}
	if p.Stdout != "/dev/null" {
		t.Errorf("Stdout default = %q, want /dev/null", p.Stdout)
	}
	if p.Stderr != "/dev/null" {
		t.Errorf("Stderr default = %q, want /dev/null", p.Stderr)
	}
	if len(p.Exitcodes) != 1 || p.Exitcodes[0] != 0 {
		t.Errorf("Exitcodes default = %v, want [0]", p.Exitcodes)
	}
}

func TestLoad_RejectsEmptyConfig(t *testing.T) {
	path := writeYAML(t, "programs: {}\n")
	loader := NewLoader(Options{})
	if _, err := loader.Load(path); err == nil {
		t.Fatal("expected error for empty programs map, got nil")
	}
}

func TestLoad_RejectsBadStopsignal(t *testing.T) {
	path := writeYAML(t, `
programs:
  bad:
    cmd: "/bin/echo hi"
    stopsignal: WAT
`)
	loader := NewLoader(Options{})
	if _, err := loader.Load(path); err == nil {
		t.Fatal("expected error for invalid signal, got nil")
	}
}

func TestLoad_RejectsRelativeCmd(t *testing.T) {
	path := writeYAML(t, `
programs:
  rel:
    cmd: "echo hi"
`)
	loader := NewLoader(Options{})
	if _, err := loader.Load(path); err == nil {
		t.Fatal("expected error for relative cmd path, got nil")
	}
}
