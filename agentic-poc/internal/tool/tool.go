// Package tool defines the Tool interface and built-in tool implementations.
package tool

import (
	"context"

	"agentic-poc/internal/provider"
)

// Tool defines the interface for tools that can be used by agents.
// Each tool has a name, description, parameter schema, and an Execute method.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what the tool does.
	Description() string

	// Parameters returns a JSON Schema object describing the tool's input parameters.
	Parameters() map[string]interface{}

	// Execute runs the tool with the provided arguments and returns the result.
	Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error)
}

// ToDefinition converts a Tool to a ToolDefinition for use in LLM requests.
func ToDefinition(t Tool) provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

// ToDefinitions converts a slice of Tools to ToolDefinitions.
func ToDefinitions(tools []Tool) []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = ToDefinition(t)
	}
	return defs
}
