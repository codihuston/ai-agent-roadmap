package mcp

import (
	"context"

	"agentic-poc/internal/provider"
)

// MCPToolWrapper adapts an MCP tool to the Tool interface.
// This allows MCP tools to be used seamlessly by agents alongside built-in tools.
type MCPToolWrapper struct {
	client MCPClient
	info   MCPToolInfo
}

// NewMCPToolWrapper creates a new MCPToolWrapper for the given tool info.
func NewMCPToolWrapper(client MCPClient, info MCPToolInfo) *MCPToolWrapper {
	return &MCPToolWrapper{
		client: client,
		info:   info,
	}
}

// Name returns the tool's name from the MCP server.
func (w *MCPToolWrapper) Name() string {
	return w.info.Name
}

// Description returns the tool's description from the MCP server.
func (w *MCPToolWrapper) Description() string {
	return w.info.Description
}

// Parameters returns the tool's input schema from the MCP server.
func (w *MCPToolWrapper) Parameters() map[string]interface{} {
	if w.info.InputSchema == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	return w.info.InputSchema
}

// Execute forwards the tool call to the MCP server and returns the result.
func (w *MCPToolWrapper) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	return w.client.CallTool(ctx, w.info.Name, args)
}

// Info returns the underlying MCPToolInfo.
func (w *MCPToolWrapper) Info() MCPToolInfo {
	return w.info
}
