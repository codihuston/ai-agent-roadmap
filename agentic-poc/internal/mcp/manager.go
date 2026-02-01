package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"

	"agentic-poc/internal/tool"
)

// MCPManager handles multiple MCP server connections and provides
// a unified interface to access all MCP tools.
type MCPManager struct {
	clients map[string]MCPClient
	tools   map[string]*MCPToolWrapper
	mu      sync.RWMutex
}

// NewMCPManager creates a new MCPManager.
func NewMCPManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]MCPClient),
		tools:   make(map[string]*MCPToolWrapper),
	}
}

// LoadFromConfig loads MCP servers from the given configuration.
// Servers that fail to connect are logged but do not prevent other servers
// from being loaded (Property 23: MCP Connection Failures Are Isolated).
func (m *MCPManager) LoadFromConfig(ctx context.Context, cfg *MCPConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil || len(cfg.Servers) == 0 {
		return nil
	}

	var loadedCount int
	for name, serverCfg := range cfg.Servers {
		if serverCfg.Disabled {
			log.Printf("MCP server %q is disabled, skipping", name)
			continue
		}

		if err := m.loadServer(ctx, name, serverCfg); err != nil {
			// Log error but continue with other servers (Property 23)
			log.Printf("Failed to load MCP server %q: %v", name, err)
			continue
		}
		loadedCount++
	}

	log.Printf("Loaded %d MCP servers", loadedCount)
	return nil
}

// loadServer connects to a single MCP server and registers its tools.
func (m *MCPManager) loadServer(ctx context.Context, name string, cfg MCPServerConfig) error {
	client := NewStdioMCPClient(cfg.Command, cfg.Args, cfg.Env)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to list tools: %w", err)
	}

	m.clients[name] = client

	for _, toolInfo := range tools {
		wrapper := NewMCPToolWrapperWithServer(client, toolInfo, name)
		// Use server name prefix to avoid tool name collisions
		toolKey := fmt.Sprintf("%s/%s", name, toolInfo.Name)
		m.tools[toolKey] = wrapper
		log.Printf("Registered MCP tool: %s", toolKey)
	}

	return nil
}

// AddClient adds a pre-configured MCP client to the manager.
// This is useful for testing or when using custom client implementations.
func (m *MCPManager) AddClient(ctx context.Context, name string, client MCPClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client %q: %w", name, err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to list tools for %q: %w", name, err)
	}

	m.clients[name] = client

	for _, toolInfo := range tools {
		wrapper := NewMCPToolWrapper(client, toolInfo)
		toolKey := fmt.Sprintf("%s/%s", name, toolInfo.Name)
		m.tools[toolKey] = wrapper
	}

	return nil
}

// GetTools returns all MCP tools as Tool interface implementations.
func (m *MCPManager) GetTools() []tool.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]tool.Tool, 0, len(m.tools))
	for _, wrapper := range m.tools {
		tools = append(tools, wrapper)
	}
	return tools
}

// GetTool returns a specific MCP tool by its full name (server/tool).
func (m *MCPManager) GetTool(name string) (tool.Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wrapper, ok := m.tools[name]
	if !ok {
		return nil, false
	}
	return wrapper, true
}

// GetClient returns a specific MCP client by server name.
func (m *MCPManager) GetClient(name string) (MCPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[name]
	return client, ok
}

// ServerNames returns the names of all connected MCP servers.
func (m *MCPManager) ServerNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// ToolCount returns the total number of registered MCP tools.
func (m *MCPManager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}

// Shutdown closes all MCP server connections.
func (m *MCPManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			log.Printf("Error closing MCP server %q: %v", name, err)
			lastErr = err
		}
	}

	// Clear maps
	m.clients = make(map[string]MCPClient)
	m.tools = make(map[string]*MCPToolWrapper)

	return lastErr
}
