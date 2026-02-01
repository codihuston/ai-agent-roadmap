package agent

import (
	"context"
	"errors"
	"testing"

	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// mockLLMProvider is a test double for LLMProvider that returns predefined responses.
type mockLLMProvider struct {
	responses []provider.LLMResponse
	errors    []error
	callCount int
	requests  []provider.GenerateRequest
}

func (m *mockLLMProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	m.requests = append(m.requests, req)
	idx := m.callCount
	m.callCount++

	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}

	if idx < len(m.responses) {
		return &m.responses[idx], nil
	}

	// Default: return empty response (no tool calls)
	return &provider.LLMResponse{Text: "default response"}, nil
}

func (m *mockLLMProvider) Name() string {
	return "mock"
}

// mockTool is a test double for Tool interface.
type mockTool struct {
	name        string
	description string
	params      map[string]interface{}
	result      *provider.ToolResult
	err         error
	callCount   int
	lastArgs    map[string]interface{}
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return t.description }
func (t *mockTool) Parameters() map[string]interface{} {
	if t.params == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	return t.params
}

func (t *mockTool) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	t.callCount++
	t.lastArgs = args
	if t.err != nil {
		return nil, t.err
	}
	if t.result != nil {
		return t.result, nil
	}
	return &provider.ToolResult{Success: true, Output: "mock result"}, nil
}

func TestNewAgent(t *testing.T) {
	tests := []struct {
		name              string
		cfg               AgentConfig
		wantMaxIterations int
		wantToolCount     int
	}{
		{
			name: "default max iterations when not set",
			cfg: AgentConfig{
				Provider: &mockLLMProvider{},
			},
			wantMaxIterations: DefaultMaxIterations,
			wantToolCount:     0,
		},
		{
			name: "custom max iterations",
			cfg: AgentConfig{
				Provider:      &mockLLMProvider{},
				MaxIterations: 5,
			},
			wantMaxIterations: 5,
			wantToolCount:     0,
		},
		{
			name: "with tools",
			cfg: AgentConfig{
				Provider: &mockLLMProvider{},
				Tools: []tool.Tool{
					&mockTool{name: "tool1"},
					&mockTool{name: "tool2"},
				},
			},
			wantMaxIterations: DefaultMaxIterations,
			wantToolCount:     2,
		},
		{
			name: "with system prompt",
			cfg: AgentConfig{
				Provider:     &mockLLMProvider{},
				SystemPrompt: "You are a helpful assistant",
			},
			wantMaxIterations: DefaultMaxIterations,
			wantToolCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(tt.cfg)

			if agent.maxIterations != tt.wantMaxIterations {
				t.Errorf("maxIterations = %d, want %d", agent.maxIterations, tt.wantMaxIterations)
			}

			if len(agent.tools) != tt.wantToolCount {
				t.Errorf("tool count = %d, want %d", len(agent.tools), tt.wantToolCount)
			}
		})
	}
}

func TestAgent_Run_FinalResponseWithoutToolCalls(t *testing.T) {
	// Property 11: For any LLMResponse without tool calls, the Agent SHALL immediately
	// return that response as the final result without making additional LLM requests.
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Hello, I'm here to help!"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Hello", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response != "Hello, I'm here to help!" {
		t.Errorf("response = %q, want %q", result.Response, "Hello, I'm here to help!")
	}

	if result.Iterations != 1 {
		t.Errorf("iterations = %d, want 1", result.Iterations)
	}

	if len(result.ToolCallsMade) != 0 {
		t.Errorf("tool calls made = %d, want 0", len(result.ToolCallsMade))
	}

	if mockProvider.callCount != 1 {
		t.Errorf("LLM call count = %d, want 1", mockProvider.callCount)
	}
}

