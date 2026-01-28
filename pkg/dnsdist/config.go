package dnsdist

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the overall configuration for the worker
type Config struct {
	Servers []ServerConfig `json:"servers"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default timeout if not specified
	for i := range config.Servers {
		if config.Servers[i].Timeout == 0 {
			config.Servers[i].Timeout = 10 * time.Second
		}
	}

	return &config, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
