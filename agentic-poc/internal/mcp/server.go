// Package mcp provides MCP (Model Context Protocol) server implementation
// that exposes tools via JSON-RPC 2.0 over stdio.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"agentic-poc/internal/tool"
)

// MCPServer handles incoming JSON-RPC requests and exposes tools via MCP protocol.
type MCPServer struct {
	tools   map[string]tool.Tool
	input   io.Reader
	output  io.Writer
	mu      sync.Mutex
	running bool

	// Server info
	name    string
	version string
}

// NewMCPServer creates a new MCP server with the given tools.
func NewMCPServer(name, version string, tools []tool.Tool) *MCPServer {
	toolMap := make(map[string]tool.Tool)
	for _, t := range tools {
		toolMap[t.Name()] = t
	}
	return &MCPServer{
		tools:   toolMap,
		name:    name,
		version: version,
	}
}

// Serve starts the MCP server, reading from input and writing to output.
// It blocks until the context is cancelled or an error occurs.
func (s *MCPServer) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.input = input
	s.output = output
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(0, -32700, "Parse error", nil)
			continue
		}

		s.handleRequest(ctx, &req)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest processes a single JSON-RPC request.
func (s *MCPServer) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	log.Printf("[MCP Server] Received request: method=%s id=%d", req.Method, req.ID)

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		log.Printf("[MCP Server] Client initialized")
		// Notification, no response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	default:
		log.Printf("[MCP Server] Unknown method: %s", req.Method)
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// handleInitialize handles the MCP initialize request.
func (s *MCPServer) handleInitialize(req *JSONRPCRequest) {
	log.Printf("[MCP Server] Initializing server: %s v%s", s.name, s.version)
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}
	s.sendResult(req.ID, result)
}

// handleToolsList handles the tools/list request.
func (s *MCPServer) handleToolsList(req *JSONRPCRequest) {
	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"inputSchema": t.Parameters(),
		})
	}

	log.Printf("[MCP Server] Listing %d tools", len(tools))
	result := map[string]interface{}{
		"tools": tools,
	}
	s.sendResult(req.ID, result)
}

// handleToolsCall handles the tools/call request.
func (s *MCPServer) handleToolsCall(ctx context.Context, req *JSONRPCRequest) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		log.Printf("[MCP Server] Invalid params for tools/call")
		s.sendError(req.ID, -32602, "Invalid params", nil)
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		log.Printf("[MCP Server] Missing tool name in tools/call")
		s.sendError(req.ID, -32602, "Missing tool name", nil)
		return
	}

	t, exists := s.tools[name]
	if !exists {
		log.Printf("[MCP Server] Unknown tool requested: %s", name)
		s.sendToolResult(req.ID, fmt.Sprintf("Unknown tool: %s", name), true)
		return
	}

	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = make(map[string]interface{})
	}

	log.Printf("[MCP Server] Executing tool %q with args: %v", name, args)

	result, err := t.Execute(ctx, args)
	if err != nil {
		log.Printf("[MCP Server] Tool %q execution error: %v", name, err)
		s.sendToolResult(req.ID, fmt.Sprintf("Tool execution error: %v", err), true)
		return
	}

	if !result.Success {
		log.Printf("[MCP Server] Tool %q returned error: %s", name, result.Error)
		s.sendToolResult(req.ID, result.Error, true)
		return
	}

	log.Printf("[MCP Server] Tool %q succeeded: %s", name, result.Output)
	s.sendToolResult(req.ID, result.Output, false)
}

// sendResult sends a successful JSON-RPC response.
func (s *MCPServer) sendResult(id int, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeResponse(resp)
}

// sendError sends a JSON-RPC error response.
func (s *MCPServer) sendError(id int, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeResponse(resp)
}

// sendToolResult sends a tool call result in MCP format.
func (s *MCPServer) sendToolResult(id int, content string, isError bool) {
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
		"isError": isError,
	}
	s.sendResult(id, result)
}

// writeResponse writes a JSON-RPC response to the output.
func (s *MCPServer) writeResponse(resp JSONRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.output.Write(append(data, '\n'))
}
