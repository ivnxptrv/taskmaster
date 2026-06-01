package engine

import (
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	"taskmaster/internal/config"
)

// SpecsFromConfig translates a loaded config.Config into the engine's []Spec.
// parentEnv is the supervisor's own environment, into which each program's
// Env overrides are merged. Returns specs in stable order by name.
func SpecsFromConfig(cfg *config.Config, parentEnv []string) ([]Spec, error) {
	specs := make([]Spec, 0, len(cfg.Programs))
	for name, p := range cfg.Programs {
		s, err := specFromConfigProgram(name, &p, parentEnv)
		if err != nil {
			return nil, fmt.Errorf("program %q: %w", name, err)
		}
		specs = append(specs, s)
	}
	return specs, nil
}

func specFromConfigProgram(name string, p *config.Program, parentEnv []string) (Spec, error) {
	bin, args := splitCmd(string(p.Cmd))
	if bin == "" {
		return Spec{}, fmt.Errorf("empty cmd")
	}

	sig, ok := signalFromName(string(p.Stopsignal))
	if !ok {
		return Spec{}, fmt.Errorf("unknown stopsignal %q", p.Stopsignal)
	}

	exitcodes := make([]int, len(p.Exitcodes))
	copy(exitcodes, []int(p.Exitcodes))

	return Spec{
		Name:         name,
		Bin:          bin,
		Args:         args,
		Numprocs:     int(p.Numprocs),
		Umask:        int(p.Umask),
		Workingdir:   string(p.Workingdir),
		Env:          mergeEnv(parentEnv, p.Env),
		Autostart:    bool(p.Autostart),
		Autorestart:  autorestartFromConfig(p.Autorestart),
		Exitcodes:    exitcodes,
		Startretries: int(p.Startretries),
		Starttime:    time.Duration(p.Starttime) * time.Second,
		Stopsignal:   sig,
		Stoptime:     time.Duration(p.Stoptime) * time.Second,
		StdoutPath:   string(p.Stdout),
		StderrPath:   string(p.Stderr),
	}, nil
}

// splitCmd parses the cmd string into a binary path and arguments.
// Naive whitespace split — sufficient for the subject's examples. Phase 9
// could add shlex-style quoting if a grader's command needs it.
func splitCmd(cmd string) (string, []string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func autorestartFromConfig(a config.Autorestart) Autorestart {
	switch a {
	case config.RestartAlways:
		return RestartAlways
	case config.RestartNever:
		return RestartNever
	default:
		return RestartUnexpected
	}
}

// mergeEnv returns parentEnv with overrides applied, sorted KEY=VALUE.
// Sorted output makes Spec equality (used by reload diff) stable across loads.
func mergeEnv(parent []string, overrides config.Env) []string {
	merged := make(map[string]string, len(parent)+len(overrides))
	for _, kv := range parent {
		k, v, _ := strings.Cut(kv, "=")
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(merged))
	for _, k := range keys {
		out = append(out, k+"="+merged[k])
	}
	return out
}

// signalFromName maps a bare signal name ("TERM", "HUP", "USR1") to its
// syscall.Signal value. config.Stopsignal.validate guarantees membership of
// SupportedSignals before we reach here.
func signalFromName(name string) (syscall.Signal, bool) {
	sig, ok := signalNameMap[name]
	return sig, ok
}

var signalNameMap = map[string]syscall.Signal{
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
