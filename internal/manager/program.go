package manager

import (
	"os"
	"os/exec"
	"syscall"
	"taskmaster/internal/config"
	"taskmaster/internal/fileutil"
	"time"
)

// -----------------------------------------------

type Cmd string

func (c *Cmd) valdiate() error {
	var err error

	p := string(*c)
	if err = fileutil.FileExists(p); err != nil {
		return err
	}
	if err = fileutil.FileExecutable(p); err != nil {
		return err
	}
	return nil
}

func (c *Cmd) String() string {
	return string(*c)
}

// -----------------------------------------------

type Workingdir string

func (wd *Workingdir) valdiate() error {
	var err error

	p := string(*wd)
	if err = fileutil.DirExists(p); err != nil {
		return err
	}
	if err = fileutil.DirExecutable(p); err != nil {
		return err
	}
	return nil
}

func (wd *Workingdir) String() string {
	return string(*wd)
}

// -----------------------------------------------

type outStream string

func (oS *outStream) valdiate() error {
	var err error

	p := string(*oS)
	if err = fileutil.FileExists(p); err != nil {
		return err
	}
	if err = fileutil.FileWritable(p); err != nil {
		return err
	}
	return nil
}

func (oS *outStream) String() string {
	return string(*oS)
}

// -----------------------------------------------

// -----------------------------------------------

type Autorestart uint8

const (
	RestartUnexpected Autorestart = iota
	RestartAlways
	RestartNever
)

// -----------------------------------------------

type Program struct {
	name string

	cmd          Cmd // validate
	numprocs     uint8
	umask        uint32
	workingdir   Workingdir // validate
	autostart    bool
	autorestart  Autorestart
	exitcodes    []uint8
	startretries uint16
	starttime    time.Duration
	stopsignal   syscall.Signal
	stoptime     time.Duration
	stdout       outStream //validate
	stderr       outStream //validate
	env          []string  // KEY=VALUE

	procs []*Process
}

func (p *Program) validate() error {
	var err error

	if err = p.cmd.valdiate(); err != nil {
		return err
	}

	if err = p.workingdir.valdiate(); err != nil {
		return err
	}

	if err = p.stdout.valdiate(); err != nil {
		return err
	}

	if err = p.stderr.valdiate(); err != nil {
		return err
	}

	return nil
}

func newProgram(nameP string, cfgP *config.Program) (*Program, error) {
	p := &Program{
		Name: nameP,
	}
	return nil, nil
}

func validateCmd(filepath string) error {

}
