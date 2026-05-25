package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Filepath string             // path to config file
	Programs map[string]Program `yaml:"programs"`
}

func (c Config) Print() {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	encoder.SetIndent(2)
	// The encoder will now automatically call MarshalYAML for any Umask fields
	if err := encoder.Encode(c); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode config: %v\n", err)
	}
}

func (c Config) Validate() error {
	if len(c.Programs) == 0 {
		return fmt.Errorf("config must contain at least one program")
	}
	for name, p := range c.Programs {
		err := p.validate()
		if err != nil {
			return fmt.Errorf("program '%s': %w", name, err)
		}
	}
	return nil
}
