package mcp

import (
	"context"
	"testing"

	"agentic-poc/internal/provider"
)

func TestMCPToolWrapper_Name(t *testing.T) {
	client := NewMockMCPClient()
	info := MCPToolInfo{
		Name:        "test_tool",
		Description: "A test tool",
	}
	wrapper := NewMCPToolWrapper(client, info)

	if got := wrapper.Name(); got != "test_tool" {
		t.Errorf("Name() = %q, want %q", got, "test_tool")
	}
}

func TestMCPToolWrapper_Description(t *testing.T) {
	client := NewMockMCPClient()
	info := MCPToolInfo{
		Name:        "test_tool",
		Description: "A test tool for testing",
	}
	wrapper := NewMCPToolWrapper(client, info)

	if got := wrapper.Description(); got != "A test tool for testing" {
		t.Errorf("Description() = %q, want %q", got, "A test tool for testing")
	}
}

func TestMCPToolWrapper_Parameters(t *testing.T) {
	tests := []struct {
		name        string
		inputSchema map[string]interface{}
		wantType    string
	}{
		{
			name: "with schema",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantType: "object",
		},
		{
			name:        "nil schema returns default",
			inputSchema: nil,
			wantType:    "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMockMCPClient()
			info := MCPToolInfo{
				Name:        "test_tool",
				InputSchema: tt.inputSchema,
			}
			wrapper := NewMCPToolWrapper(client, info)

			params := wrapper.Parameters()
			if params == nil {
				t.Fatal("Parameters() returned nil")
			}

			typeVal, ok := params["type"].(string)
			if !ok {
				t.Fatal("Parameters() missing 'type' field")
			}
			if typeVal != tt.wantType {
				t.Errorf("Parameters()['type'] = %q, want %q", typeVal, tt.wantType)
			}
		})
	}
}

func TestMCPToolWrapper_Execute(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]interface{}
		callResult *provider.ToolResult
		callErr    error
		wantOutput string
		wantErr    bool
	}{
		{
			name: "successful execution",
			args: map[string]interface{}{"input": "test"},
			callResult: &provider.ToolResult{
				Success: true,
				Output:  "Result: test",
			},
			wantOutput: "Result: test",
			wantErr:    false,
		},
		{
			name: "execution with error result",
			args: map[string]interface{}{"input": "bad"},
			callResult: &provider.ToolResult{
				Success: false,
				Error:   "Invalid input",
			},
			wantErr: false,
		},
		{
			name: "empty args",
			args: map[string]interface{}{},
			callResult: &provider.ToolResult{
				Success: true,
				Output:  "No args",
			},
			wantOutput: "No args",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				if tt.callErr != nil {
					return nil, tt.callErr
				}
				return tt.callResult, nil
			}

			info := MCPToolInfo{Name: "test_tool"}
			wrapper := NewMCPToolWrapper(client, info)

			result, err := wrapper.Execute(context.Background(), tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != nil {
				if tt.wantOutput != "" && result.Output != tt.wantOutput {
					t.Errorf("Execute() output = %q, want %q", result.Output, tt.wantOutput)
				}
			}
		})
	}
}

func TestMCPToolWrapper_Info(t *testing.T) {
	client := NewMockMCPClient()
	info := MCPToolInfo{
		Name:        "test_tool",
		Description: "Test description",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	wrapper := NewMCPToolWrapper(client, info)

	got := wrapper.Info()
	if got.Name != info.Name {
		t.Errorf("Info().Name = %q, want %q", got.Name, info.Name)
	}
	if got.Description != info.Description {
		t.Errorf("Info().Description = %q, want %q", got.Description, info.Description)
	}
}

func TestMCPToolWrapper_ForwardsCorrectToolName(t *testing.T) {
	// Property 22: MCP Tool Calls Forward Correctly
	var capturedName string
	var capturedArgs map[string]interface{}

	client := NewMockMCPClient()
	client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
		capturedName = name
		capturedArgs = args
		return &provider.ToolResult{Success: true, Output: "ok"}, nil
	}

	info := MCPToolInfo{Name: "specific_tool_name"}
	wrapper := NewMCPToolWrapper(client, info)

	args := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	_, err := wrapper.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedName != "specific_tool_name" {
		t.Errorf("Tool name not forwarded correctly: got %q, want %q", capturedName, "specific_tool_name")
	}

	if capturedArgs["key1"] != "value1" {
		t.Errorf("Args not forwarded correctly: key1 = %v, want %v", capturedArgs["key1"], "value1")
	}
	if capturedArgs["key2"] != 42 {
		t.Errorf("Args not forwarded correctly: key2 = %v, want %v", capturedArgs["key2"], 42)
	}
}
