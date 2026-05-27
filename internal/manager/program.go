package manager

import (
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

type Autorestart uint8

const (
	RestartUnexpected Autorestart = iota
	RestartAlways
	RestartNever
)

func AutorestartToEnum(r config.Autorestart) Autorestart {
	switch r {
	case config.RestartUnexpected:
		return RestartUnexpected
	case config.RestartAlways:
		return RestartAlways
	case config.RestartNever:
		return RestartNever
	}
	return RestartUnexpected
}

// -----------------------------------------------

var stringSignalToSyscall = map[string]syscall.Signal{
	"HUP":  syscall.SIGHUP,
	"INT":  syscall.SIGINT,
	"QUIT": syscall.SIGQUIT,
	"ILL":  syscall.SIGILL,
	"TRAP": syscall.SIGTRAP,
	"ABRT": syscall.SIGABRT,
	"BUS":  syscall.SIGBUS,
	"FPE":  syscall.SIGFPE,
	"KILL": syscall.SIGKILL,
	"USR1": syscall.SIGUSR1,
	"SEGV": syscall.SIGSEGV,
	"USR2": syscall.SIGUSR2,
	"PIPE": syscall.SIGPIPE,
	"ALRM": syscall.SIGALRM,
	"TERM": syscall.SIGTERM,
	"CHLD": syscall.SIGCHLD,
	"CONT": syscall.SIGCONT,
	"STOP": syscall.SIGSTOP,
	"TSTP": syscall.SIGTSTP,
	"TTIN": syscall.SIGTTIN,
	"TTOU": syscall.SIGTTOU,
}

// -----------------------------------------------

func envToSlice(env config.Env) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

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

func newProgram(nameP string, cfgP *config.Program) *Program {
	num := uint8(cfgP.Numprocs)

	return &Program{
		name:         nameP,
		cmd:          Cmd(cfgP.Cmd),
		numprocs:     num,
		umask:        uint32(cfgP.Umask),
		workingdir:   Workingdir(cfgP.Workingdir),
		autostart:    bool(cfgP.Autostart),
		autorestart:  AutorestartToEnum(cfgP.Autorestart),
		exitcodes:    []uint8(cfgP.Exitcodes),
		startretries: uint16(cfgP.Startretries),
		starttime:    time.Duration(cfgP.Starttime),
		stopsignal:   stringSignalToSyscall[string(cfgP.Stopsignal)],
		stoptime:     time.Duration(cfgP.Stoptime),
		stdout:       outStream(cfgP.Stdout),
		stderr:       outStream(cfgP.Stderr),
		env:          envToSlice(cfgP.Env),
		procs:        make([]*Process, 0, num),
	}
}
