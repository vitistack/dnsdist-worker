package dnsdist

import (
	"testing"
	"time"
)

func TestNewWorker(t *testing.T) {
	worker := NewWorker()
	
	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}
	
	if worker.servers == nil {
		t.Error("Worker servers map is nil")
	}
	
	if worker.configs == nil {
		t.Error("Worker configs map is nil")
	}
}

func TestAddServer(t *testing.T) {
	worker := NewWorker()
	
	config := &ServerConfig{
		Name:    "test-server",
		Address: "127.0.0.1:5199",
		APIKey:  "test-key",
	}
	
	err := worker.AddServer(config)
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}
	
	names := worker.GetServerNames()
	if len(names) != 1 {
		t.Errorf("Expected 1 server, got %d", len(names))
	}
	
	if names[0] != "test-server" {
		t.Errorf("Expected server name test-server, got %s", names[0])
	}
	
	// Test that timeout is set to default if not specified
	if config.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", config.Timeout)
	}
}

func TestAddServerValidation(t *testing.T) {
	worker := NewWorker()
	
	// Test empty name
	err := worker.AddServer(&ServerConfig{
		Address: "127.0.0.1:5199",
		APIKey:  "test-key",
	})
	if err == nil {
		t.Error("Expected error for empty server name")
	}
	
	// Test empty address
	err = worker.AddServer(&ServerConfig{
		Name:   "test-server",
		APIKey: "test-key",
	})
	if err == nil {
		t.Error("Expected error for empty server address")
	}
}

func TestRemoveServer(t *testing.T) {
	worker := NewWorker()
	
	config := &ServerConfig{
		Name:    "test-server",
		Address: "127.0.0.1:5199",
		APIKey:  "test-key",
	}
	
	worker.AddServer(config)
	
	err := worker.RemoveServer("test-server")
	if err != nil {
		t.Fatalf("RemoveServer failed: %v", err)
	}
	
	names := worker.GetServerNames()
	if len(names) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(names))
	}
}

func TestGetServerNames(t *testing.T) {
	worker := NewWorker()
	
	worker.AddServer(&ServerConfig{
		Name:    "server1",
		Address: "127.0.0.1:5199",
		APIKey:  "key1",
	})
	
	worker.AddServer(&ServerConfig{
		Name:    "server2",
		Address: "127.0.0.1:5200",
		APIKey:  "key2",
	})
	
	names := worker.GetServerNames()
	if len(names) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(names))
	}
}

func TestGetConnectedServers(t *testing.T) {
	worker := NewWorker()
	
	connected := worker.GetConnectedServers()
	if len(connected) != 0 {
		t.Errorf("Expected 0 connected servers, got %d", len(connected))
	}
}

func TestWorkerIsConnected(t *testing.T) {
	worker := NewWorker()
	
	worker.AddServer(&ServerConfig{
		Name:    "test-server",
		Address: "127.0.0.1:5199",
		APIKey:  "test-key",
	})
	
	if worker.IsConnected("test-server") {
		t.Error("Server should not be connected")
	}
	
	if worker.IsConnected("nonexistent") {
		t.Error("Nonexistent server should not be connected")
	}
}

func TestWorkerClose(t *testing.T) {
	worker := NewWorker()
	
	worker.AddServer(&ServerConfig{
		Name:    "test-server",
		Address: "127.0.0.1:5199",
		APIKey:  "test-key",
	})
	
	// Close should not panic
	worker.Close()
	
	connected := worker.GetConnectedServers()
	if len(connected) != 0 {
		t.Errorf("Expected 0 connected servers after close, got %d", len(connected))
	}
}