func TestAgent_Run_WithToolCalls(t *testing.T) {
	// Property 10: For any LLMResponse containing tool calls, the Agent loop SHALL
	// execute those tools and make another LLM request.
	mockTool := &mockTool{
		name:   "calculator",
		result: &provider.ToolResult{Success: true, Output: "42"},
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "calculator", Arguments: map[string]interface{}{"a": 1, "b": 2}},
				},
			},
			{Text: "The result is 42"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool},
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "What is 1 + 2?", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response != "The result is 42" {
		t.Errorf("response = %q, want %q", result.Response, "The result is 42")
	}

	if result.Iterations != 2 {
		t.Errorf("iterations = %d, want 2", result.Iterations)
	}

	if len(result.ToolCallsMade) != 1 {
		t.Errorf("tool calls made = %d, want 1", len(result.ToolCallsMade))
	}

	if mockTool.callCount != 1 {
		t.Errorf("tool call count = %d, want 1", mockTool.callCount)
	}

	if mockProvider.callCount != 2 {
		t.Errorf("LLM call count = %d, want 2", mockProvider.callCount)
	}
}

func TestAgent_Run_MultipleToolCalls(t *testing.T) {
	// Test multiple tool calls in a single response
	mockTool1 := &mockTool{
		name:   "tool1",
		result: &provider.ToolResult{Success: true, Output: "result1"},
	}
	mockTool2 := &mockTool{
		name:   "tool2",
		result: &provider.ToolResult{Success: true, Output: "result2"},
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "tool1", Arguments: map[string]interface{}{}},
					{ID: "call_2", Name: "tool2", Arguments: map[string]interface{}{}},
				},
			},
			{Text: "Both tools executed"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool1, mockTool2},
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Use both tools", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ToolCallsMade) != 2 {
		t.Errorf("tool calls made = %d, want 2", len(result.ToolCallsMade))
	}

	if mockTool1.callCount != 1 {
		t.Errorf("tool1 call count = %d, want 1", mockTool1.callCount)
	}

	if mockTool2.callCount != 1 {
		t.Errorf("tool2 call count = %d, want 1", mockTool2.callCount)
	}
}

func TestAgent_Run_MaxIterationsExceeded(t *testing.T) {
	// Property 12: For any agent run that reaches the configured maxIterations without
	// receiving a final response, the Agent SHALL return an error indicating max iterations exceeded.
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{ToolCalls: []provider.ToolCall{{ID: "1", Name: "tool", Arguments: map[string]interface{}{}}}},
			{ToolCalls: []provider.ToolCall{{ID: "2", Name: "tool", Arguments: map[string]interface{}{}}}},
			{ToolCalls: []provider.ToolCall{{ID: "3", Name: "tool", Arguments: map[string]interface{}{}}}},
		},
	}

	mockTool := &mockTool{
		name:   "tool",
		result: &provider.ToolResult{Success: true, Output: "ok"},
	}

	agent := NewAgent(AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{mockTool},
		MaxIterations: 3,
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Keep calling tools", mem)

	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrMaxIterationsExceeded) {
		t.Errorf("error = %v, want ErrMaxIterationsExceeded", err)
	}
}

func TestAgent_Run_UnknownTool(t *testing.T) {
	// Property 9: For any tool execution that returns an error, the Agent SHALL continue
	// operation by sending the error message to the LLM rather than panicking.
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "unknown_tool", Arguments: map[string]interface{}{}},
				},
			},
			{Text: "I couldn't find that tool"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		// No tools registered
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Use unknown tool", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Agent should continue and get final response
	if result.Response != "I couldn't find that tool" {
		t.Errorf("response = %q, want %q", result.Response, "I couldn't find that tool")
	}

	// Verify error message was sent to LLM
	if len(mockProvider.requests) < 2 {
		t.Fatal("expected at least 2 LLM requests")
	}

	messages := mockProvider.requests[1].Messages
	foundErrorMsg := false
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolName == "unknown_tool" {
			foundErrorMsg = true
			break
		}
	}
	if !foundErrorMsg {
		t.Error("expected tool error message in second request")
	}
}

func TestAgent_Run_ToolExecutionError(t *testing.T) {
	// Tool returns an error - agent should continue
	mockTool := &mockTool{
		name: "failing_tool",
		err:  errors.New("tool execution failed"),
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "failing_tool", Arguments: map[string]interface{}{}},
				},
			},
			{Text: "Tool failed, but I can help another way"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool},
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Use failing tool", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response != "Tool failed, but I can help another way" {
		t.Errorf("response = %q, want %q", result.Response, "Tool failed, but I can help another way")
	}
}

