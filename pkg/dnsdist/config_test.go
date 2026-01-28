package dnsdist

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	configData := `{
		"servers": [
			{
				"name": "test-server",
				"address": "127.0.0.1:5199",
				"api_key": "test-key",
				"timeout": 5000000000
			}
		]
	}`
	
	if _, err := tmpFile.Write([]byte(configData)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.Servers))
	}
	
	server := config.Servers[0]
	if server.Name != "test-server" {
		t.Errorf("Expected name test-server, got %s", server.Name)
	}
	
	if server.Address != "127.0.0.1:5199" {
		t.Errorf("Expected address 127.0.0.1:5199, got %s", server.Address)
	}
	
	if server.APIKey != "test-key" {
		t.Errorf("Expected api_key test-key, got %s", server.APIKey)
	}
	
	if server.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", server.Timeout)
	}
}

func TestLoadConfigDefaultTimeout(t *testing.T) {
	// Create a temporary config file without timeout
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	configData := `{
		"servers": [
			{
				"name": "test-server",
				"address": "127.0.0.1:5199",
				"api_key": "test-key"
			}
		]
	}`
	
	if _, err := tmpFile.Write([]byte(configData)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()
	
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	server := config.Servers[0]
	if server.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", server.Timeout)
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	_, err := LoadConfig("nonexistent-file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.Write([]byte("invalid json")); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()
	
	_, err = LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	
	config := &Config{
		Servers: []ServerConfig{
			{
				Name:    "test-server",
				Address: "127.0.0.1:5199",
				APIKey:  "test-key",
				Timeout: 10 * time.Second,
			},
		},
	}
	
	err = SaveConfig(tmpFile.Name(), config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	
	// Load it back to verify
	loadedConfig, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	if len(loadedConfig.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(loadedConfig.Servers))
	}
	
	server := loadedConfig.Servers[0]
	if server.Name != "test-server" {
		t.Errorf("Expected name test-server, got %s", server.Name)
	}
}
