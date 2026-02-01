package mcp

import (
	"context"
	"fmt"

	"agentic-poc/internal/provider"
)

// MockMCPClient is a mock implementation of MCPClient for testing.
type MockMCPClient struct {
	ConnectFunc   func(ctx context.Context) error
	ListToolsFunc func(ctx context.Context) ([]MCPToolInfo, error)
	CallToolFunc  func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error)
	CloseFunc     func() error

	connected bool
}

// NewMockMCPClient creates a new MockMCPClient with default implementations.
func NewMockMCPClient() *MockMCPClient {
	return &MockMCPClient{}
}

// Connect implements MCPClient.
func (m *MockMCPClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	m.connected = true
	return nil
}

// ListTools implements MCPClient.
func (m *MockMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	if m.ListToolsFunc != nil {
		return m.ListToolsFunc(ctx)
	}
	return []MCPToolInfo{
		{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Input value",
					},
				},
				"required": []interface{}{"input"},
			},
		},
	}, nil
}

// CallTool implements MCPClient.
func (m *MockMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
	if m.CallToolFunc != nil {
		return m.CallToolFunc(ctx, name, args)
	}
	return &provider.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Called %s with args: %v", name, args),
	}, nil
}

// Close implements MCPClient.
func (m *MockMCPClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	m.connected = false
	return nil
}

// IsConnected returns whether the mock client is connected.
func (m *MockMCPClient) IsConnected() bool {
	return m.connected
}