func TestAgent_Run_ToolResultFailure(t *testing.T) {
	// Tool returns a result with Success=false
	mockTool := &mockTool{
		name:   "tool",
		result: &provider.ToolResult{Success: false, Error: "invalid input"},
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "tool", Arguments: map[string]interface{}{}},
				},
			},
			{Text: "I see there was an error"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool},
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Use tool", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response != "I see there was an error" {
		t.Errorf("response = %q, want %q", result.Response, "I see there was an error")
	}
}

func TestAgent_Run_LLMError(t *testing.T) {
	// LLM returns an error - agent should propagate it
	mockProvider := &mockLLMProvider{
		errors: []error{errors.New("API rate limit exceeded")},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
	})

	mem := memory.NewConversationMemory()
	result, err := agent.Run(context.Background(), "Hello", mem)

	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "LLM generation failed: API rate limit exceeded" {
		t.Errorf("error = %q, want wrapped error", err.Error())
	}
}

func TestAgent_Run_SystemPromptIncluded(t *testing.T) {
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Response"},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider:     mockProvider,
		SystemPrompt: "You are a helpful assistant",
	})

	mem := memory.NewConversationMemory()
	_, err := agent.Run(context.Background(), "Hello", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockProvider.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mockProvider.requests))
	}

	if mockProvider.requests[0].SystemPrompt != "You are a helpful assistant" {
		t.Errorf("system prompt = %q, want %q", mockProvider.requests[0].SystemPrompt, "You are a helpful assistant")
	}
}

func TestAgent_Run_ToolDefinitionsIncluded(t *testing.T) {
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Response"},
		},
	}

	mockTool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		params: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{"type": "string"},
			},
		},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool},
	})

	mem := memory.NewConversationMemory()
	_, err := agent.Run(context.Background(), "Hello", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockProvider.requests[0].Tools) != 1 {
		t.Fatalf("expected 1 tool definition, got %d", len(mockProvider.requests[0].Tools))
	}

	toolDef := mockProvider.requests[0].Tools[0]
	if toolDef.Name != "test_tool" {
		t.Errorf("tool name = %q, want %q", toolDef.Name, "test_tool")
	}
	if toolDef.Description != "A test tool" {
		t.Errorf("tool description = %q, want %q", toolDef.Description, "A test tool")
	}
}

func TestAgent_Run_ConversationMemoryUpdated(t *testing.T) {
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: "tool", Arguments: map[string]interface{}{}},
				},
			},
			{Text: "Final response"},
		},
	}

	mockTool := &mockTool{
		name:   "tool",
		result: &provider.ToolResult{Success: true, Output: "tool output"},
	}

	agent := NewAgent(AgentConfig{
		Provider: mockProvider,
		Tools:    []tool.Tool{mockTool},
	})

	mem := memory.NewConversationMemory()
	_, err := agent.Run(context.Background(), "Hello", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := mem.GetMessages()

	// Should have: user message, tool result, assistant response
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("message[0] = %+v, want user message", messages[0])
	}

	if messages[1].Role != "tool" || messages[1].Content != "tool output" {
		t.Errorf("message[1] = %+v, want tool result", messages[1])
	}

	if messages[2].Role != "assistant" || messages[2].Content != "Final response" {
		t.Errorf("message[2] = %+v, want assistant response", messages[2])
	}
}

func TestAgent_RegisterTool(t *testing.T) {
	agent := NewAgent(AgentConfig{
		Provider: &mockLLMProvider{},
	})

	if len(agent.tools) != 0 {
		t.Errorf("initial tool count = %d, want 0", len(agent.tools))
	}

	mockTool := &mockTool{name: "new_tool"}
	agent.RegisterTool(mockTool)

	if len(agent.tools) != 1 {
		t.Errorf("tool count after register = %d, want 1", len(agent.tools))
	}

	if _, exists := agent.tools["new_tool"]; !exists {
		t.Error("registered tool not found")
	}
}

func TestAgent_GetTools(t *testing.T) {
	mockTool1 := &mockTool{name: "tool1"}
	mockTool2 := &mockTool{name: "tool2"}

	agent := NewAgent(AgentConfig{
		Provider: &mockLLMProvider{},
		Tools:    []tool.Tool{mockTool1, mockTool2},
	})

	tools := agent.GetTools()

	if len(tools) != 2 {
		t.Errorf("tool count = %d, want 2", len(tools))
	}
}
