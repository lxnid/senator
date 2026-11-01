package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Manager handles the lifecycle of multiple MCP server connections
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewManager creates a new MCP connection manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// Connect establishes a connection to an MCP server and registers it
func (m *Manager) Connect(ctx context.Context, cfg ClientConfig) (*Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if existing, ok := m.clients[cfg.ServerName]; ok {
		return existing, nil
	}

	// Create new client
	client, err := NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Register the client
	m.clients[cfg.ServerName] = client
	return client, nil
}

// Get retrieves a connected client by name
func (m *Manager) Get(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// Disconnect closes a specific MCP server connection
func (m *Manager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("no client found with name: %s", name)
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to close client %s: %w", name, err)
	}

	delete(m.clients, name)
	return nil
}

// DisconnectAll closes all MCP server connections
func (m *Manager) DisconnectAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
		}
		delete(m.clients, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}
	return nil
}

// ListConnected returns the names of all connected MCP servers
func (m *Manager) ListConnected() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// Count returns the number of connected MCP servers
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}
