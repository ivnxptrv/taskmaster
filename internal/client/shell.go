// Package client implements the interactive control shell.
package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

// Backend is everything the shell needs from the engine.
// *engine.Manager implicitly satisfies this interface — the engine package
// does NOT import this one, so the interface lives with its consumer.
type Backend interface {
	Start(name string, index int) error
	Stop(name string, index int) error
	Restart(name string, index int) error
	Status() []ProcInfo
	Reload() error
	Shutdown()
}

// ProcInfo mirrors engine.ProcInfo. We re-declare it here so the client
// package doesn't import engine. Backend implementations convert.
type ProcInfo struct {
	Name      string
	Index     int
	State     string
	Pid       int
	StartedAt time.Time
	Retries   int
}

// Shell owns the readline loop and dispatches commands to the Backend.
type Shell struct {
	backend Backend
	log     *slog.Logger
}

func NewShell(backend Backend, log *slog.Logger) *Shell {
	return &Shell{backend: backend, log: log}
}

// Run drives the readline loop until ctx is cancelled, the user types exit,
// or Ctrl-D is pressed. Returns nil on clean exit.
func (s *Shell) Run(ctx context.Context) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32mtaskmaster> \033[0m",
		HistoryFile:     "/tmp/taskmaster.history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("readline init: %w", err)
	}
	defer rl.Close()

	// Watcher that closes readline when ctx is cancelled so Readline() returns.
	go func() {
		<-ctx.Done()
		_ = rl.Close()
	}()

	fmt.Fprintln(rl.Stdout(), "taskmaster shell. Type 'help' for commands.")
	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue // Ctrl-C clears the line; only Ctrl-D / 'exit' leaves.
			}
			if err == io.EOF {
				return nil
			}
			return nil // ctx-driven close also lands here
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if exit := s.dispatch(rl.Stdout(), line); exit {
			return nil
		}
	}
}

// dispatch returns true if the user asked to exit.
func (s *Shell) dispatch(out io.Writer, line string) bool {
	parts := strings.Fields(line)
	cmd, args := parts[0], parts[1:]

	switch cmd {
	case "help", "?":
		printHelp(out)
	case "status":
		printStatus(out, s.backend.Status())
	case "start":
		s.runTarget(out, args, s.backend.Start, "start")
	case "stop":
		s.runTarget(out, args, s.backend.Stop, "stop")
	case "restart":
		s.runTarget(out, args, s.backend.Restart, "restart")
	case "reload":
		if err := s.backend.Reload(); err != nil {
			fmt.Fprintf(out, "reload: %v\n", err)
		} else {
			fmt.Fprintln(out, "reload ok")
		}
	case "shutdown":
		fmt.Fprintln(out, "shutting down...")
		s.backend.Shutdown()
		return true
	case "exit", "quit":
		return true
	default:
		fmt.Fprintf(out, "unknown command: %s (try 'help')\n", cmd)
	}
	return false
}

// runTarget handles the start/stop/restart family: parses "<name> [index]"
// (no index means "all") and dispatches via fn.
func (s *Shell) runTarget(out io.Writer, args []string, fn func(string, int) error, label string) {
	if len(args) == 0 {
		fmt.Fprintf(out, "usage: %s <name> [index]\n", label)
		return
	}
	name := args[0]
	index := -1
	if len(args) >= 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(out, "bad index %q: %v\n", args[1], err)
			return
		}
		index = n
	}
	if err := fn(name, index); err != nil {
		fmt.Fprintf(out, "%s: %v\n", label, err)
	} else {
		fmt.Fprintf(out, "%s %s ok\n", label, name)
	}
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  status                          list all processes")
	fmt.Fprintln(out, "  start    <name> [index]         start (all procs or one)")
	fmt.Fprintln(out, "  stop     <name> [index]         stop")
	fmt.Fprintln(out, "  restart  <name> [index]         restart")
	fmt.Fprintln(out, "  reload                          reload config (SIGHUP)")
	fmt.Fprintln(out, "  shutdown                        stop everything and exit")
	fmt.Fprintln(out, "  exit | quit                     leave shell (taskmaster keeps running)")
}

func printStatus(out io.Writer, infos []ProcInfo) {
	if len(infos) == 0 {
		fmt.Fprintln(out, "no programs")
		return
	}
	fmt.Fprintf(out, "%-20s %-5s %-12s %-7s %-12s %-7s\n",
		"NAME", "INDEX", "STATE", "PID", "UPTIME", "RETRIES")
	for _, p := range infos {
		uptime := "-"
		if !p.StartedAt.IsZero() {
			uptime = time.Since(p.StartedAt).Truncate(time.Second).String()
		}
		pid := "-"
		if p.Pid > 0 {
			pid = strconv.Itoa(p.Pid)
		}
		fmt.Fprintf(out, "%-20s %-5d %-12s %-7s %-12s %-7d\n",
			p.Name, p.Index, p.State, pid, uptime, p.Retries)
	}
}
