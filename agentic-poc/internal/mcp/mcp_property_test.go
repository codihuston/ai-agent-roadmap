package mcp

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"agentic-poc/internal/provider"
)

// TestMCPToolInterfaceCompliance validates Property 21:
// MCP Tools Implement Tool Interface
//
// **Validates: Requirements 11.4**
//
// For any tool discovered from an MCP server, the MCPToolWrapper SHALL correctly
// implement the Tool interface, returning the tool's name, description, and
// parameters from the MCP server's metadata.
func TestMCPToolInterfaceCompliance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("MCPToolWrapper returns correct name from MCP metadata", prop.ForAll(
		func(name string) bool {
			client := NewMockMCPClient()
			info := MCPToolInfo{Name: name, Description: "test"}
			wrapper := NewMCPToolWrapper(client, info)
			return wrapper.Name() == name
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("MCPToolWrapper returns correct description from MCP metadata", prop.ForAll(
		func(desc string) bool {
			client := NewMockMCPClient()
			info := MCPToolInfo{Name: "test", Description: desc}
			wrapper := NewMCPToolWrapper(client, info)
			return wrapper.Description() == desc
		},
		gen.AnyString(),
	))

	properties.Property("MCPToolWrapper returns valid parameters schema", prop.ForAll(
		func(schemaIdx int) bool {
			schemas := []map[string]interface{}{
				{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
				{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{
							"type":        "string",
							"description": "Input value",
						},
					},
				},
				{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []interface{}{"path"},
				},
			}
			schema := schemas[schemaIdx%len(schemas)]

			client := NewMockMCPClient()
			info := MCPToolInfo{
				Name:        "test",
				Description: "test",
				InputSchema: schema,
			}
			wrapper := NewMCPToolWrapper(client, info)
			params := wrapper.Parameters()

			// Parameters should never be nil
			if params == nil {
				return false
			}

			// Should have a type field
			_, hasType := params["type"]
			return hasType
		},
		gen.IntRange(0, 100),
	))

	properties.Property("MCPToolWrapper with nil schema returns default object schema", prop.ForAll(
		func(name string) bool {
			client := NewMockMCPClient()
			info := MCPToolInfo{
				Name:        name,
				Description: "test",
				InputSchema: nil,
			}
			wrapper := NewMCPToolWrapper(client, info)
			params := wrapper.Parameters()

			if params == nil {
				return false
			}

			typeVal, ok := params["type"].(string)
			if !ok {
				return false
			}

			return typeVal == "object"
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t)
}

// TestMCPToolCallForwarding validates Property 22:
// MCP Tool Calls Forward Correctly
//
// **Validates: Requirements 11.5**
//
// For any tool call made through an MCPToolWrapper, the call SHALL be forwarded
// to the MCP server with the exact tool name and arguments, and the result
// SHALL be returned unchanged.
func TestMCPToolCallForwarding(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Tool name is forwarded exactly to MCP server", prop.ForAll(
		func(toolName string) bool {
			var capturedName string
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				capturedName = name
				return &provider.ToolResult{Success: true}, nil
			}

			info := MCPToolInfo{Name: toolName}
			wrapper := NewMCPToolWrapper(client, info)

			_, err := wrapper.Execute(context.Background(), nil)
			if err != nil {
				return false
			}

			return capturedName == toolName
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.Property("Arguments are forwarded exactly to MCP server", prop.ForAll(
		func(key, value string) bool {
			var capturedArgs map[string]interface{}
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				capturedArgs = args
				return &provider.ToolResult{Success: true}, nil
			}

			info := MCPToolInfo{Name: "test"}
			wrapper := NewMCPToolWrapper(client, info)

			args := map[string]interface{}{key: value}
			_, err := wrapper.Execute(context.Background(), args)
			if err != nil {
				return false
			}

			capturedValue, ok := capturedArgs[key].(string)
			return ok && capturedValue == value
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AnyString(),
	))

	properties.Property("Result output is returned unchanged", prop.ForAll(
		func(output string) bool {
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				return &provider.ToolResult{Success: true, Output: output}, nil
			}

			info := MCPToolInfo{Name: "test"}
			wrapper := NewMCPToolWrapper(client, info)

			result, err := wrapper.Execute(context.Background(), nil)
			if err != nil {
				return false
			}

			return result.Output == output
		},
		gen.AnyString(),
	))

	properties.Property("Result success flag is returned unchanged", prop.ForAll(
		func(success bool) bool {
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				return &provider.ToolResult{Success: success, Output: "test"}, nil
			}

			info := MCPToolInfo{Name: "test"}
			wrapper := NewMCPToolWrapper(client, info)

			result, err := wrapper.Execute(context.Background(), nil)
			if err != nil {
				return false
			}

			return result.Success == success
		},
		gen.Bool(),
	))

	properties.Property("Result error message is returned unchanged", prop.ForAll(
		func(errMsg string) bool {
			client := NewMockMCPClient()
			client.CallToolFunc = func(ctx context.Context, name string, args map[string]interface{}) (*provider.ToolResult, error) {
				return &provider.ToolResult{Success: false, Error: errMsg}, nil
			}

			info := MCPToolInfo{Name: "test"}
			wrapper := NewMCPToolWrapper(client, info)

			result, err := wrapper.Execute(context.Background(), nil)
			if err != nil {
				return false
			}

			return result.Error == errMsg
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}
