// Package integration provides end-to-end integration tests for the agentic system.
package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-poc/internal/agent"
	"agentic-poc/internal/cli"
	"agentic-poc/internal/memory"
	"agentic-poc/internal/orchestrator"
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

func newMockProvider(responses ...provider.LLMResponse) *mockLLMProvider {
	return &mockLLMProvider{
		responses: responses,
		requests:  make([]provider.GenerateRequest, 0),
	}
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

// =============================================================================
// Task 9.1: Single Agent Flow Integration Tests
// Validates: Requirements 3 (Single Agent Tool Use), 4 (Agent Loop Implementation)
// =============================================================================

// TestSingleAgentFlow_CalculatorToolUse tests the end-to-end single agent flow
// with the calculator tool using a mocked LLM.
func TestSingleAgentFlow_CalculatorToolUse(t *testing.T) {
	// Mock LLM that requests calculator tool, then provides final response
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "add",
						"a":         float64(15),
						"b":         float64(27),
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "The sum of 15 and 27 is 42.",
		},
	)

	// Create agent with real calculator tool
	calcTool := tool.NewCalculatorTool()
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{calcTool},
		SystemPrompt:  "You are a helpful assistant with a calculator.",
		MaxIterations: 10,
	})

	mem := memory.NewConversationMemory()
	result, err := agentInstance.Run(context.Background(), "What is 15 + 27?", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify final response
	if result.Response != "The sum of 15 and 27 is 42." {
		t.Errorf("response = %q, want %q", result.Response, "The sum of 15 and 27 is 42.")
	}

	// Verify tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("tool calls made = %d, want 1", len(result.ToolCallsMade))
	}

	// Verify iterations
	if result.Iterations != 2 {
		t.Errorf("iterations = %d, want 2", result.Iterations)
	}

	// Verify LLM received tool result
	if len(mockProvider.requests) < 2 {
		t.Fatal("expected at least 2 LLM requests")
	}

	// Check that the second request contains the tool result
	secondReq := mockProvider.requests[1]
	foundToolResult := false
	for _, msg := range secondReq.Messages {
		if msg.Role == "tool" && msg.ToolName == "calculator" {
			foundToolResult = true
			if !strings.Contains(msg.Content, "42") {
				t.Errorf("tool result should contain '42', got: %s", msg.Content)
			}
		}
	}
	if !foundToolResult {
		t.Error("expected tool result in second LLM request")
	}
}

// TestSingleAgentFlow_FileReaderToolUse tests the end-to-end single agent flow
// with the file reader tool using a mocked LLM.
func TestSingleAgentFlow_FileReaderToolUse(t *testing.T) {
	// Create a temporary file to read
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, this is test content!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Mock LLM that requests file read, then provides final response
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "read_file",
					Arguments: map[string]interface{}{
						"path": "test.txt",
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "The file contains: Hello, this is test content!",
		},
	)

	// Create agent with real file reader tool
	fileReader := tool.NewFileReaderTool(tmpDir)
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{fileReader},
		SystemPrompt:  "You are a helpful assistant that can read files.",
		MaxIterations: 10,
	})

	mem := memory.NewConversationMemory()
	result, err := agentInstance.Run(context.Background(), "Read the test.txt file", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify final response
	if result.Response != "The file contains: Hello, this is test content!" {
		t.Errorf("response = %q, want %q", result.Response, "The file contains: Hello, this is test content!")
	}

	// Verify tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("tool calls made = %d, want 1", len(result.ToolCallsMade))
	}

	// Verify LLM received file content
	if len(mockProvider.requests) < 2 {
		t.Fatal("expected at least 2 LLM requests")
	}

	secondReq := mockProvider.requests[1]
	foundToolResult := false
	for _, msg := range secondReq.Messages {
		if msg.Role == "tool" && msg.ToolName == "read_file" {
			foundToolResult = true
			if !strings.Contains(msg.Content, testContent) {
				t.Errorf("tool result should contain file content, got: %s", msg.Content)
			}
		}
	}
	if !foundToolResult {
		t.Error("expected tool result in second LLM request")
	}
}

