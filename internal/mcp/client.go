// Package mcp provides MCP (Model Context Protocol) integration for connecting
// to external tool servers.
package mcp

import (
	"context"

	"agentic-poc/internal/provider"
)

// MCPToolInfo represents tool metadata from an MCP server.
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPClient handles communication with an MCP server.
type MCPClient interface {
	// Connect establishes a connection to the MCP server.
	Connect(ctx context.Context) error

	// ListTools retrieves the list of available tools from the MCP server.
	ListTools(ctx context.Context) ([]MCPToolInfo, error)

	// CallTool invokes a tool on the MCP server with the given arguments.
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error)

	// Close terminates the connection to the MCP server.
	Close() error
}

// JSONRPCRequest represents a JSON-RPC 2.0 request message.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response message.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents an error in a JSON-RPC 2.0 response.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface for JSONRPCError.
func (e *JSONRPCError) Error() string {
	return e.Message
}
