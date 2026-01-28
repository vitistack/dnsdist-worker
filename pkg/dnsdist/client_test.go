package dnsdist

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("127.0.0.1:5199", "test-key")
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.address != "127.0.0.1:5199" {
		t.Errorf("Expected address 127.0.0.1:5199, got %s", client.address)
	}
	
	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey test-key, got %s", client.apiKey)
	}
	
	if client.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.timeout)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient("127.0.0.1:5199", "test-key")
	
	newTimeout := 30 * time.Second
	client.SetTimeout(newTimeout)
	
	if client.timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, client.timeout)
	}
}

func TestIsConnected(t *testing.T) {
	client := NewClient("127.0.0.1:5199", "test-key")
	
	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}
}

func TestClose(t *testing.T) {
	client := NewClient("127.0.0.1:5199", "test-key")
	
	// Close should not error even if not connected
	if err := client.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}