// TestSingleAgentFlow_MultipleToolCalls tests the agent handling multiple
// sequential tool calls in a single conversation.
func TestSingleAgentFlow_MultipleToolCalls(t *testing.T) {
	// Mock LLM that makes two calculator calls, then provides final response
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "multiply",
						"a":         float64(6),
						"b":         float64(7),
					},
				},
			},
		},
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_2",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "add",
						"a":         float64(42),
						"b":         float64(8),
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "6 * 7 = 42, and 42 + 8 = 50.",
		},
	)

	calcTool := tool.NewCalculatorTool()
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{calcTool},
		SystemPrompt:  "You are a helpful calculator assistant.",
		MaxIterations: 10,
	})

	mem := memory.NewConversationMemory()
	result, err := agentInstance.Run(context.Background(), "Calculate 6*7, then add 8", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify two tool calls were made
	if len(result.ToolCallsMade) != 2 {
		t.Errorf("tool calls made = %d, want 2", len(result.ToolCallsMade))
	}

	// Verify iterations (3: first tool call, second tool call, final response)
	if result.Iterations != 3 {
		t.Errorf("iterations = %d, want 3", result.Iterations)
	}

	// Verify final response
	if result.Response != "6 * 7 = 42, and 42 + 8 = 50." {
		t.Errorf("response = %q, want %q", result.Response, "6 * 7 = 42, and 42 + 8 = 50.")
	}
}

// TestSingleAgentFlow_ToolErrorHandling tests that the agent continues
// gracefully when a tool returns an error.
func TestSingleAgentFlow_ToolErrorHandling(t *testing.T) {
	// Mock LLM that requests division by zero, then handles the error
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "divide",
						"a":         float64(10),
						"b":         float64(0),
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "I cannot divide by zero. Please provide a non-zero divisor.",
		},
	)

	calcTool := tool.NewCalculatorTool()
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{calcTool},
		SystemPrompt:  "You are a helpful calculator assistant.",
		MaxIterations: 10,
	})

	mem := memory.NewConversationMemory()
	result, err := agentInstance.Run(context.Background(), "Divide 10 by 0", mem)

	// Agent should NOT return an error - it should handle tool errors gracefully
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the agent got a response
	if result.Response == "" {
		t.Error("expected non-empty response")
	}

	// Verify the error was passed to the LLM
	if len(mockProvider.requests) < 2 {
		t.Fatal("expected at least 2 LLM requests")
	}

	secondReq := mockProvider.requests[1]
	foundErrorResult := false
	for _, msg := range secondReq.Messages {
		if msg.Role == "tool" && msg.ToolName == "calculator" {
			foundErrorResult = true
			if !strings.Contains(strings.ToLower(msg.Content), "error") &&
				!strings.Contains(strings.ToLower(msg.Content), "zero") {
				t.Errorf("tool result should contain error about division by zero, got: %s", msg.Content)
			}
		}
	}
	if !foundErrorResult {
		t.Error("expected tool error result in second LLM request")
	}
}

// TestSingleAgentFlow_ConversationMemoryPersistence tests that conversation
// memory is properly maintained across the agent loop.
func TestSingleAgentFlow_ConversationMemoryPersistence(t *testing.T) {
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "add",
						"a":         float64(5),
						"b":         float64(3),
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "The result is 8.",
		},
	)

	calcTool := tool.NewCalculatorTool()
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider:      mockProvider,
		Tools:         []tool.Tool{calcTool},
		SystemPrompt:  "You are a calculator.",
		MaxIterations: 10,
	})

	mem := memory.NewConversationMemory()
	_, err := agentInstance.Run(context.Background(), "Add 5 and 3", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify conversation memory contains all messages
	messages := mem.GetMessages()

	// Should have: user message, tool result, assistant response
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages in memory, got %d", len(messages))
	}

	// Verify message order and content
	if messages[0].Role != "user" {
		t.Errorf("first message role = %q, want 'user'", messages[0].Role)
	}
	if messages[1].Role != "tool" {
		t.Errorf("second message role = %q, want 'tool'", messages[1].Role)
	}
	if messages[2].Role != "assistant" {
		t.Errorf("third message role = %q, want 'assistant'", messages[2].Role)
	}
}

// =============================================================================
// Task 9.2: Multi-Agent Architect->Coder Flow Integration Tests
// Validates: Requirements 5 (Architect Agent), 6 (Coder Agent), 7 (Multi-Agent Orchestration)
// =============================================================================

