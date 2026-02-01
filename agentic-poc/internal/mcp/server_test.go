package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agentic-poc/internal/tool"
)

func TestMCPServer_Initialize(t *testing.T) {
	server := NewMCPServer("test-server", "1.0.0", nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected serverInfo map, got %T", result["serverInfo"])
	}

	if serverInfo["name"] != "test-server" {
		t.Errorf("Expected server name test-server, got %v", serverInfo["name"])
	}
}

func TestMCPServer_ToolsList(t *testing.T) {
	calc := tool.NewCalculatorTool()
	server := NewMCPServer("test-server", "1.0.0", []tool.Tool{calc})

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools array, got %T", result["tools"])
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	toolInfo, ok := tools[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tool map, got %T", tools[0])
	}

	if toolInfo["name"] != "calculator" {
		t.Errorf("Expected tool name calculator, got %v", toolInfo["name"])
	}
}

func TestMCPServer_ToolsCall_Success(t *testing.T) {
	calc := tool.NewCalculatorTool()
	server := NewMCPServer("test-server", "1.0.0", []tool.Tool{calc})

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"calculator","arguments":{"operation":"add","a":2,"b":3}}}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	isError, _ := result["isError"].(bool)
	if isError {
		t.Error("Expected isError to be false")
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array, got %T", result["content"])
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected content item map, got %T", content[0])
	}

	if contentItem["text"] != "5" {
		t.Errorf("Expected result 5, got %v", contentItem["text"])
	}
}

func TestMCPServer_ToolsCall_UnknownTool(t *testing.T) {
	server := NewMCPServer("test-server", "1.0.0", nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"unknown","arguments":{}}}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected JSON-RPC error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError to be true for unknown tool")
	}
}

func TestMCPServer_ToolsCall_ToolError(t *testing.T) {
	calc := tool.NewCalculatorTool()
	server := NewMCPServer("test-server", "1.0.0", []tool.Tool{calc})

	// Division by zero
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"calculator","arguments":{"operation":"divide","a":10,"b":0}}}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected JSON-RPC error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected isError to be true for division by zero")
	}
}

func TestMCPServer_UnknownMethod(t *testing.T) {
	server := NewMCPServer("test-server", "1.0.0", nil)

	input := `{"jsonrpc":"2.0","id":1,"method":"unknown/method"}
`
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go server.Serve(ctx, strings.NewReader(input), &output)
	time.Sleep(50 * time.Millisecond)

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}
