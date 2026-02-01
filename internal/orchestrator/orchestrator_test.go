// Package orchestrator provides tests for the orchestrator.
package orchestrator

import (
	"context"
	"errors"
	"testing"

	"agentic-poc/internal/provider"
)

// MockLLMProvider implements provider.LLMProvider for testing.
type MockLLMProvider struct {
	responses []provider.LLMResponse
	callCount int
	err       error
}

func (m *MockLLMProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.callCount >= len(m.responses) {
		return &provider.LLMResponse{Text: "No more responses"}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *MockLLMProvider) Name() string {
	return "mock"
}

// TestSuccessfulArchitectCoderWorkflow tests the happy path where Architect creates a plan
// and Coder executes it successfully.
// Validates: Requirements 7.1-7.6, Properties 15, 16, 17
func TestSuccessfulArchitectCoderWorkflow(t *testing.T) {
	// Mock responses:
	// 1. Architect calls finish_plan with a valid plan
	// 2. Coder processes the plan and returns success
	mockProvider := &MockLLMProvider{
		responses: []provider.LLMResponse{
			// Architect response - calls finish_plan tool with proper arguments
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Test goal",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Step 1",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "test.txt",
										"content": "hello",
									},
								},
							},
						},
					},
				},
			},
			// Architect final response after tool result
			{
				Text: "Plan created successfully",
			},
			// Coder response - no tool calls, just completion
			{
				Text: "Plan executed successfully",
			},
		},
	}

	orch := NewOrchestrator(mockProvider, "/tmp/test")

	// Verify initial state is idle
	state := orch.State()
	if state.Phase != PhaseIdle {
		t.Errorf("Expected initial phase to be idle, got %s", state.Phase)
	}

	result, err := orch.Run(context.Background(), "Test goal")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success=true, got false. Error: %s", result.Error)
	}

	if result.Plan == nil {
		t.Fatal("Expected plan to be set")
	}

	if result.Plan.Goal != "Test goal" {
		t.Errorf("Expected plan goal 'Test goal', got '%s'", result.Plan.Goal)
	}

	if len(result.Plan.Steps) != 1 {
		t.Errorf("Expected 1 plan step, got %d", len(result.Plan.Steps))
	}

	// Verify final state is complete
	state = orch.State()
	if state.Phase != PhaseComplete {
		t.Errorf("Expected final phase to be complete, got %s", state.Phase)
	}
}

// TestWorkflowStateTransitions verifies that the orchestrator correctly transitions
// through phases: idle -> planning -> executing -> complete
// Validates: Property 16
func TestWorkflowStateTransitions(t *testing.T) {
	mockProvider := &MockLLMProvider{
		responses: []provider.LLMResponse{
			// Architect calls finish_plan
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Test",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Do something",
									"action":      "test",
									"parameters":  map[string]interface{}{},
								},
							},
						},
					},
				},
			},
			// Architect final response
			{Text: "Done planning"},
			// Coder final response
			{Text: "Done executing"},
		},
	}

	orch := NewOrchestrator(mockProvider, "/tmp/test")

	// Initial state should be idle
	if orch.State().Phase != PhaseIdle {
		t.Errorf("Expected initial phase idle, got %s", orch.State().Phase)
	}

	result, err := orch.Run(context.Background(), "Test")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}

	// Final state should be complete
	finalState := orch.State()
	if finalState.Phase != PhaseComplete {
		t.Errorf("Expected final phase complete, got %s", finalState.Phase)
	}

	// Plan should be stored in state
	if finalState.Plan == nil {
		t.Error("Expected plan to be stored in state")
	}
}

// TestArchitectFailureSetsFailedPhase verifies that when the Architect agent fails,
// the orchestrator sets the phase to Failed and returns an appropriate error.
// Validates: Property 17
func TestArchitectFailureSetsFailedPhase(t *testing.T) {
	mockProvider := &MockLLMProvider{
		err: errors.New("LLM API error"),
	}

	orch := NewOrchestrator(mockProvider, "/tmp/test")

	result, err := orch.Run(context.Background(), "Test goal")

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when architect fails")
	}

	// Error should mention architect
	if !contains(err.Error(), "architect") {
		t.Errorf("Expected error to mention 'architect', got: %v", err)
	}

	// Result should indicate failure
	if result.Success {
		t.Error("Expected success=false when architect fails")
	}

	// Phase should be Failed
	state := orch.State()
	if state.Phase != PhaseFailed {
		t.Errorf("Expected phase Failed, got %s", state.Phase)
	}

	// Error should be stored in state
	if state.Error == "" {
		t.Error("Expected error to be stored in state")
	}
}

