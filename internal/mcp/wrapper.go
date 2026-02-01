package mcp

import (
	"context"
	"log"

	"agentic-poc/internal/provider"
)

// MCPToolWrapper adapts an MCP tool to the Tool interface.
// This allows MCP tools to be used seamlessly by agents alongside built-in tools.
type MCPToolWrapper struct {
	client     MCPClient
	info       MCPToolInfo
	serverName string
}

// NewMCPToolWrapper creates a new MCPToolWrapper for the given tool info.
func NewMCPToolWrapper(client MCPClient, info MCPToolInfo) *MCPToolWrapper {
	return &MCPToolWrapper{
		client: client,
		info:   info,
	}
}

// NewMCPToolWrapperWithServer creates a new MCPToolWrapper with server name for logging.
func NewMCPToolWrapperWithServer(client MCPClient, info MCPToolInfo, serverName string) *MCPToolWrapper {
	return &MCPToolWrapper{
		client:     client,
		info:       info,
		serverName: serverName,
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
	log.Printf("[MCP Client] Calling tool %q on server %q with args: %v", w.info.Name, w.serverName, args)

	result, err := w.client.CallTool(ctx, w.info.Name, args)
	if err != nil {
		log.Printf("[MCP Client] Tool %q error: %v", w.info.Name, err)
		return nil, err
	}

	if result.Success {
		log.Printf("[MCP Client] Tool %q succeeded: %s", w.info.Name, truncate(result.Output, 100))
	} else {
		log.Printf("[MCP Client] Tool %q failed: %s", w.info.Name, result.Error)
	}

	return result, nil
}

// Info returns the underlying MCPToolInfo.
func (w *MCPToolWrapper) Info() MCPToolInfo {
	return w.info
}

// truncate shortens a string for logging.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
