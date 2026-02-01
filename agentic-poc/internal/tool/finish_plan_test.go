package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFinishPlanTool_Name(t *testing.T) {
	tool := NewFinishPlanTool()
	if tool.Name() != "finish_plan" {
		t.Errorf("expected name 'finish_plan', got '%s'", tool.Name())
	}
}

func TestFinishPlanTool_Description(t *testing.T) {
	tool := NewFinishPlanTool()
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestFinishPlanTool_Parameters(t *testing.T) {
	tool := NewFinishPlanTool()
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Error("expected parameters type to be 'object'")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	if _, ok := props["goal"]; !ok {
		t.Error("expected 'goal' property")
	}
	if _, ok := props["steps"]; !ok {
		t.Error("expected 'steps' property")
	}
}

func TestFinishPlanTool_Execute_ValidPlan(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Create a hello world program",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Create main.go file",
				"action":      "write_file",
				"parameters": map[string]interface{}{
					"path":    "main.go",
					"content": "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
				},
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify plan was captured
	if !tool.HasCapturedPlan() {
		t.Error("expected plan to be captured")
	}

	captured := tool.GetCapturedPlan()
	if captured == "" {
		t.Error("expected non-empty captured plan")
	}

	// Verify captured plan is valid JSON
	var capturedPlan map[string]interface{}
	if err := json.Unmarshal([]byte(captured), &capturedPlan); err != nil {
		t.Errorf("captured plan is not valid JSON: %v", err)
	}
}

func TestFinishPlanTool_Execute_MultipleSteps(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Build a web server",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Create main.go",
				"action":      "write_file",
			},
			map[string]interface{}{
				"description": "Create handler.go",
				"action":      "write_file",
			},
			map[string]interface{}{
				"description": "Create config.json",
				"action":      "write_file",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestFinishPlanTool_Execute_MissingGoal(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Do something",
				"action":      "write_file",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing goal")
	}
}

func TestFinishPlanTool_Execute_EmptyGoal(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Do something",
				"action":      "write_file",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for empty goal")
	}
}

func TestFinishPlanTool_Execute_MissingSteps(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Do something",
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing steps")
	}
}

func TestFinishPlanTool_Execute_EmptySteps(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal":  "Do something",
		"steps": []interface{}{},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for empty steps")
	}
}

func TestFinishPlanTool_Execute_StepMissingDescription(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Do something",
		"steps": []interface{}{
			map[string]interface{}{
				"action": "write_file",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for step missing description")
	}
}

func TestFinishPlanTool_Execute_StepMissingAction(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Do something",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Do something",
			},
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for step missing action")
	}
}

func TestFinishPlanTool_Execute_InvalidStepType(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	args := map[string]interface{}{
		"goal": "Do something",
		"steps": []interface{}{
			"not a map",
		},
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for invalid step type")
	}
}

func TestFinishPlanTool_ClearCapturedPlan(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	// First capture a plan
	args := map[string]interface{}{
		"goal": "Test goal",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Test step",
				"action":      "test_action",
			},
		},
	}

	_, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tool.HasCapturedPlan() {
		t.Error("expected plan to be captured")
	}

	// Clear the plan
	tool.ClearCapturedPlan()

	if tool.HasCapturedPlan() {
		t.Error("expected plan to be cleared")
	}
	if tool.GetCapturedPlan() != "" {
		t.Error("expected empty captured plan after clear")
	}
}

func TestFinishPlanTool_OverwritesPreviousPlan(t *testing.T) {
	tool := NewFinishPlanTool()
	ctx := context.Background()

	// Capture first plan
	args1 := map[string]interface{}{
		"goal": "First goal",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "First step",
				"action":      "first_action",
			},
		},
	}

	_, err := tool.Execute(ctx, args1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	firstPlan := tool.GetCapturedPlan()

	// Capture second plan
	args2 := map[string]interface{}{
		"goal": "Second goal",
		"steps": []interface{}{
			map[string]interface{}{
				"description": "Second step",
				"action":      "second_action",
			},
		},
	}

	_, err = tool.Execute(ctx, args2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	secondPlan := tool.GetCapturedPlan()

	if firstPlan == secondPlan {
		t.Error("expected second plan to overwrite first plan")
	}

	// Verify second plan contains the new goal
	var plan map[string]interface{}
	if err := json.Unmarshal([]byte(secondPlan), &plan); err != nil {
		t.Fatalf("failed to parse captured plan: %v", err)
	}
	if plan["goal"] != "Second goal" {
		t.Errorf("expected goal 'Second goal', got '%v'", plan["goal"])
	}
}

func TestFinishPlanTool_ImplementsInterface(t *testing.T) {
	var _ Tool = (*FinishPlanTool)(nil)
}
