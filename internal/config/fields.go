package config

import (
	"fmt"
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

// -----------------------------------------------

type Umask uint32

func (u *Umask) setDefault() {
	*u = 022
}

func (u Umask) validate() error {
	return nil
}

func (u Umask) String() string {
	return fmt.Sprintf("%03o", u)
}
func (u Umask) MarshalYAML() (any, error) {
	return fmt.Sprintf("%03o", u), nil
}

// -----------------------------------------------

type Workingdir string

func (wd *Workingdir) setDefault() {
	*wd = "./"
}

func (wd Workingdir) validate() error {
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

type Exitcodes []uint8

func (ec *Exitcodes) setDefault() {
	*ec = []uint8{0}
}

func (as Exitcodes) validate() error {
	return nil
}

// -----------------------------------------------

type Startretries uint32

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

const (
	SigTerm Stopsignal = "TERM"
)

func (ss *Stopsignal) setDefault() {
	*ss = SigTerm
}

func (ss Stopsignal) validate() error {
	switch ss {
	case SigTerm:
		return nil
	default:
		return fmt.Errorf("invalid signal: %s", ss)
	}
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
	*s = ""
}

func (s Stdout) validate() error {
	return nil
}

// -----------------------------------------------

type Stderr string

func (s *Stderr) setDefault() {
	*s = ""
}

func (s Stderr) validate() error {
	return nil
}

// -----------------------------------------------

type Env map[string]string

func (e *Env) setDefault() {
	*e = make(map[string]string)
}

func (e Env) validate() error {
	return nil
}

// -----------------------------------------------
