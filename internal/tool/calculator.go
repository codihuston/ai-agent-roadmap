package tool

import (
	"context"
	"fmt"

	"agentic-poc/internal/provider"
)

// CalculatorTool performs basic arithmetic operations.
type CalculatorTool struct{}

// NewCalculatorTool creates a new CalculatorTool instance.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// Name returns the tool's identifier.
func (c *CalculatorTool) Name() string {
	return "calculator"
}

// Description returns what the tool does.
func (c *CalculatorTool) Description() string {
	return "Performs basic arithmetic operations: add, subtract, multiply, divide"
}

// Parameters returns the JSON Schema for the tool's input.
func (c *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
				"description": "The arithmetic operation to perform",
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "The first operand",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "The second operand",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

// Execute performs the arithmetic operation.
func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing or invalid 'operation' argument",
		}, nil
	}

	a, err := toFloat64(args["a"])
	if err != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid 'a' argument: %v", err),
		}, nil
	}

	b, err := toFloat64(args["b"])
	if err != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid 'b' argument: %v", err),
		}, nil
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return &provider.ToolResult{
				Success: false,
				Error:   "division by zero",
			}, nil
		}
		result = a / b
	default:
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown operation: %s", operation),
		}, nil
	}

	return &provider.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("%v", result),
	}, nil
}

// toFloat64 converts an interface{} to float64.
// Handles both float64 and int types that may come from JSON parsing.
func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}