// TestMultiAgentFlow_ArchitectCoderWorkflow tests the end-to-end multi-agent
// workflow where Architect creates a plan and Coder executes it.
func TestMultiAgentFlow_ArchitectCoderWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Track which agent is being called
	callSequence := []string{}

	// Create a mock provider that handles both architect and coder
	mockProvider := &sequentialMockProvider{
		responseSequences: [][]provider.LLMResponse{
			// Architect responses
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "plan_1",
							Name: "finish_plan",
							Arguments: map[string]interface{}{
								"goal": "Create a hello world file",
								"steps": []interface{}{
									map[string]interface{}{
										"description": "Create hello.txt with greeting",
										"action":      "write_file",
										"parameters": map[string]interface{}{
											"path":    "hello.txt",
											"content": "Hello, World!",
										},
									},
								},
							},
						},
					},
				},
				{
					Text: "Plan created successfully.",
				},
			},
			// Coder responses
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "write_1",
							Name: "write_file",
							Arguments: map[string]interface{}{
								"path":    "hello.txt",
								"content": "Hello, World!",
							},
						},
					},
				},
				{
					Text: "Successfully created hello.txt with the greeting.",
				},
			},
		},
		currentSequence: 0,
		callSequence:    &callSequence,
	}

	orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)
	result, err := orch.Run(context.Background(), "Create a hello world file")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify success
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify plan was captured
	if result.Plan == nil {
		t.Fatal("expected plan to be captured")
	}

	if result.Plan.Goal != "Create a hello world file" {
		t.Errorf("plan goal = %q, want %q", result.Plan.Goal, "Create a hello world file")
	}

	if len(result.Plan.Steps) != 1 {
		t.Errorf("plan steps = %d, want 1", len(result.Plan.Steps))
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "hello.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("file content = %q, want %q", string(content), "Hello, World!")
	}

	// Verify actions were taken
	if len(result.ActionsTaken) == 0 {
		t.Error("expected actions to be recorded")
	}

	// Verify workflow completed
	state := orch.State()
	if state.Phase != orchestrator.PhaseComplete {
		t.Errorf("workflow phase = %q, want %q", state.Phase, orchestrator.PhaseComplete)
	}
}

// sequentialMockProvider handles multiple agent calls in sequence.
type sequentialMockProvider struct {
	responseSequences [][]provider.LLMResponse
	currentSequence   int
	currentIndex      int
	callSequence      *[]string
}

func (m *sequentialMockProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	// Track which sequence we're in based on system prompt
	if strings.Contains(req.SystemPrompt, "Architect") {
		*m.callSequence = append(*m.callSequence, "architect")
		if m.currentSequence != 0 {
			m.currentSequence = 0
			m.currentIndex = 0
		}
	} else if strings.Contains(req.SystemPrompt, "Coder") {
		*m.callSequence = append(*m.callSequence, "coder")
		if m.currentSequence != 1 {
			m.currentSequence = 1
			m.currentIndex = 0
		}
	}

	if m.currentSequence >= len(m.responseSequences) {
		return &provider.LLMResponse{Text: "default response"}, nil
	}

	sequence := m.responseSequences[m.currentSequence]
	if m.currentIndex >= len(sequence) {
		return &provider.LLMResponse{Text: "default response"}, nil
	}

	resp := sequence[m.currentIndex]
	m.currentIndex++
	return &resp, nil
}

func (m *sequentialMockProvider) Name() string {
	return "sequential-mock"
}

// TestMultiAgentFlow_WorkflowStateTransitions tests that the orchestrator
// properly transitions through workflow phases.
func TestMultiAgentFlow_WorkflowStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()

	// Simple mock that completes quickly
	mockProvider := &sequentialMockProvider{
		responseSequences: [][]provider.LLMResponse{
			// Architect
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "plan_1",
							Name: "finish_plan",
							Arguments: map[string]interface{}{
								"goal": "Test goal",
								"steps": []interface{}{
									map[string]interface{}{
										"description": "Test step",
										"action":      "write_file",
									},
								},
							},
						},
					},
				},
				{Text: "Done"},
			},
			// Coder
			{
				{Text: "Completed all steps."},
			},
		},
		callSequence: &[]string{},
	}

	orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)

	// Initial state should be idle
	state := orch.State()
	if state.Phase != orchestrator.PhaseIdle {
		t.Errorf("initial phase = %q, want %q", state.Phase, orchestrator.PhaseIdle)
	}

	// Run the workflow
	result, err := orch.Run(context.Background(), "Test goal")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Final state should be complete
	state = orch.State()
	if state.Phase != orchestrator.PhaseComplete {
		t.Errorf("final phase = %q, want %q", state.Phase, orchestrator.PhaseComplete)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// TestMultiAgentFlow_ArchitectFailure tests that the orchestrator properly
// handles architect agent failures.
func TestMultiAgentFlow_ArchitectFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock that doesn't call finish_plan (architect failure)
	mockProvider := &sequentialMockProvider{
		responseSequences: [][]provider.LLMResponse{
			// Architect - doesn't call finish_plan
			{
				{Text: "I don't know how to create a plan."},
			},
		},
		callSequence: &[]string{},
	}

	orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)
	result, err := orch.Run(context.Background(), "Create something")

	// Should return an error
	if err == nil {
		t.Fatal("expected error when architect doesn't produce plan")
	}

	// Result should indicate failure
	if result.Success {
		t.Error("expected failure result")
	}

	// State should be failed
	state := orch.State()
	if state.Phase != orchestrator.PhaseFailed {
		t.Errorf("phase = %q, want %q", state.Phase, orchestrator.PhaseFailed)
	}
}

