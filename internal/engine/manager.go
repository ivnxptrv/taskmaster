package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"syscall"
	"time"
)

// Manager is the supervisor's actor: a single goroutine owns every
// Program/Process mutation. External goroutines (shell, signal watchers,
// timers, wait-goroutines) only ever send events into m.events; the loop
// drains them serially.
//
// Lifecycle:
//
//	NewManager → Run(parent)
//	  ├─ shutdownCtx (alive — gates timers and wait-goroutine sends)
//	  ├─ autostart
//	  ├─ serve(parent)   blocks until parent ctx OR internal exitSignal
//	  ├─ gracefulDrain   SIGTERM live procs, drain ProcExited up to deadline
//	  └─ shutdownCancel  releases timer/wait goroutines
type Manager struct {
	log     *slog.Logger
	runtime Runtime

	programs map[string]*Program

	events chan event

	// shutdownCtx lives for the entire Run; it gates timer callbacks and
	// the wait-goroutine's "is anyone reading?" guard. Only cancelled by
	// the defer at the END of Run, AFTER gracefulDrain returns.
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	// exitSignal is closed by shutdownCmd or by an external ctx cancellation
	// to tell serve() to stop accepting new events and proceed to drain.
	exitSignal chan struct{}
	exitOnce   sync.Once

	// gracefulDeadline bounds the drain phase.
	gracefulDeadline time.Duration
}

// NewManager builds a Manager pre-populated with one Program per Spec.
// The runtime is injected so tests can swap in a fake.
func NewManager(specs []Spec, runtime Runtime, log *slog.Logger) (*Manager, error) {
	if runtime == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	if log == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	m := &Manager{
		log:              log,
		runtime:          runtime,
		programs:         make(map[string]*Program, len(specs)),
		events:           make(chan event, 256),
		exitSignal:       make(chan struct{}),
		gracefulDeadline: 10 * time.Second,
	}
	// Pre-Run sentinel — never cancels. Replaced by Run with a real
	// cancellable child of the parent ctx. Lets API methods be called
	// before Run starts (during construction in tests) without nil deref.
	m.shutdownCtx, m.shutdownCancel = context.WithCancel(context.Background())
	for _, s := range specs {
		if _, dup := m.programs[s.Name]; dup {
			return nil, fmt.Errorf("duplicate program name: %q", s.Name)
		}
		m.programs[s.Name] = newProgram(s)
	}
	return m, nil
}

// Run drives the supervisor: autostart, serve events, graceful drain on exit.
// Returns the cause of exit (parent ctx error or nil for internal shutdown).
func (m *Manager) Run(parent context.Context) error {
	// shutdownCtx was initialized in NewManager and is shared between Run
	// and API callers. We do NOT replace it here — that would race with
	// API methods reading the pointer. We just arrange for it to cancel
	// when Run returns so timer/wait-goroutines unblock.
	defer m.shutdownCancel()

	m.autostart()

	exitErr := m.serve(parent)
	m.gracefulDrain()
	return exitErr
}

// serve runs the event loop until parent ctx cancels or exitSignal closes.
func (m *Manager) serve(parent context.Context) error {
	for {
		select {
		case <-parent.Done():
			m.log.Info("event loop stopping", "reason", parent.Err())
			return parent.Err()
		case <-m.exitSignal:
			m.log.Info("event loop stopping", "reason", "internal shutdown")
			return nil
		case e := <-m.events:
			e.handle(m)
		}
	}
}

// gracefulDrain signals every live process and waits for them to exit, up
// to gracefulDeadline. Whoever's still alive at the deadline gets SIGKILL.
func (m *Manager) gracefulDrain() {
	pending := 0
	for _, prg := range m.programs {
		for _, p := range prg.procs {
			p.restartPending = false
			if isLive(p.state) {
				stopProcess(m, &prg.Spec, p)
				pending++
			}
		}
	}
	if pending == 0 {
		return
	}
	m.log.Info("draining", "pending", pending, "deadline", m.gracefulDeadline)

	end := time.Now().Add(m.gracefulDeadline)
	for pending > 0 {
		remaining := time.Until(end)
		if remaining <= 0 {
			m.log.Warn("drain deadline; SIGKILL stragglers", "pending", pending)
			m.killAllLive()
			return
		}
		select {
		case e := <-m.events:
			if _, ok := e.(ProcExited); ok {
				pending--
			}
			e.handle(m)
		case <-time.After(remaining):
			// loop will hit deadline check
		}
	}
	m.log.Info("graceful drain complete")
}

func (m *Manager) killAllLive() {
	for _, prg := range m.programs {
		for _, p := range prg.procs {
			if isLive(p.state) && p.cmd != nil {
				_ = m.runtime.Signal(p.cmd, syscall.SIGKILL)
			}
		}
	}
}

// signalExit closes exitSignal exactly once.
func (m *Manager) signalExit() {
	m.exitOnce.Do(func() { close(m.exitSignal) })
}

// autostart spawns every process for every program whose Spec.Autostart is true.
func (m *Manager) autostart() {
	for _, prg := range m.programs {
		if !prg.Spec.Autostart {
			continue
		}
		for _, proc := range prg.procs {
			if err := m.spawn(&prg.Spec, proc); err != nil {
				m.log.Error("autostart failed",
					"program", prg.Spec.Name, "index", proc.index, "err", err)
			}
		}
	}
}

// spawn asks the runtime to start one process and wires up the start timer.
// Caller must run on the event loop goroutine.
func (m *Manager) spawn(spec *Spec, proc *Process) error {
	req := SpawnRequest{
		Name:       spec.Name,
		Index:      proc.index,
		Bin:        spec.Bin,
		Args:       spec.Args,
		Umask:      spec.Umask,
		Workingdir: spec.Workingdir,
		StdoutPath: spec.StdoutPath,
		StderrPath: spec.StderrPath,
		Env:        spec.Env,
	}
	res, err := m.runtime.SpawnProcess(m.shutdownCtx, req, m.events)
	if err != nil {
		proc.state = Fatal
		return err
	}

	proc.cmd = res.Cmd
	proc.pid = res.Cmd.Process.Pid
	proc.startedAt = res.StartedAt
	proc.state = Starting

	name, idx := spec.Name, proc.index
	delay := spec.Starttime
	if delay <= 0 {
		delay = time.Second
	}
	ctx := m.shutdownCtx
	proc.startTimer = time.AfterFunc(delay, func() {
		select {
		case m.events <- StartTimerFired{Name: name, Index: idx}:
		case <-ctx.Done():
		}
	})

	return nil
}

// lookup returns the Process and its parent Program by (name, index).
// Returns (nil, nil) if the program or index is unknown.
func (m *Manager) lookup(name string, index int) (*Process, *Program) {
	prg, ok := m.programs[name]
	if !ok {
		return nil, nil
	}
	p := prg.process(index)
	if p == nil {
		return nil, prg
	}
	return p, prg
}
