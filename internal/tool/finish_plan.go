package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"agentic-poc/internal/provider"
)

// FinishPlanTool captures the Architect agent's completed plan.
// It stores the plan for later retrieval by the orchestrator.
type FinishPlanTool struct {
	capturedPlan string
	mu           sync.RWMutex
}

// NewFinishPlanTool creates a new FinishPlanTool instance.
func NewFinishPlanTool() *FinishPlanTool {
	return &FinishPlanTool{}
}

// Name returns the tool's identifier.
func (f *FinishPlanTool) Name() string {
	return "finish_plan"
}

// Description returns what the tool does.
func (f *FinishPlanTool) Description() string {
	return "Completes the planning phase by outputting the final plan. Call this when you have finished creating the implementation plan."
}

// Parameters returns the JSON Schema for the tool's input.
func (f *FinishPlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"goal": map[string]interface{}{
				"type":        "string",
				"description": "The high-level goal this plan addresses",
			},
			"steps": map[string]interface{}{
				"type":        "array",
				"description": "The ordered list of steps to execute",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"description": map[string]interface{}{
							"type":        "string",
							"description": "A description of what this step accomplishes",
						},
						"action": map[string]interface{}{
							"type":        "string",
							"description": "The action to perform (e.g., 'write_file', 'read_file')",
						},
						"parameters": map[string]interface{}{
							"type":        "object",
							"description": "Parameters for the action",
						},
					},
					"required": []string{"description", "action"},
				},
			},
		},
		"required": []string{"goal", "steps"},
	}
}

// Execute captures the plan and returns success.
func (f *FinishPlanTool) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	// Validate goal
	goal, ok := args["goal"].(string)
	if !ok || goal == "" {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing or invalid 'goal' argument",
		}, nil
	}

	// Validate steps
	stepsRaw, ok := args["steps"]
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing 'steps' argument",
		}, nil
	}

	steps, ok := stepsRaw.([]interface{})
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "invalid 'steps' argument: expected array",
		}, nil
	}

	if len(steps) == 0 {
		return &provider.ToolResult{
			Success: false,
			Error:   "plan must have at least one step",
		}, nil
	}

	// Validate each step
	for i, stepRaw := range steps {
		step, ok := stepRaw.(map[string]interface{})
		if !ok {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("step %d is not a valid object", i+1),
			}, nil
		}

		desc, ok := step["description"].(string)
		if !ok || desc == "" {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("step %d is missing required field: description", i+1),
			}, nil
		}

		action, ok := step["action"].(string)
		if !ok || action == "" {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("step %d is missing required field: action", i+1),
			}, nil
		}
	}

	// Serialize the plan to JSON for storage
	planJSON, err := json.Marshal(args)
	if err != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to serialize plan: %v", err),
		}, nil
	}

	// Store the captured plan
	f.mu.Lock()
	f.capturedPlan = string(planJSON)
	f.mu.Unlock()

	return &provider.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Plan captured successfully with %d steps", len(steps)),
	}, nil
}

// GetCapturedPlan returns the captured plan JSON string.
// Returns empty string if no plan has been captured.
func (f *FinishPlanTool) GetCapturedPlan() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.capturedPlan
}

// ClearCapturedPlan clears any previously captured plan.
func (f *FinishPlanTool) ClearCapturedPlan() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.capturedPlan = ""
}

// HasCapturedPlan returns true if a plan has been captured.
func (f *FinishPlanTool) HasCapturedPlan() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.capturedPlan != ""
}
