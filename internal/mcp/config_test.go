package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPConfig(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *MCPConfig)
	}{
		{
			name: "valid config with single server",
			content: `{
				"mcpServers": {
					"filesystem": {
						"command": "npx",
						"args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
						"disabled": false
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				if cfg.ServerCount() != 1 {
					t.Errorf("expected 1 server, got %d", cfg.ServerCount())
				}
				server, ok := cfg.Servers["filesystem"]
				if !ok {
					t.Fatal("expected filesystem server")
				}
				if server.Command != "npx" {
					t.Errorf("expected command 'npx', got %q", server.Command)
				}
				if len(server.Args) != 3 {
					t.Errorf("expected 3 args, got %d", len(server.Args))
				}
			},
		},
		{
			name: "valid config with multiple servers",
			content: `{
				"mcpServers": {
					"filesystem": {
						"command": "npx",
						"args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
					},
					"web-search": {
						"command": "python",
						"args": ["-m", "mcp_server_web_search"]
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				if cfg.ServerCount() != 2 {
					t.Errorf("expected 2 servers, got %d", cfg.ServerCount())
				}
			},
		},
		{
			name: "config with disabled server",
			content: `{
				"mcpServers": {
					"filesystem": {
						"command": "npx",
						"args": [],
						"disabled": true
					},
					"web-search": {
						"command": "python",
						"args": []
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				if cfg.ServerCount() != 2 {
					t.Errorf("expected 2 servers, got %d", cfg.ServerCount())
				}
				if cfg.EnabledServerCount() != 1 {
					t.Errorf("expected 1 enabled server, got %d", cfg.EnabledServerCount())
				}
				enabled := cfg.EnabledServers()
				if _, ok := enabled["filesystem"]; ok {
					t.Error("filesystem should not be in enabled servers")
				}
				if _, ok := enabled["web-search"]; !ok {
					t.Error("web-search should be in enabled servers")
				}
			},
		},
		{
			name: "config with environment variables",
			content: `{
				"mcpServers": {
					"api-server": {
						"command": "node",
						"args": ["server.js"],
						"env": {
							"API_KEY": "secret123",
							"DEBUG": "true"
						}
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				server := cfg.Servers["api-server"]
				if len(server.Env) != 2 {
					t.Errorf("expected 2 env vars, got %d", len(server.Env))
				}
				if server.Env["API_KEY"] != "secret123" {
					t.Errorf("expected API_KEY=secret123, got %q", server.Env["API_KEY"])
				}
			},
		},
		{
			name: "config with autoApprove",
			content: `{
				"mcpServers": {
					"filesystem": {
						"command": "npx",
						"args": [],
						"autoApprove": ["read_file", "list_directory"]
					}
				}
			}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				server := cfg.Servers["filesystem"]
				if len(server.AutoApprove) != 2 {
					t.Errorf("expected 2 autoApprove tools, got %d", len(server.AutoApprove))
				}
			},
		},
		{
			name:        "invalid JSON",
			content:     `{invalid json}`,
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "missing command",
			content: `{
				"mcpServers": {
					"bad-server": {
						"args": ["arg1"]
					}
				}
			}`,
			wantErr:     true,
			errContains: "command is required",
		},
		{
			name:    "empty config",
			content: `{}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				if cfg.ServerCount() != 0 {
					t.Errorf("expected 0 servers, got %d", cfg.ServerCount())
				}
			},
		},
		{
			name:    "empty servers",
			content: `{"mcpServers": {}}`,
			wantErr: false,
			validate: func(t *testing.T, cfg *MCPConfig) {
				if cfg.ServerCount() != 0 {
					t.Errorf("expected 0 servers, got %d", cfg.ServerCount())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "mcp.json")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp config: %v", err)
			}

			cfg, err := LoadMCPConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadMCPConfig_FileNotFound(t *testing.T) {
	_, err := LoadMCPConfig("/nonexistent/path/mcp.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestMCPConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPConfig
		wantErr bool
	}{
		{
			name:    "nil servers",
			config:  MCPConfig{Servers: nil},
			wantErr: false,
		},
		{
			name:    "empty servers",
			config:  MCPConfig{Servers: map[string]MCPServerConfig{}},
			wantErr: false,
		},
		{
			name: "valid server",
			config: MCPConfig{
				Servers: map[string]MCPServerConfig{
					"test": {Command: "echo"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing command",
			config: MCPConfig{
				Servers: map[string]MCPServerConfig{
					"test": {Args: []string{"arg1"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
