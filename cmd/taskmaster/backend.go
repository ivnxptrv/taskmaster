package main

import (
	"fmt"
	"os"

	"taskmaster/internal/client"
	"taskmaster/internal/config"
	"taskmaster/internal/engine"
)

// engineBackend adapts *engine.Manager to client.Backend. It additionally
// owns the config path so it can drive a reload (re-parsing the file).
type engineBackend struct {
	m       *engine.Manager
	cfgPath string
}

func (b *engineBackend) Start(name string, index int) error   { return b.m.Start(name, index) }
func (b *engineBackend) Stop(name string, index int) error    { return b.m.Stop(name, index) }
func (b *engineBackend) Restart(name string, index int) error { return b.m.Restart(name, index) }
func (b *engineBackend) Shutdown()                            { b.m.Shutdown() }

func (b *engineBackend) Reload() error {
	specs, err := loadSpecs(b.cfgPath)
	if err != nil {
		return err
	}
	return b.m.Reload(specs)
}

func (b *engineBackend) Status() []client.ProcInfo {
	src := b.m.Status()
	out := make([]client.ProcInfo, len(src))
	for i, p := range src {
		out[i] = client.ProcInfo{
			Name:      p.Name,
			Index:     p.Index,
			State:     p.State,
			Pid:       p.Pid,
			StartedAt: p.StartedAt,
			Retries:   p.Retries,
		}
	}
	return out
}

// loadSpecs reads the config file at path and translates it into engine specs.
// Shared by initial load (in run) and reload paths (SIGHUP + shell `reload`).
func loadSpecs(path string) ([]engine.Spec, error) {
	loader := config.NewLoader(config.Options{})
	cfg, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	specs, err := engine.SpecsFromConfig(cfg, os.Environ())
	if err != nil {
		return nil, fmt.Errorf("translate config: %w", err)
	}
	return specs, nil
}