// TestMissingPlanReturnsError verifies that when the Architect doesn't call finish_plan,
// the orchestrator returns an appropriate error.
// Validates: Property 15
func TestMissingPlanReturnsError(t *testing.T) {
	// Architect responds without calling finish_plan
	mockProvider := &MockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "I'll help you with that goal"},
		},
	}

	orch := NewOrchestrator(mockProvider, "/tmp/test")

	result, err := orch.Run(context.Background(), "Test goal")

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when architect doesn't produce a plan")
	}

	// Error should mention plan
	if !contains(err.Error(), "plan") {
		t.Errorf("Expected error to mention 'plan', got: %v", err)
	}

	// Result should indicate failure
	if result.Success {
		t.Error("Expected success=false when no plan produced")
	}

	// Phase should be Failed
	state := orch.State()
	if state.Phase != PhaseFailed {
		t.Errorf("Expected phase Failed, got %s", state.Phase)
	}
}

// FailAfterNCallsProvider fails after N successful calls.
type FailAfterNCallsProvider struct {
	responses []provider.LLMResponse
	callCount int
	failAfter int
	failError error
}

func (m *FailAfterNCallsProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	m.callCount++
	if m.callCount > m.failAfter {
		return nil, m.failError
	}
	if m.callCount-1 >= len(m.responses) {
		return &provider.LLMResponse{Text: "No more responses"}, nil
	}
	return &m.responses[m.callCount-1], nil
}

func (m *FailAfterNCallsProvider) Name() string {
	return "fail-after-n"
}

// TestCoderFailureReturnsPartialResult verifies that when the Coder agent fails,
// the orchestrator returns the plan that was created along with the error.
// Validates: Property 17
func TestCoderFailureReturnsPartialResult(t *testing.T) {
	// Provider that succeeds for architect (2 calls) but fails for coder
	mockProvider := &FailAfterNCallsProvider{
		responses: []provider.LLMResponse{
			// Architect calls finish_plan
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Test goal",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Step 1",
									"action":      "test",
									"parameters":  map[string]interface{}{},
								},
							},
						},
					},
				},
			},
			// Architect final response
			{Text: "Plan created"},
		},
		failAfter: 2,
		failError: errors.New("coder LLM error"),
	}

	orch := NewOrchestrator(mockProvider, "/tmp/test")

	result, err := orch.Run(context.Background(), "Test goal")

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when coder fails")
	}

	// Error should mention coder
	if !contains(err.Error(), "coder") {
		t.Errorf("Expected error to mention 'coder', got: %v", err)
	}

	// Result should indicate failure
	if result.Success {
		t.Error("Expected success=false when coder fails")
	}

	// Plan should still be present (partial result)
	if result.Plan == nil {
		t.Error("Expected plan to be present in partial result")
	}

	if result.Plan.Goal != "Test goal" {
		t.Errorf("Expected plan goal 'Test goal', got '%s'", result.Plan.Goal)
	}

	// Phase should be Failed
	state := orch.State()
	if state.Phase != PhaseFailed {
		t.Errorf("Expected phase Failed, got %s", state.Phase)
	}
}

// TestNewOrchestratorInitialization verifies that NewOrchestrator creates
// a properly initialized orchestrator.
func TestNewOrchestratorInitialization(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	basePath := "/test/path"

	orch := NewOrchestrator(mockProvider, basePath)

	if orch == nil {
		t.Fatal("Expected non-nil orchestrator")
	}

	state := orch.State()
	if state.Phase != PhaseIdle {
		t.Errorf("Expected initial phase idle, got %s", state.Phase)
	}

	if state.CurrentAgent != "" {
		t.Errorf("Expected empty current agent, got %s", state.CurrentAgent)
	}

	if state.Plan != nil {
		t.Error("Expected nil plan initially")
	}

	if state.Error != "" {
		t.Errorf("Expected empty error, got %s", state.Error)
	}
}

// TestStateMethodIsThreadSafe verifies that the State() method returns
// a copy and doesn't expose internal state.
func TestStateMethodIsThreadSafe(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	orch := NewOrchestrator(mockProvider, "/tmp/test")

	// Get state copy
	state1 := orch.State()
	state1.Phase = PhaseFailed // Modify the copy

	// Get another copy
	state2 := orch.State()

	// Original should be unchanged
	if state2.Phase != PhaseIdle {
		t.Errorf("Expected phase idle (unchanged), got %s", state2.Phase)
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
