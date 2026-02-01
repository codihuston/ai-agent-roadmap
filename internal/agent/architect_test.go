package agent

import (
	"context"
	"testing"

	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
)

func TestNewArchitectAgent(t *testing.T) {
	mockProvider := &mockLLMProvider{}

	agent, finishPlanTool := NewArchitectAgent(mockProvider)

	// Verify agent is created
	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	// Verify finish_plan tool is returned
	if finishPlanTool == nil {
		t.Fatal("expected finishPlanTool to be returned, got nil")
	}

	// Verify system prompt is set
	if agent.systemPrompt != ArchitectSystemPrompt {
		t.Errorf("system prompt not set correctly")
	}

	// Verify finish_plan tool is registered
	tools := agent.GetTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	foundFinishPlan := false
	for _, tool := range tools {
		if tool.Name() == "finish_plan" {
			foundFinishPlan = true
			break
		}
	}
	if !foundFinishPlan {
		t.Error("finish_plan tool not found in agent tools")
	}
}

func TestArchitectAgent_GeneratesPlan(t *testing.T) {
	// Test that the architect agent can generate a plan using the finish_plan tool
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Create a hello world program",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Create main.go file with hello world code",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "main.go",
										"content": "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
									},
								},
							},
						},
					},
				},
			},
			{Text: "I have created a plan to build a hello world program."},
		},
	}

	agent, finishPlanTool := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create a hello world program", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the agent completed successfully
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify finish_plan tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCallsMade))
	}

	if result.ToolCallsMade[0].Name != "finish_plan" {
		t.Errorf("expected finish_plan tool call, got %s", result.ToolCallsMade[0].Name)
	}

	// Verify plan was captured
	if !finishPlanTool.HasCapturedPlan() {
		t.Error("expected plan to be captured")
	}

	capturedPlan := finishPlanTool.GetCapturedPlan()
	if capturedPlan == "" {
		t.Error("captured plan is empty")
	}
}

func TestArchitectAgent_SystemPromptIncluded(t *testing.T) {
	// Verify that the system prompt is included in LLM requests
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Here is my plan..."},
		},
	}

	agent, _ := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	_, err := agent.Run(context.Background(), "Build a web server", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify system prompt was sent to LLM
	if len(mockProvider.requests) == 0 {
		t.Fatal("expected at least one LLM request")
	}

	if mockProvider.requests[0].SystemPrompt != ArchitectSystemPrompt {
		t.Error("system prompt not included in LLM request")
	}
}

func TestArchitectAgent_ToolDefinitionIncluded(t *testing.T) {
	// Verify that the finish_plan tool definition is included in LLM requests
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Here is my plan..."},
		},
	}

	agent, _ := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	_, err := agent.Run(context.Background(), "Build a calculator", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool definition was sent to LLM
	if len(mockProvider.requests) == 0 {
		t.Fatal("expected at least one LLM request")
	}

	tools := mockProvider.requests[0].Tools
	if len(tools) != 1 {
		t.Errorf("expected 1 tool definition, got %d", len(tools))
	}

	if tools[0].Name != "finish_plan" {
		t.Errorf("expected finish_plan tool, got %s", tools[0].Name)
	}
}

func TestArchitectAgent_MultiStepPlan(t *testing.T) {
	// Test that the architect can create plans with multiple steps
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Create a REST API",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Create project structure",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "main.go",
										"content": "package main",
									},
								},
								map[string]interface{}{
									"description": "Create handler file",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "handlers.go",
										"content": "package main",
									},
								},
								map[string]interface{}{
									"description": "Create routes file",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "routes.go",
										"content": "package main",
									},
								},
							},
						},
					},
				},
			},
			{Text: "I have created a comprehensive plan for the REST API."},
		},
	}

	agent, finishPlanTool := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create a REST API", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify plan was captured
	if !finishPlanTool.HasCapturedPlan() {
		t.Error("expected plan to be captured")
	}
}

func TestArchitectAgent_InvalidPlanRejected(t *testing.T) {
	// Test that invalid plans (missing required fields) are rejected
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal":  "Create something",
							"steps": []interface{}{}, // Empty steps - invalid
						},
					},
				},
			},
			{Text: "I need to provide a valid plan with steps."},
		},
	}

	agent, finishPlanTool := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create something", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Plan should NOT be captured because it was invalid
	if finishPlanTool.HasCapturedPlan() {
		t.Error("expected invalid plan to be rejected")
	}
}

func TestArchitectAgent_PlanWithReadFileAction(t *testing.T) {
	// Test that the architect can create plans that include read_file actions
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Refactor existing code",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Read existing main.go",
									"action":      "read_file",
									"parameters": map[string]interface{}{
										"path": "main.go",
									},
								},
								map[string]interface{}{
									"description": "Update main.go with refactored code",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "main.go",
										"content": "// refactored code",
									},
								},
							},
						},
					},
				},
			},
			{Text: "Plan created for refactoring."},
		},
	}

	agent, finishPlanTool := NewArchitectAgent(mockProvider)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Refactor existing code", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify plan was captured
	if !finishPlanTool.HasCapturedPlan() {
		t.Error("expected plan to be captured")
	}
}
