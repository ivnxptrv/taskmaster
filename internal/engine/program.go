package engine

import (
	"log/slog"
	"strings"
	"syscall"
	"taskmaster/internal/config"
	"taskmaster/internal/fileutil"
	"time"
)

// -----------------------------------------------

type Bin string

func (b *Bin) validate() error {
	var err error

	p := string(*b)
	if err = fileutil.FileExists(p); err != nil {
		return err
	}
	if err = fileutil.FileExecutable(p); err != nil {
		return err
	}
	return nil
}

func cmdToBin(c config.Cmd) Bin {
	bin, _, _ := strings.Cut(string(c), " ")
	return Bin(bin)
}

func cmdToArgs(c config.Cmd) []string {
	parts := strings.Fields(string(c))
	return parts[1:]
}

// -----------------------------------------------

type Workingdir string

func (wd *Workingdir) validate() error {
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

func (oS *outStream) validate() error {
	var err error

	p := string(*oS)
	if err = fileutil.FileExists(p); err == nil {
		if err = fileutil.FileWritable(p); err != nil {
			return err
		}
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

func buildEnv(env config.Env, environ map[string]string) []string {
	resultMap := make(map[string]string)
	for k, v := range environ {
		resultMap[k] = v
	}

	if env != nil {
		for k, v := range env {
			resultMap[k] = v
		}
	}

	result := make([]string, 0, len(resultMap))
	for k, v := range resultMap {
		result = append(result, k+"="+v)
	}
	return result

}

// -----------------------------------------------

type Spec struct {
	name         string
	bin          Bin // validate
	args         []string
	numprocs     uint8
	umask        int
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
}

// -----------------------------------------------

type Program struct {
	spec  Spec       // desired
	procs []*Process // current
}

func (s *Spec) validate() error {
	var err error

	if err = s.bin.validate(); err != nil {
		return err
	}

	if err = s.workingdir.validate(); err != nil {
		return err
	}

	if err = s.stdout.validate(); err != nil {
		return err
	}

	if err = s.stderr.validate(); err != nil {
		return err
	}

	return nil
}

func (prg *Program) getProcess(index int) (*Process, bool) {
	if index >= len(prg.procs) {
		return nil, false
	}
	return prg.procs[index], true
}

func (prg *Program) delProcess(proc *Process) {
	if prg == nil {
		return
	}
	prg.procs = append(prg.procs[:proc.index], prg.procs[proc.index+1:]...)
}

func newProgram(nameP string, cfgP *config.Program, environ map[string]string) *Program {
	num := uint8(cfgP.Numprocs)

	prg := &Program{
		spec: Spec{
			name:         nameP,
			bin:          cmdToBin(cfgP.Cmd),
			args:         cmdToArgs(cfgP.Cmd),
			numprocs:     num,
			umask:        int(cfgP.Umask),
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
			env:          buildEnv(cfgP.Env, environ),
		},
		procs: make([]*Process, 0, num),
	}

	// for i := range num {
	// 	p := newProcess(i)
	// 	prg.procs = append(prg.procs, p)
	// }
	return prg
}