// TestMultiAgentFlow_PlanPassedToCoder tests that the plan from Architect
// is correctly passed to the Coder agent.
func TestMultiAgentFlow_PlanPassedToCoder(t *testing.T) {
	tmpDir := t.TempDir()

	// Track requests to verify plan is passed
	var coderRequests []provider.GenerateRequest

	mockProvider := &planTrackingMockProvider{
		architectResponses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "plan_1",
						Name: "finish_plan",
						Arguments: map[string]interface{}{
							"goal": "Create config file",
							"steps": []interface{}{
								map[string]interface{}{
									"description": "Write config.json",
									"action":      "write_file",
									"parameters": map[string]interface{}{
										"path":    "config.json",
										"content": `{"key": "value"}`,
									},
								},
							},
						},
					},
				},
			},
			{Text: "Plan ready"},
		},
		coderResponses: []provider.LLMResponse{
			{Text: "Config file created."},
		},
		coderRequests: &coderRequests,
	}

	orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)
	_, err := orch.Run(context.Background(), "Create config file")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify coder received the plan
	if len(coderRequests) == 0 {
		t.Fatal("expected coder to receive requests")
	}

	// Check that the plan was included in the coder's input
	firstCoderReq := coderRequests[0]
	foundPlan := false
	for _, msg := range firstCoderReq.Messages {
		if msg.Role == "user" && strings.Contains(msg.Content, "Create config file") {
			foundPlan = true
			break
		}
	}

	if !foundPlan {
		t.Error("expected plan to be passed to coder agent")
	}
}

// planTrackingMockProvider tracks requests to different agents.
type planTrackingMockProvider struct {
	architectResponses []provider.LLMResponse
	coderResponses     []provider.LLMResponse
	architectIndex     int
	coderIndex         int
	coderRequests      *[]provider.GenerateRequest
}

func (m *planTrackingMockProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	if strings.Contains(req.SystemPrompt, "Architect") {
		if m.architectIndex >= len(m.architectResponses) {
			return &provider.LLMResponse{Text: "default"}, nil
		}
		resp := m.architectResponses[m.architectIndex]
		m.architectIndex++
		return &resp, nil
	}

	if strings.Contains(req.SystemPrompt, "Coder") {
		*m.coderRequests = append(*m.coderRequests, req)
		if m.coderIndex >= len(m.coderResponses) {
			return &provider.LLMResponse{Text: "default"}, nil
		}
		resp := m.coderResponses[m.coderIndex]
		m.coderIndex++
		return &resp, nil
	}

	return &provider.LLMResponse{Text: "unknown agent"}, nil
}

func (m *planTrackingMockProvider) Name() string {
	return "plan-tracking-mock"
}

