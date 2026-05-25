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

func Load(filepath string) (*Config, error) {
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

// config.Print(cfg)
func (c Config) Print() {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	encoder.SetIndent(2)
	// The encoder will now automatically call MarshalYAML for any Umask fields
	if err := encoder.Encode(c); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode config: %v\n", err)
	}
}

// func validConfig(c *Config) error {
// 	if len(c.Programs) == 0 {
// 		return fmt.Errorf("config must contain at least one program")
// 	}
// 	for name, p := range c.Programs {
// 		if p.Cmd == "" {
// 			return fmt.Errorf("program %s: cmd is required", name)
// 		}
// 		if p.Numprocs == ""
// 	}
// 	return nil
// }
