package manager

import (
	"fmt"
	"taskmaster/internal/config"
)

type Event bool
type Command bool

type Manager struct {
	cfg      *config.Config
	programs map[string]*Program

	events   chan Event
	commands chan Command
}

func (m Manager) Validate() error {
	for _, p := range m.programs {
		err := p.validate()
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

func NewManager(cfg *config.Config) *Manager {
	mgr := &Manager{
		cfg:      cfg,
		programs: make(map[string]*Program),

		events:   make(chan Event),
		commands: make(chan Command),
	}

	hydratePrograms(mgr.programs, cfg)

	return mgr
}

func hydratePrograms(programs map[string]*Program, cfg *config.Config) {

	for name, cfgP := range cfg.Programs {
		p := newProgram(name, &cfgP)
		programs[name] = p
	}
}
