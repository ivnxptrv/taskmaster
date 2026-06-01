package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// -----------------------------------------------

// config field
type Field interface {
	setDefault()
	validate() error
}

// -----------------------------------------------

type Cmd string

func (c *Cmd) setDefault() {
	*c = ""
}

func (c Cmd) validate() error {
	p := string(c)
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains invalid characters")
	}

	if !filepath.IsAbs(p) {
		return fmt.Errorf("path '%s' must be an absolute path", p)
	}
	return nil
}

// -----------------------------------------------

type Numprocs uint8

func (n *Numprocs) setDefault() {
	*n = 1
}

func (n Numprocs) validate() error {
	return nil
}

// func (n *Numprocs) UnmarshalYAML(value *yaml.Node) error {
// 	if n > 25
// 	type Alias Program
// 	return value.Decode((*Alias)(p))
// }

// -----------------------------------------------

type Umask uint32

func (u *Umask) setDefault() {
	*u = 022
}

func (u Umask) validate() error {
	if u > 0777 {
		return fmt.Errorf("invalid umask: %s", u)
	}
	return nil
}

func (u Umask) String() string {
	return fmt.Sprintf("%04o", u)
}
func (u Umask) MarshalYAML() (any, error) {
	return fmt.Sprintf("%04o", u), nil
}

// -----------------------------------------------

type Workingdir string

func (wd *Workingdir) setDefault() {
	*wd = "./"
}

func (wd Workingdir) validate() error {
	p := string(wd)
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains invalid characters")
	}
	return nil
}

// -----------------------------------------------

type Autostart bool

func (as *Autostart) setDefault() {
	*as = true
}

func (as Autostart) validate() error {
	return nil
}

// -----------------------------------------------

type Autorestart string

const (
	RestartUnexpected Autorestart = "unexpected"
	RestartAlways     Autorestart = "always"
	RestartNever      Autorestart = "never"
)

func (r Autorestart) validate() error {
	switch r {
	case RestartUnexpected, RestartAlways, RestartNever:
		return nil
	default:
		return fmt.Errorf("invalid restart policy: %s", r)
	}
}

func (r *Autorestart) setDefault() {
	*r = RestartUnexpected
}

// -----------------------------------------------

type Exitcodes []int

func (ec *Exitcodes) setDefault() {
	*ec = []int{0}
}

func (as Exitcodes) validate() error {
	for c := range as {
		if c < 0 || c > 255 {
			return fmt.Errorf("ivalid exitcode")
		}
	}
	return nil
}

// -----------------------------------------------

type Startretries uint16

func (sr *Startretries) setDefault() {
	*sr = 3
}

func (sr Startretries) validate() error {
	return nil
}

// -----------------------------------------------

type Starttime uint32

func (st *Starttime) setDefault() {
	*st = 5
}

func (st Starttime) validate() error {
	return nil
}

// -----------------------------------------------

type Stopsignal string

var SupportedSignals = map[string]struct{}{
	"HUP": {}, "INT": {}, "QUIT": {}, "ILL": {}, "TRAP": {},
	"ABRT": {}, "BUS": {}, "FPE": {}, "KILL": {}, "USR1": {},
	"SEGV": {}, "USR2": {}, "PIPE": {}, "ALRM": {}, "TERM": {},
	"CHLD": {}, "CONT": {}, "STOP": {}, "TSTP": {}, "TTIN": {},
	"TTOU": {},
}

func (ss *Stopsignal) setDefault() {
	*ss = "TERM"
}

func (ss Stopsignal) validate() error {
	if _, exists := SupportedSignals[string(ss)]; exists {
		return nil
	}
	return fmt.Errorf("invalid signal: %s", ss)
}

// -----------------------------------------------

type Stoptime uint32

func (st *Stoptime) setDefault() {
	*st = 10
}

func (st Stoptime) validate() error {
	return nil
}

// -----------------------------------------------

type Stdout string

func (s *Stdout) setDefault() {
	*s = "/dev/null"
}

func (s Stdout) validate() error {
	p := string(s)
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains invalid characters")
	}
	return nil
}

// -----------------------------------------------

type Stderr string

func (s *Stderr) setDefault() {
	*s = "/dev/null"
}

func (s Stderr) validate() error {
	p := string(s)
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains invalid characters")
	}
	return nil
}

// -----------------------------------------------

type Env map[string]string

func (e *Env) setDefault() {
	*e = make(map[string]string)
}

func (e Env) validate() error {
	// return fmt.Errorf("invalid env")
	return nil
}

// -----------------------------------------------
