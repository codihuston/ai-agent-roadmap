package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

// MCPConfig defines MCP server connections.
type MCPConfig struct {
	Servers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig defines the configuration for a single MCP server.
type MCPServerConfig struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Disabled    bool              `json:"disabled"`
	AutoApprove []string          `json:"autoApprove"`
}

// LoadMCPConfig loads MCP configuration from a JSON file.
func LoadMCPConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("MCP config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read MCP config file: %w", err)
	}

	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCP config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the configuration is valid.
func (c *MCPConfig) Validate() error {
	if c.Servers == nil {
		return nil
	}

	for name, server := range c.Servers {
		if server.Command == "" {
			return fmt.Errorf("server %q: command is required", name)
		}
	}

	return nil
}

// EnabledServers returns only the servers that are not disabled.
func (c *MCPConfig) EnabledServers() map[string]MCPServerConfig {
	if c.Servers == nil {
		return nil
	}

	enabled := make(map[string]MCPServerConfig)
	for name, server := range c.Servers {
		if !server.Disabled {
			enabled[name] = server
		}
	}
	return enabled
}

// ServerCount returns the total number of configured servers.
func (c *MCPConfig) ServerCount() int {
	if c.Servers == nil {
		return 0
	}
	return len(c.Servers)
}

// EnabledServerCount returns the number of enabled servers.
func (c *MCPConfig) EnabledServerCount() int {
	if c.Servers == nil {
		return 0
	}

	count := 0
	for _, server := range c.Servers {
		if !server.Disabled {
			count++
		}
	}
	return count
}
