package mcp

import (
	"context"
	"errors"
	"testing"

	"agentic-poc/internal/provider"
)

func TestNewMCPManager(t *testing.T) {
	manager := NewMCPManager()
	if manager == nil {
		t.Fatal("NewMCPManager() returned nil")
	}
	if manager.ToolCount() != 0 {
		t.Errorf("new manager should have 0 tools, got %d", manager.ToolCount())
	}
	if len(manager.ServerNames()) != 0 {
		t.Errorf("new manager should have 0 servers, got %d", len(manager.ServerNames()))
	}
}

func TestMCPManager_AddClient(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{Name: "tool1", Description: "Tool 1"},
			{Name: "tool2", Description: "Tool 2"},
		}, nil
	}

	err := manager.AddClient(context.Background(), "test-server", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	if manager.ToolCount() != 2 {
		t.Errorf("expected 2 tools, got %d", manager.ToolCount())
	}

	names := manager.ServerNames()
	if len(names) != 1 || names[0] != "test-server" {
		t.Errorf("expected server 'test-server', got %v", names)
	}
}

func TestMCPManager_AddClient_ConnectError(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ConnectFunc = func(ctx context.Context) error {
		return errors.New("connection failed")
	}

	err := manager.AddClient(context.Background(), "test-server", client)
	if err == nil {
		t.Error("expected error when connection fails")
	}
}

func TestMCPManager_AddClient_ListToolsError(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return nil, errors.New("list tools failed")
	}

	err := manager.AddClient(context.Background(), "test-server", client)
	if err == nil {
		t.Error("expected error when list tools fails")
	}
}

func TestMCPManager_GetTools(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{Name: "read_file", Description: "Read a file"},
			{Name: "write_file", Description: "Write a file"},
		}, nil
	}

	err := manager.AddClient(context.Background(), "filesystem", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	tools := manager.GetTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Verify tools implement the Tool interface
	for _, tool := range tools {
		if tool.Name() == "" {
			t.Error("tool Name() should not be empty")
		}
		if tool.Description() == "" {
			t.Error("tool Description() should not be empty")
		}
		if tool.Parameters() == nil {
			t.Error("tool Parameters() should not be nil")
		}
	}
}

func TestMCPManager_GetTool(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{Name: "read_file", Description: "Read a file"},
		}, nil
	}

	err := manager.AddClient(context.Background(), "filesystem", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	// Tool should be accessible with server prefix
	tool, ok := manager.GetTool("filesystem/read_file")
	if !ok {
		t.Error("expected to find tool 'filesystem/read_file'")
	}
	if tool.Name() != "read_file" {
		t.Errorf("expected tool name 'read_file', got %q", tool.Name())
	}

	// Non-existent tool
	_, ok = manager.GetTool("filesystem/nonexistent")
	if ok {
		t.Error("should not find nonexistent tool")
	}
}

func TestMCPManager_GetClient(t *testing.T) {
	manager := NewMCPManager()
	client := NewMockMCPClient()

	err := manager.AddClient(context.Background(), "test-server", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	got, ok := manager.GetClient("test-server")
	if !ok {
		t.Error("expected to find client 'test-server'")
	}
	if got != client {
		t.Error("returned client should be the same as added client")
	}

	_, ok = manager.GetClient("nonexistent")
	if ok {
		t.Error("should not find nonexistent client")
	}
}

func TestMCPManager_Shutdown(t *testing.T) {
	manager := NewMCPManager()

	closeCalled := false
	client := NewMockMCPClient()
	client.CloseFunc = func() error {
		closeCalled = true
		return nil
	}

	err := manager.AddClient(context.Background(), "test-server", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	err = manager.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if !closeCalled {
		t.Error("Close() should have been called on client")
	}

	if manager.ToolCount() != 0 {
		t.Error("tools should be cleared after shutdown")
	}
	if len(manager.ServerNames()) != 0 {
		t.Error("servers should be cleared after shutdown")
	}
}

func TestMCPManager_Shutdown_WithError(t *testing.T) {
	manager := NewMCPManager()

	client := NewMockMCPClient()
	client.CloseFunc = func() error {
		return errors.New("close failed")
	}

	err := manager.AddClient(context.Background(), "test-server", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	err = manager.Shutdown()
	if err == nil {
		t.Error("expected error from Shutdown()")
	}
}

func TestMCPManager_MultipleServers(t *testing.T) {
	manager := NewMCPManager()

	// Add first server
	client1 := NewMockMCPClient()
	client1.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{Name: "tool_a", Description: "Tool A"},
		}, nil
	}

	// Add second server
	client2 := NewMockMCPClient()
	client2.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{Name: "tool_b", Description: "Tool B"},
			{Name: "tool_c", Description: "Tool C"},
		}, nil
	}

	err := manager.AddClient(context.Background(), "server1", client1)
	if err != nil {
		t.Fatalf("AddClient(server1) error = %v", err)
	}

	err = manager.AddClient(context.Background(), "server2", client2)
	if err != nil {
		t.Fatalf("AddClient(server2) error = %v", err)
	}

	if manager.ToolCount() != 3 {
		t.Errorf("expected 3 tools, got %d", manager.ToolCount())
	}

	names := manager.ServerNames()
	if len(names) != 2 {
		t.Errorf("expected 2 servers, got %d", len(names))
	}

	// Verify tools from different servers are accessible
	_, ok := manager.GetTool("server1/tool_a")
	if !ok {
		t.Error("expected to find server1/tool_a")
	}

	_, ok = manager.GetTool("server2/tool_b")
	if !ok {
		t.Error("expected to find server2/tool_b")
	}
}

