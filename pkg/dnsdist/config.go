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

	// Validate and set defaults
	seenNames := make(map[string]bool)
	for i := range config.Servers {
		// Validate required fields
		if config.Servers[i].Name == "" {
			return nil, fmt.Errorf("server at index %d has empty name", i)
		}
		if config.Servers[i].Address == "" {
			return nil, fmt.Errorf("server '%s' has empty address", config.Servers[i].Name)
		}

		// Check for duplicate names
		if seenNames[config.Servers[i].Name] {
			return nil, fmt.Errorf("duplicate server name: %s", config.Servers[i].Name)
		}
		seenNames[config.Servers[i].Name] = true

		// Set default timeout if not specified
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

	// Use restrictive permissions (0600) to protect sensitive API keys
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
