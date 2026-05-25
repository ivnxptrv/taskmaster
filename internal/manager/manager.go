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

func NewManager(cfg *config.Config) (*Manager, error) {
	mgr := &Manager{
		cfg:      cfg,
		programs: make(map[string]*Program),

		events:   make(chan Event),
		commands: make(chan Command),
	}

	err := hydratePrograms(mgr.programs, cfg)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func hydratePrograms(programs map[string]*Program, cfg *config.Config) error {

	for name, cfgP := range cfg.Programs {
		p, err := newProgram(&cfgP)
		if err != nil {
			return err
		}
		programs[name] = p
	}
	return nil
}