func TestMCPManager_LoadFromConfig_NilConfig(t *testing.T) {
	manager := NewMCPManager()
	err := manager.LoadFromConfig(context.Background(), nil)
	if err != nil {
		t.Errorf("LoadFromConfig(nil) should not error: %v", err)
	}
}

func TestMCPManager_LoadFromConfig_EmptyConfig(t *testing.T) {
	manager := NewMCPManager()
	cfg := &MCPConfig{Servers: map[string]MCPServerConfig{}}
	err := manager.LoadFromConfig(context.Background(), cfg)
	if err != nil {
		t.Errorf("LoadFromConfig(empty) should not error: %v", err)
	}
}

func TestMCPManager_ToolsImplementInterface(t *testing.T) {
	// Property 21: MCP Tools Implement Tool Interface
	manager := NewMCPManager()
	client := NewMockMCPClient()
	client.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{
			{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{"type": "string"},
					},
				},
			},
		}, nil
	}
	client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
		return &provider.ToolResult{Success: true, Output: "result"}, nil
	}

	err := manager.AddClient(context.Background(), "test", client)
	if err != nil {
		t.Fatalf("AddClient() error = %v", err)
	}

	tools := manager.GetTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]

	// Verify Tool interface methods
	if tool.Name() == "" {
		t.Error("Name() should return non-empty string")
	}
	if tool.Description() == "" {
		t.Error("Description() should return non-empty string")
	}
	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() should return non-nil map")
	}
	if _, ok := params["type"]; !ok {
		t.Error("Parameters() should contain 'type' field")
	}

	// Verify Execute works
	result, err := tool.Execute(context.Background(), map[string]interface{}{"input": "test"})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if result == nil {
		t.Error("Execute() should return non-nil result")
	}
	if !result.Success {
		t.Error("Execute() result should be successful")
	}
}

func TestMCPManager_ConnectionFailuresIsolated(t *testing.T) {
	// Property 23: MCP Connection Failures Are Isolated
	// This test verifies that one server failing doesn't affect others
	// Note: LoadFromConfig uses StdioMCPClient internally, so we test
	// the isolation behavior through AddClient

	manager := NewMCPManager()

	// First client succeeds
	client1 := NewMockMCPClient()
	client1.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{{Name: "tool1"}}, nil
	}

	// Second client fails to connect
	client2 := NewMockMCPClient()
	client2.ConnectFunc = func(ctx context.Context) error {
		return errors.New("connection refused")
	}

	// Third client succeeds
	client3 := NewMockMCPClient()
	client3.ListToolsFunc = func(ctx context.Context) ([]MCPToolInfo, error) {
		return []MCPToolInfo{{Name: "tool3"}}, nil
	}

	// Add clients - second one should fail but not affect others
	err := manager.AddClient(context.Background(), "server1", client1)
	if err != nil {
		t.Errorf("server1 should succeed: %v", err)
	}

	err = manager.AddClient(context.Background(), "server2", client2)
	if err == nil {
		t.Error("server2 should fail")
	}

	err = manager.AddClient(context.Background(), "server3", client3)
	if err != nil {
		t.Errorf("server3 should succeed: %v", err)
	}

	// Verify server1 and server3 tools are available
	if manager.ToolCount() != 2 {
		t.Errorf("expected 2 tools (from server1 and server3), got %d", manager.ToolCount())
	}

	_, ok := manager.GetTool("server1/tool1")
	if !ok {
		t.Error("server1/tool1 should be available")
	}

	_, ok = manager.GetTool("server3/tool3")
	if !ok {
		t.Error("server3/tool3 should be available")
	}
}
