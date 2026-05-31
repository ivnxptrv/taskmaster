package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"taskmaster/internal/config"
	"time"
)

type Manager struct {
	ctx context.Context
	cfg *config.Config
	log *slog.Logger

	runtime Runtime

	programs map[string]*Program
	environ  map[string]string

	events chan event
}

func (m Manager) Validate() error {
	for _, p := range m.programs {
		err := p.spec.validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m Manager) Print() {
	for name, p := range m.programs {
		fmt.Printf("program[%s]: %+v\n\n", name, p)
	}
}

func getCurrentEnv() map[string]string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envMap[pair[0]] = pair[1]
	}
	return envMap
}

func (m *Manager) getProgram(name string) (*Program, bool) {
	p, ok := m.programs[name]
	if !ok {
		return nil, false
	}
	return p, ok
}

func (m *Manager) getProcess(name string, index int) (*Process, bool) {
	prg, ok := m.getProgram(name)
	if !ok {
		return nil, false
	}

	proc, ok := prg.getProcess(index)
	if !ok {
		return nil, false
	}
	return proc, true
}

func popProgram() {}

func NewManager(cfg *config.Config, runtime Runtime, log *slog.Logger) (*Manager, error) {
	mgr := &Manager{
		cfg:      cfg,
		log:      log,
		runtime:  runtime,
		programs: make(map[string]*Program),
		environ:  getCurrentEnv(),
		events:   make(chan event, 100),
	}

	for name, cfgP := range cfg.Programs {
		p := newProgram(name, &cfgP, mgr.environ)
		mgr.programs[name] = p
	}

	// validate manager
	err := mgr.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate manager: %w", err)
	}

	return mgr, nil
}

// eventloop
func (m *Manager) listen() {
	for {
		select {

		case <-m.ctx.Done():
			return // garefully drain events and shutdown

		case e := <-m.events:
			err := e.handle(m)
			err = fmt.Errorf("failed to handle message: %w", err)
			m.log.Warn(err.Error())
		}
	}
}

// loop through proccesses and start all which are autostart true
func (m *Manager) Run(ctx context.Context) error {

	// attach context
	m.ctx = ctx

	// autostart
	for _, prg := range m.programs {
		if prg.spec.autostart {
			for _, proc := range prg.procs {
				err := m.spawn(prg.spec, proc)
				proc.state = Starting
				if err != nil {
					err = fmt.Errorf("failed to spawn a process: %w", err)
					m.log.Warn(err.Error())
				}
			}
		}
	}

	go m.listen()

	return nil
}

func (m *Manager) submitEvent(e event) {
	m.events <- e
}

// allocate running process object
func (m *Manager) spawn(s Spec, proc *Process) error {

	err := s.validate()
	if err != nil {
		return err
	}

	req := SpawnRequest{
		name:       s.name,
		index:      proc.index,
		bin:        string(s.bin),
		args:       s.args,
		umask:      s.umask,
		workingdir: string(s.workingdir),
		stdoutPath: string(s.stdout),
		stderrPath: string(s.stderr),
		env:        s.env,
	}

	// TODO: own context?
	res, err := m.runtime.SpawnProcess(m.ctx, m.events, req)
	if err != nil {
		return err
	}
	// spawn listener for starttime to consider process Running
	proc.startTimer = time.AfterFunc(s.starttime, func() {
		m.events <- StartTimerFired{Name: s.name, Index: proc.index}
	})
	proc.startedAt = time.Now()
	proc.cmd = res.cmd
	proc.stdoutFile = res.stdoutFile
	proc.stderrFile = res.stderrFile

	return nil
}

// gracefully shutdown
// func (m *Manager) despawn(p *Process) error {}

// func (s *realSpawner) Despawn(p *Process) error {
// 	if p.cmd == nil || p.cmd.Process == nil {
// 		return nil // Process isn't running
// 	}

// 	p.log.Info("sending SIGTERM to process", "pid", p.pid)
// 	err := p.cmd.Process.Signal(syscall.SIGTERM)
// 	if err != nil {
// 		return err
// 	}

// 	// Schedule SIGKILL after 5 seconds
// 	p.stopTimer = time.AfterFunc(5*time.Second, func() {
// 		p.log.Warn("process did not exit, sending SIGKILL", "pid", p.pid)
// 		_ = p.cmd.Process.Kill()
// 	})

// 	return nil
// }
