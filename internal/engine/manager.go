package engine

import (
	"fmt"
	"os"
	"strings"
	"taskmaster/internal/client"
	"taskmaster/internal/config"
)

type Manager struct {
	cfg     *config.Config
	spawner *Spawner
	// gateway  *Gateway
	// logger   *Logger ????

	programs map[string]*Program
	environ  map[string]string

	events   chan event
	commands chan command
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

func getCurrentEnv() map[string]string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envMap[pair[0]] = pair[1]
	}
	return envMap
}

func NewManager(s *Spawner, cfg *config.Config) *Manager {
	mgr := &Manager{
		cfg:      cfg,
		spawner:  s,
		programs: make(map[string]*Program),
		environ:  getCurrentEnv(),
		events:   make(chan event, 100),
		commands: make(chan command),
	}

	mgr.hydratePrograms(cfg)

	return mgr
}

func (m *Manager) hydratePrograms(cfg *config.Config) {

	for name, cfgP := range cfg.Programs {
		p := newProgram(name, m, &cfgP)
		m.programs[name] = p
	}
}

func (m *Manager) iter(f func(*Program) error) error {
	for _, v := range m.programs {
		err := f(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// eventloop
func (m *Manager) listen() {
	var err error
	for {
		select {
		case e := <-m.events: // listen for events happened to processes
			err = e.handle()
		case c := <-m.commands: // listen to commands sent to manager by client
			err = c.handle()
		}
	}
}

func autostart(p *Program) error {
	if p.autostart == true {
		err := p.iter(p.manager.spawner.spawn)
		if err != nil {
			return err
		}
	}
	return nil
}

// loop through proccesses and start all which with autostart true
func (m *Manager) Boot() error {

	err := m.iter(autostart)
	if err != nil {
		return err
	}

	m.listen()

	return nil
}
