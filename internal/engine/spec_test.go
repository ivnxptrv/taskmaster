package engine

import (
	"syscall"
	"testing"
	"time"
)

func baseSpec() Spec {
	return Spec{
		Name:         "x",
		Bin:          "/bin/true",
		Args:         []string{"a", "b"},
		Numprocs:     2,
		Umask:        022,
		Workingdir:   "/tmp",
		Env:          []string{"K=V"},
		Autostart:    true,
		Autorestart:  RestartUnexpected,
		Exitcodes:    []int{0, 2},
		Startretries: 3,
		Starttime:    time.Second,
		Stopsignal:   syscall.SIGTERM,
		Stoptime:     5 * time.Second,
		StdoutPath:   "/tmp/o",
		StderrPath:   "/tmp/e",
	}
}

func TestSpecEqual_Identical(t *testing.T) {
	a, b := baseSpec(), baseSpec()
	if !a.Equal(b) {
		t.Fatal("identical specs should be Equal")
	}
}

func TestSpecEqual_DifferenceDetected(t *testing.T) {
	type mut func(s *Spec)
	cases := map[string]mut{
		"Bin":          func(s *Spec) { s.Bin = "/bin/false" },
		"Args":         func(s *Spec) { s.Args = []string{"x"} },
		"Numprocs":     func(s *Spec) { s.Numprocs = 4 },
		"Umask":        func(s *Spec) { s.Umask = 077 },
		"Workingdir":   func(s *Spec) { s.Workingdir = "/other" },
		"Env":          func(s *Spec) { s.Env = []string{"K=V2"} },
		"Autostart":    func(s *Spec) { s.Autostart = false },
		"Autorestart":  func(s *Spec) { s.Autorestart = RestartAlways },
		"Exitcodes":    func(s *Spec) { s.Exitcodes = []int{0} },
		"Startretries": func(s *Spec) { s.Startretries = 9 },
		"Starttime":    func(s *Spec) { s.Starttime = 2 * time.Second },
		"Stopsignal":   func(s *Spec) { s.Stopsignal = syscall.SIGUSR1 },
		"Stoptime":     func(s *Spec) { s.Stoptime = time.Second },
		"StdoutPath":   func(s *Spec) { s.StdoutPath = "/other" },
		"StderrPath":   func(s *Spec) { s.StderrPath = "/other" },
	}
	for field, m := range cases {
		a := baseSpec()
		b := baseSpec()
		m(&b)
		if a.Equal(b) {
			t.Errorf("spec change to %s should not be Equal", field)
		}
	}
}
