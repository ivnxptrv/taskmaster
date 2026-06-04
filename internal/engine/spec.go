package engine

import (
	"slices"
	"syscall"
	"time"
)

// Autorestart policy. Phase 3 wires the semantics; Phase 1 just carries it.
type Autorestart int

const (
	RestartUnexpected Autorestart = iota
	RestartAlways
	RestartNever
)

func (a Autorestart) String() string {
	switch a {
	case RestartAlways:
		return "always"
	case RestartNever:
		return "never"
	default:
		return "unexpected"
	}
}

// Spec is the *immutable desired state* of a program, derived from config.
// Reload diffs two Specs to decide whether running processes need to be
// restarted. Keep it value-comparable where possible (slices are compared
// element-wise in spec.go's equal method, added in Phase 6).
type Spec struct {
	Name string

	Bin  string
	Args []string

	Numprocs   int
	Umask      int
	Workingdir string
	Env        []string // KEY=VALUE pairs, fully resolved (parent env + overrides)

	Autostart   bool
	Autorestart Autorestart
	Exitcodes   []int

	Startretries int
	Starttime    time.Duration

	Stopsignal syscall.Signal
	Stoptime   time.Duration

	StdoutPath string
	StderrPath string
}

// Equal returns true iff every field of two Specs is identical. Used by the
// reload diff to decide whether a program's running processes need to be
// restarted. Env is compared element-wise; SpecsFromConfig always emits Env
// in a canonical sorted order so order-only changes don't trigger restarts.
func (s Spec) Equal(o Spec) bool {
	return s.Name == o.Name &&
		s.Bin == o.Bin &&
		slices.Equal(s.Args, o.Args) &&
		s.Numprocs == o.Numprocs &&
		s.Umask == o.Umask &&
		s.Workingdir == o.Workingdir &&
		slices.Equal(s.Env, o.Env) &&
		s.Autostart == o.Autostart &&
		s.Autorestart == o.Autorestart &&
		slices.Equal(s.Exitcodes, o.Exitcodes) &&
		s.Startretries == o.Startretries &&
		s.Starttime == o.Starttime &&
		s.Stopsignal == o.Stopsignal &&
		s.Stoptime == o.Stoptime &&
		s.StdoutPath == o.StdoutPath &&
		s.StderrPath == o.StderrPath
}

// equalIgnoringNumprocs returns true iff s and o are Equal modulo Numprocs.
// Used by the reload diff so that a pure numprocs change only resizes the
// pool (grow → spawn new slots, shrink → stop surplus) without restarting
// the processes that survive the change.
func (s Spec) equalIgnoringNumprocs(o Spec) bool {
	s.Numprocs = o.Numprocs
	return s.Equal(o)
}
