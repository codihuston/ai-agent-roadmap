package tool

import (
	"context"
	"testing"
)

func TestCalculatorTool_Name(t *testing.T) {
	calc := NewCalculatorTool()
	if calc.Name() != "calculator" {
		t.Errorf("expected name 'calculator', got '%s'", calc.Name())
	}
}

func TestCalculatorTool_Description(t *testing.T) {
	calc := NewCalculatorTool()
	if calc.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestCalculatorTool_Parameters(t *testing.T) {
	calc := NewCalculatorTool()
	params := calc.Parameters()

	if params["type"] != "object" {
		t.Error("expected parameters type to be 'object'")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	if _, ok := props["operation"]; !ok {
		t.Error("expected 'operation' property")
	}
	if _, ok := props["a"]; !ok {
		t.Error("expected 'a' property")
	}
	if _, ok := props["b"]; !ok {
		t.Error("expected 'b' property")
	}
}

func TestCalculatorTool_Execute(t *testing.T) {
	calc := NewCalculatorTool()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantOut   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "add positive numbers",
			args:    map[string]interface{}{"operation": "add", "a": 5.0, "b": 3.0},
			wantOut: "8",
		},
		{
			name:    "add negative numbers",
			args:    map[string]interface{}{"operation": "add", "a": -5.0, "b": -3.0},
			wantOut: "-8",
		},
		{
			name:    "add mixed numbers",
			args:    map[string]interface{}{"operation": "add", "a": 5.0, "b": -3.0},
			wantOut: "2",
		},
		{
			name:    "subtract positive numbers",
			args:    map[string]interface{}{"operation": "subtract", "a": 10.0, "b": 4.0},
			wantOut: "6",
		},
		{
			name:    "subtract resulting in negative",
			args:    map[string]interface{}{"operation": "subtract", "a": 4.0, "b": 10.0},
			wantOut: "-6",
		},
		{
			name:    "multiply positive numbers",
			args:    map[string]interface{}{"operation": "multiply", "a": 6.0, "b": 7.0},
			wantOut: "42",
		},
		{
			name:    "multiply by zero",
			args:    map[string]interface{}{"operation": "multiply", "a": 100.0, "b": 0.0},
			wantOut: "0",
		},
		{
			name:    "multiply negative numbers",
			args:    map[string]interface{}{"operation": "multiply", "a": -3.0, "b": -4.0},
			wantOut: "12",
		},
		{
			name:    "divide positive numbers",
			args:    map[string]interface{}{"operation": "divide", "a": 20.0, "b": 4.0},
			wantOut: "5",
		},
		{
			name:    "divide with decimal result",
			args:    map[string]interface{}{"operation": "divide", "a": 7.0, "b": 2.0},
			wantOut: "3.5",
		},
		{
			name:      "divide by zero",
			args:      map[string]interface{}{"operation": "divide", "a": 10.0, "b": 0.0},
			wantErr:   true,
			errSubstr: "division by zero",
		},
		{
			name:      "unknown operation",
			args:      map[string]interface{}{"operation": "modulo", "a": 10.0, "b": 3.0},
			wantErr:   true,
			errSubstr: "unknown operation",
		},
		{
			name:      "missing operation",
			args:      map[string]interface{}{"a": 10.0, "b": 3.0},
			wantErr:   true,
			errSubstr: "operation",
		},
		{
			name:      "missing operand a",
			args:      map[string]interface{}{"operation": "add", "b": 3.0},
			wantErr:   true,
			errSubstr: "'a'",
		},
		{
			name:      "missing operand b",
			args:      map[string]interface{}{"operation": "add", "a": 10.0},
			wantErr:   true,
			errSubstr: "'b'",
		},
		{
			name:      "invalid operand type",
			args:      map[string]interface{}{"operation": "add", "a": "ten", "b": 3.0},
			wantErr:   true,
			errSubstr: "'a'",
		},
		{
			name:    "integer operands",
			args:    map[string]interface{}{"operation": "add", "a": 5, "b": 3},
			wantOut: "8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(ctx, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result.Success {
					t.Errorf("expected failure, got success with output: %s", result.Output)
				}
				if tt.errSubstr != "" && result.Error == "" {
					t.Errorf("expected error containing '%s', got empty error", tt.errSubstr)
				}
			} else {
				if !result.Success {
					t.Errorf("expected success, got error: %s", result.Error)
				}
				if result.Output != tt.wantOut {
					t.Errorf("expected output '%s', got '%s'", tt.wantOut, result.Output)
				}
			}
		})
	}
}

func TestCalculatorTool_ImplementsInterface(t *testing.T) {
	var _ Tool = (*CalculatorTool)(nil)
}
