package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// Options allows callers to configure the loader's behavior
type Options struct {
}

type Loader struct {
	opts Options
}

func NewLoader(opts Options) *Loader {
	return &Loader{opts: opts}
}

func (l *Loader) Load(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	config := &Config{Filepath: filepath}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// func (l *Loader) Parse(data []byte) (*Config, error) { ... }