// TestMultiAgentFlow_CoderExecutesMultipleSteps tests that the Coder agent
// can execute a plan with multiple steps.
func TestMultiAgentFlow_CoderExecutesMultipleSteps(t *testing.T) {
	tmpDir := t.TempDir()

	mockProvider := &sequentialMockProvider{
		responseSequences: [][]provider.LLMResponse{
			// Architect
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "plan_1",
							Name: "finish_plan",
							Arguments: map[string]interface{}{
								"goal": "Create multiple files",
								"steps": []interface{}{
									map[string]interface{}{
										"description": "Create file1.txt",
										"action":      "write_file",
									},
									map[string]interface{}{
										"description": "Create file2.txt",
										"action":      "write_file",
									},
								},
							},
						},
					},
				},
				{Text: "Plan with 2 steps created"},
			},
			// Coder
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "write_1",
							Name: "write_file",
							Arguments: map[string]interface{}{
								"path":    "file1.txt",
								"content": "Content 1",
							},
						},
					},
				},
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "write_2",
							Name: "write_file",
							Arguments: map[string]interface{}{
								"path":    "file2.txt",
								"content": "Content 2",
							},
						},
					},
				},
				{Text: "Both files created successfully."},
			},
		},
		callSequence: &[]string{},
	}

	orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)
	result, err := orch.Run(context.Background(), "Create multiple files")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify both files were created
	file1Path := filepath.Join(tmpDir, "file1.txt")
	file2Path := filepath.Join(tmpDir, "file2.txt")

	content1, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("failed to read file1.txt: %v", err)
	} else if string(content1) != "Content 1" {
		t.Errorf("file1.txt content = %q, want %q", string(content1), "Content 1")
	}

	content2, err := os.ReadFile(file2Path)
	if err != nil {
		t.Errorf("failed to read file2.txt: %v", err)
	} else if string(content2) != "Content 2" {
		t.Errorf("file2.txt content = %q, want %q", string(content2), "Content 2")
	}

	// Verify actions were recorded
	if len(result.ActionsTaken) < 2 {
		t.Errorf("expected at least 2 actions, got %d", len(result.ActionsTaken))
	}
}

// =============================================================================
// Task 9.3: CLI Mode Switching Integration Tests
// Validates: Requirements 9.2 (Single-agent mode), 9.3 (Multi-agent mode)
// =============================================================================

// TestCLI_SingleAgentModeIntegration tests the CLI in single-agent mode
// with a complete interaction flow.
func TestCLI_SingleAgentModeIntegration(t *testing.T) {
	// Mock provider that handles a calculator request
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "calc_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "multiply",
						"a":         float64(7),
						"b":         float64(8),
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "7 times 8 equals 56.",
		},
	)

	// Simulate user input: ask a question, then exit
	input := strings.NewReader("What is 7 * 8?\nexit\n")
	output := &bytes.Buffer{}

	cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
	err := cliInstance.RunSingleAgentMode()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputStr := output.String()

	// Verify mode header is displayed
	if !strings.Contains(outputStr, "Single Agent Mode") {
		t.Error("output should contain mode header")
	}

	// Verify available tools are listed
	if !strings.Contains(outputStr, "calculator") {
		t.Error("output should list calculator tool")
	}

	// Verify intermediate steps are shown
	if !strings.Contains(outputStr, "Intermediate Steps") {
		t.Error("output should show intermediate steps")
	}

	// Verify tool call is displayed
	if !strings.Contains(outputStr, "[Tool Call]") {
		t.Error("output should show tool call")
	}

	// Verify final response is displayed
	if !strings.Contains(outputStr, "56") {
		t.Error("output should contain the calculation result")
	}

	// Verify graceful exit
	if !strings.Contains(outputStr, "Goodbye!") {
		t.Error("output should contain goodbye message")
	}
}

// TestCLI_MultiAgentModeIntegration tests the CLI in multi-agent mode
// with a complete Architect->Coder workflow.
func TestCLI_MultiAgentModeIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock provider for multi-agent workflow
	mockProvider := &sequentialMockProvider{
		responseSequences: [][]provider.LLMResponse{
			// Architect
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "plan_1",
							Name: "finish_plan",
							Arguments: map[string]interface{}{
								"goal": "Create greeting file",
								"steps": []interface{}{
									map[string]interface{}{
										"description": "Write greeting",
										"action":      "write_file",
									},
								},
							},
						},
					},
				},
				{Text: "Plan created"},
			},
			// Coder
			{
				{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "write_1",
							Name: "write_file",
							Arguments: map[string]interface{}{
								"path":    "greeting.txt",
								"content": "Hello!",
							},
						},
					},
				},
				{Text: "Greeting file created successfully."},
			},
		},
		callSequence: &[]string{},
	}

	// Simulate user input: provide goal, then exit
	input := strings.NewReader("Create a greeting file\nexit\n")
	output := &bytes.Buffer{}

	cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
	cliInstance.SetBasePath(tmpDir)
	err := cliInstance.RunMultiAgentMode()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputStr := output.String()

	// Verify mode header is displayed
	if !strings.Contains(outputStr, "Multi-Agent Mode") {
		t.Error("output should contain mode header")
	}

	// Verify agent transition is shown
	if !strings.Contains(outputStr, "Agent Transition") {
		t.Error("output should show agent transition")
	}

	// Verify plan is displayed
	if !strings.Contains(outputStr, "Plan") {
		t.Error("output should display the plan")
	}

	// Verify workflow phase is shown
	if !strings.Contains(outputStr, "Workflow Phase") {
		t.Error("output should show workflow phase")
	}

	// Verify success indication
	if !strings.Contains(outputStr, "Success: true") {
		t.Error("output should indicate success")
	}

	// Verify graceful exit
	if !strings.Contains(outputStr, "Goodbye!") {
		t.Error("output should contain goodbye message")
	}
}

