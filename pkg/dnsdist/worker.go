package dnsdist

import (
	"fmt"
	"sync"
	"time"
)

// ServerConfig represents the configuration for a single dnsdist server
type ServerConfig struct {
	Name    string        `json:"name"`
	Address string        `json:"address"`
	APIKey  string        `json:"api_key"`
	Timeout time.Duration `json:"timeout"`
}

// Worker manages connections to multiple dnsdist servers
type Worker struct {
	servers map[string]*Client
	configs map[string]*ServerConfig
	mu      sync.RWMutex
}

// NewWorker creates a new dnsdist worker
func NewWorker() *Worker {
	return &Worker{
		servers: make(map[string]*Client),
		configs: make(map[string]*ServerConfig),
	}
}

// AddServer adds a server configuration to the worker
func (w *Worker) AddServer(config *ServerConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if config.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	w.configs[config.Name] = config
	return nil
}

// RemoveServer removes a server from the worker
func (w *Worker) RemoveServer(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if client, exists := w.servers[name]; exists {
		client.Close()
		delete(w.servers, name)
	}

	delete(w.configs, name)
	return nil
}

// Connect connects to a specific server
func (w *Worker) Connect(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	config, exists := w.configs[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	// Close existing connection if any
	if client, exists := w.servers[name]; exists {
		client.Close()
	}

	// Create new client
	client := NewClient(config.Address, config.APIKey)
	client.SetTimeout(config.Timeout)

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", name, err)
	}

	w.servers[name] = client
	return nil
}

// ConnectAll connects to all configured servers
func (w *Worker) ConnectAll() map[string]error {
	w.mu.RLock()
	names := make([]string, 0, len(w.configs))
	for name := range w.configs {
		names = append(names, name)
	}
	w.mu.RUnlock()

	errors := make(map[string]error)
	for _, name := range names {
		if err := w.Connect(name); err != nil {
			errors[name] = err
		}
	}

	return errors
}

// ExecuteCommand executes a command on a specific server
func (w *Worker) ExecuteCommand(name, command string) (string, error) {
	w.mu.RLock()
	client, exists := w.servers[name]
	w.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("not connected to server %s", name)
	}

	return client.ExecuteCommand(command)
}

// ExecuteCommandAll executes a command on all connected servers
func (w *Worker) ExecuteCommandAll(command string) map[string]CommandResult {
	w.mu.RLock()
	names := make([]string, 0, len(w.servers))
	for name := range w.servers {
		names = append(names, name)
	}
	w.mu.RUnlock()

	results := make(map[string]CommandResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, name := range names {
		wg.Add(1)
		go func(serverName string) {
			defer wg.Done()

			response, err := w.ExecuteCommand(serverName, command)
			
			mu.Lock()
			results[serverName] = CommandResult{
				Response: response,
				Error:    err,
			}
			mu.Unlock()
		}(name)
	}

	wg.Wait()
	return results
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Response string
	Error    error
}

// Close closes all server connections
func (w *Worker) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for name, client := range w.servers {
		client.Close()
		delete(w.servers, name)
	}
}

// GetServerNames returns a list of all configured server names
func (w *Worker) GetServerNames() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	names := make([]string, 0, len(w.configs))
	for name := range w.configs {
		names = append(names, name)
	}
	return names
}

// GetConnectedServers returns a list of connected server names
func (w *Worker) GetConnectedServers() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	names := make([]string, 0, len(w.servers))
	for name := range w.servers {
		names = append(names, name)
	}
	return names
}

// IsConnected checks if a server is connected
func (w *Worker) IsConnected(name string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	client, exists := w.servers[name]
	if !exists {
		return false
	}

	return client.IsConnected()
}
