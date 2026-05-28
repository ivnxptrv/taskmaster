package engine

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"taskmaster/internal/config"
)

type Manager struct {
	cfg     *config.Config
	spawner *Spawner
	log     *slog.Logger

	programs map[string]*Program
	environ  map[string]string

	Events chan event
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

func NewManager(s *Spawner, cfg *config.Config, log *slog.Logger) *Manager {
	mgr := &Manager{
		cfg:      cfg,
		log:      log,
		spawner:  s,
		programs: make(map[string]*Program),
		environ:  getCurrentEnv(),
		Events:   make(chan event, 100),
	}

	for name, cfgP := range cfg.Programs {
		log := log.With("program", name)
		p := newProgram(name, mgr, &cfgP, log)
		mgr.programs[name] = p
	}

	return mgr
}

func (m *Manager) hydratePrograms(cfg *config.Config) {

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
	for e := range m.Events {
		e.handle()
		// logger log err happened
	}
}

func autostart(p *Program) error {
	if p.autostart == true {
		p.iter(setStarting)
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

	go m.listen()

	return nil
}