// TestCLI_ModeSwitchingCapability tests that the CLI can be configured
// to run in either mode based on user choice.
func TestCLI_ModeSwitchingCapability(t *testing.T) {
	mockProvider := newMockProvider(
		provider.LLMResponse{Text: "Hello!"},
	)

	tests := []struct {
		name       string
		runMode    func(*cli.CLI) error
		wantHeader string
	}{
		{
			name: "single agent mode",
			runMode: func(c *cli.CLI) error {
				return c.RunSingleAgentMode()
			},
			wantHeader: "Single Agent Mode",
		},
		{
			name: "multi agent mode",
			runMode: func(c *cli.CLI) error {
				return c.RunMultiAgentMode()
			},
			wantHeader: "Multi-Agent Mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader("exit\n")
			output := &bytes.Buffer{}

			cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
			err := tt.runMode(cliInstance)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output.String(), tt.wantHeader) {
				t.Errorf("output should contain %q header", tt.wantHeader)
			}
		})
	}
}

// TestCLI_GracefulExitCommands tests that both "exit" and "quit" commands
// work in both modes.
func TestCLI_GracefulExitCommands(t *testing.T) {
	mockProvider := newMockProvider()

	exitCommands := []string{"exit", "quit", "EXIT", "QUIT", "Exit", "Quit"}
	modes := []struct {
		name    string
		runMode func(*cli.CLI) error
	}{
		{"single", func(c *cli.CLI) error { return c.RunSingleAgentMode() }},
		{"multi", func(c *cli.CLI) error { return c.RunMultiAgentMode() }},
	}

	for _, mode := range modes {
		for _, cmd := range exitCommands {
			t.Run(mode.name+"_"+cmd, func(t *testing.T) {
				input := strings.NewReader(cmd + "\n")
				output := &bytes.Buffer{}

				cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
				err := mode.runMode(cliInstance)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !strings.Contains(output.String(), "Goodbye!") {
					t.Error("output should contain goodbye message")
				}
			})
		}
	}
}

// TestCLI_SingleAgentModeWithFileReader tests the CLI in single-agent mode
// with file reading capability.
func TestCLI_SingleAgentModeWithFileReader(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "data.txt")
	if err := os.WriteFile(testFile, []byte("Important data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Mock provider that reads a file
	mockProvider := newMockProvider(
		provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "read_1",
					Name: "read_file",
					Arguments: map[string]interface{}{
						"path": "data.txt",
					},
				},
			},
		},
		provider.LLMResponse{
			Text: "The file contains: Important data",
		},
	)

	input := strings.NewReader("Read data.txt\nexit\n")
	output := &bytes.Buffer{}

	cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
	cliInstance.SetBasePath(tmpDir)
	err := cliInstance.RunSingleAgentMode()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputStr := output.String()

	// Verify file reading tool was used
	if !strings.Contains(outputStr, "read_file") {
		t.Error("output should show read_file tool call")
	}

	// Verify response contains file content
	if !strings.Contains(outputStr, "Important data") {
		t.Error("output should contain file content")
	}
}

// TestCLI_EmptyInputHandling tests that empty inputs are handled gracefully
// in both modes without calling the LLM.
func TestCLI_EmptyInputHandling(t *testing.T) {
	mockProvider := newMockProvider()

	modes := []struct {
		name    string
		runMode func(*cli.CLI) error
	}{
		{"single", func(c *cli.CLI) error { return c.RunSingleAgentMode() }},
		{"multi", func(c *cli.CLI) error { return c.RunMultiAgentMode() }},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			// Multiple empty lines followed by exit
			input := strings.NewReader("\n\n\nexit\n")
			output := &bytes.Buffer{}

			cliInstance := cli.NewCLIWithIO(mockProvider, input, output)
			err := mode.runMode(cliInstance)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Provider should not have been called for empty inputs
			if mockProvider.callCount > 0 {
				t.Errorf("provider was called %d times for empty inputs", mockProvider.callCount)
			}
		})
	}
}
