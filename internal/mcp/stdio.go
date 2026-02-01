package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"sync/atomic"

	"agentic-poc/internal/provider"
)

// StdioMCPClient spawns an MCP server as a subprocess and communicates
// via JSON-RPC 2.0 over stdin/stdout.
type StdioMCPClient struct {
	command string
	args    []string
	env     map[string]string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bufio.Reader

	requestID atomic.Int64
	mu        sync.Mutex
	connected bool
}

// NewStdioMCPClient creates a new StdioMCPClient with the given command and arguments.
func NewStdioMCPClient(command string, args []string, env map[string]string) *StdioMCPClient {
	return &StdioMCPClient{
		command: command,
		args:    args,
		env:     env,
	}
}

// Connect starts the MCP server subprocess and initializes the connection.
func (c *StdioMCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	c.cmd = exec.CommandContext(ctx, c.command, c.args...)

	// Set environment variables
	if len(c.env) > 0 {
		c.cmd.Env = c.cmd.Environ()
		for k, v := range c.env {
			c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)

	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	c.stderr = bufio.NewReader(stderr)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start goroutine to read and log server stderr
	go c.logServerStderr()

	// Send initialize request
	if err := c.initialize(ctx); err != nil {
		c.cmd.Process.Kill()
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	c.connected = true
	return nil
}

// initialize sends the MCP initialize request and waits for response.
func (c *StdioMCPClient) initialize(ctx context.Context) error {
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "agentic-poc",
			"version": "1.0.0",
		},
	}

	resp, err := c.sendRequest(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	notification := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	if err := c.writeMessage(notification); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// ListTools retrieves the list of available tools from the MCP server.
func (c *StdioMCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	resp, err := c.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	// Parse the result
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	toolsRaw, ok := resultMap["tools"]
	if !ok {
		return []MCPToolInfo{}, nil
	}

	toolsList, ok := toolsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected tools type: %T", toolsRaw)
	}

	tools := make([]MCPToolInfo, 0, len(toolsList))
	for _, t := range toolsList {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		info := MCPToolInfo{
			Name:        getString(toolMap, "name"),
			Description: getString(toolMap, "description"),
		}

		if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
			info.InputSchema = schema
		}

		tools = append(tools, info)
	}

	return tools, nil
}

// CallTool invokes a tool on the MCP server with the given arguments.
func (c *StdioMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool call failed: %v", err),
		}, nil
	}

	if resp.Error != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   resp.Error.Message,
		}, nil
	}

	// Parse the result
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unexpected result type: %T", resp.Result),
		}, nil
	}

	// Check for isError flag
	isError, _ := resultMap["isError"].(bool)

	// Extract content
	content := extractContent(resultMap)

	if isError {
		return &provider.ToolResult{
			Success: false,
			Error:   content,
		}, nil
	}

	return &provider.ToolResult{
		Success: true,
		Output:  content,
	}, nil
}

// Close terminates the MCP server subprocess.
func (c *StdioMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false

	if c.stdin != nil {
		c.stdin.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}

	return nil
}

// logServerStderr reads and logs the server's stderr output.
func (c *StdioMCPClient) logServerStderr() {
	for {
		line, err := c.stderr.ReadString('\n')
		if err != nil {
			return // EOF or error, stop logging
		}
		if line != "" {
			log.Printf("[MCP Server stderr] %s", line)
		}
	}
}

// sendRequest sends a JSON-RPC request and waits for the response.
func (c *StdioMCPClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	id := int(c.requestID.Add(1))

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	resp, err := c.readResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}

// writeMessage writes a JSON-RPC message to stdin.
func (c *StdioMCPClient) writeMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write message with newline delimiter
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// readResponse reads a JSON-RPC response from stdout.
func (c *StdioMCPClient) readResponse(ctx context.Context) (*JSONRPCResponse, error) {
	// Read line from stdout
	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// getString safely extracts a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractContent extracts text content from an MCP tool result.
func extractContent(resultMap map[string]interface{}) string {
	contentRaw, ok := resultMap["content"]
	if !ok {
		return ""
	}

	contentList, ok := contentRaw.([]interface{})
	if !ok {
		return fmt.Sprintf("%v", contentRaw)
	}

	var result string
	for _, item := range contentList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if text, ok := itemMap["text"].(string); ok {
			if result != "" {
				result += "\n"
			}
			result += text
		}
	}

	return result
}
