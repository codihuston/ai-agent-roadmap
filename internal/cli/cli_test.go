// Package cli provides tests for the CLI.
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"agentic-poc/internal/provider"
)

// mockProvider implements provider.LLMProvider for testing.
type mockProvider struct {
	responses []*provider.LLMResponse
	callIndex int
	calls     []provider.GenerateRequest
}

func newMockProvider(responses ...*provider.LLMResponse) *mockProvider {
	return &mockProvider{
		responses: responses,
		calls:     make([]provider.GenerateRequest, 0),
	}
}

func (m *mockProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
	m.calls = append(m.calls, req)
	if m.callIndex >= len(m.responses) {
		return &provider.LLMResponse{Text: "No more responses configured"}, nil
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func (m *mockProvider) Name() string {
	return "mock"
}

func TestNewCLI(t *testing.T) {
	mock := newMockProvider()
	cli := NewCLI(mock)

	if cli == nil {
		t.Fatal("NewCLI returned nil")
	}
	if cli.provider != mock {
		t.Error("CLI provider not set correctly")
	}
	if cli.output == nil {
		t.Error("CLI output not set")
	}
	if cli.input == nil {
		t.Error("CLI input not set")
	}
}

func TestNewCLIWithIO(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("test input")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)

	if cli == nil {
		t.Fatal("NewCLIWithIO returned nil")
	}
	if cli.provider != mock {
		t.Error("CLI provider not set correctly")
	}
}

func TestSetBasePath(t *testing.T) {
	mock := newMockProvider()
	cli := NewCLI(mock)

	cli.SetBasePath("/tmp/test")

	if cli.basePath != "/tmp/test" {
		t.Errorf("basePath = %q, want %q", cli.basePath, "/tmp/test")
	}
}

func TestIsExitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"exit", true},
		{"EXIT", true},
		{"Exit", true},
		{"  exit  ", true},
		{"quit", true},
		{"QUIT", true},
		{"Quit", true},
		{"  quit  ", true},
		{"hello", false},
		{"exiting", false},
		{"quitting", false},
		{"", false},
		{"exit now", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isExitCommand(tt.input)
			if result != tt.expected {
				t.Errorf("isExitCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSingleAgentMode_ExitCommand(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("exit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Single Agent Mode") {
		t.Error("Output should contain mode header")
	}
	if !strings.Contains(outputStr, "Goodbye!") {
		t.Error("Output should contain goodbye message")
	}
}

func TestSingleAgentMode_QuitCommand(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("quit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error: %v", err)
	}

	if !strings.Contains(output.String(), "Goodbye!") {
		t.Error("Output should contain goodbye message")
	}
}

func TestSingleAgentMode_SimpleInteraction(t *testing.T) {
	// Mock provider returns a simple response
	mock := newMockProvider(
		&provider.LLMResponse{Text: "Hello! How can I help you?"},
	)
	input := strings.NewReader("hello\nexit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Hello! How can I help you?") {
		t.Errorf("Output should contain assistant response, got: %s", outputStr)
	}
}

func TestSingleAgentMode_ToolCall(t *testing.T) {
	// Mock provider returns a tool call, then a final response
	mock := newMockProvider(
		&provider.LLMResponse{
			ToolCalls: []provider.ToolCall{
				{
					ID:   "call_1",
					Name: "calculator",
					Arguments: map[string]interface{}{
						"operation": "add",
						"a":         float64(2),
						"b":         float64(3),
					},
				},
			},
		},
		&provider.LLMResponse{Text: "The result is 5."},
	)
	input := strings.NewReader("what is 2 + 3?\nexit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Intermediate Steps") {
		t.Errorf("Output should show intermediate steps, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "calculator") {
		t.Errorf("Output should show tool name, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "The result is 5.") {
		t.Errorf("Output should contain final response, got: %s", outputStr)
	}
}

func TestSingleAgentMode_EmptyInput(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("\n\nexit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error: %v", err)
	}

	// Should not have called the provider for empty inputs
	if len(mock.calls) > 0 {
		t.Error("Should not call provider for empty inputs")
	}
}

func TestSingleAgentMode_EOF(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("") // EOF immediately
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunSingleAgentMode()

	if err != nil {
		t.Errorf("RunSingleAgentMode returned error on EOF: %v", err)
	}

	if !strings.Contains(output.String(), "Goodbye!") {
		t.Error("Output should contain goodbye message on EOF")
	}
}

func TestMultiAgentMode_ExitCommand(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("exit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunMultiAgentMode()

	if err != nil {
		t.Errorf("RunMultiAgentMode returned error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Multi-Agent Mode") {
		t.Error("Output should contain mode header")
	}
	if !strings.Contains(outputStr, "Goodbye!") {
		t.Error("Output should contain goodbye message")
	}
}

func TestMultiAgentMode_QuitCommand(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("quit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunMultiAgentMode()

	if err != nil {
		t.Errorf("RunMultiAgentMode returned error: %v", err)
	}

	if !strings.Contains(output.String(), "Goodbye!") {
		t.Error("Output should contain goodbye message")
	}
}

func TestMultiAgentMode_EOF(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("") // EOF immediately
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunMultiAgentMode()

	if err != nil {
		t.Errorf("RunMultiAgentMode returned error on EOF: %v", err)
	}

	if !strings.Contains(output.String(), "Goodbye!") {
		t.Error("Output should contain goodbye message on EOF")
	}
}

func TestMultiAgentMode_EmptyInput(t *testing.T) {
	mock := newMockProvider()
	input := strings.NewReader("\n\nexit\n")
	output := &bytes.Buffer{}

	cli := NewCLIWithIO(mock, input, output)
	err := cli.RunMultiAgentMode()

	if err != nil {
		t.Errorf("RunMultiAgentMode returned error: %v", err)
	}

	// Should not have called the provider for empty inputs
	if len(mock.calls) > 0 {
		t.Error("Should not call provider for empty inputs")
	}
}

func TestPrintToolCall(t *testing.T) {
	mock := newMockProvider()
	output := &bytes.Buffer{}
	cli := NewCLIWithIO(mock, strings.NewReader(""), output)

	tc := provider.ToolCall{
		ID:   "call_1",
		Name: "calculator",
		Arguments: map[string]interface{}{
			"operation": "add",
			"a":         float64(2),
			"b":         float64(3),
		},
	}

	cli.printToolCall(tc)

	outputStr := output.String()
	if !strings.Contains(outputStr, "[Tool Call]") {
		t.Error("Output should contain tool call marker")
	}
	if !strings.Contains(outputStr, "calculator") {
		t.Error("Output should contain tool name")
	}
}

func TestPrintAgentTransition(t *testing.T) {
	mock := newMockProvider()
	output := &bytes.Buffer{}
	cli := NewCLIWithIO(mock, strings.NewReader(""), output)

	cli.printAgentTransition("architect", "coder")

	outputStr := output.String()
	if !strings.Contains(outputStr, "Agent Transition") {
		t.Error("Output should contain transition marker")
	}
	if !strings.Contains(outputStr, "architect") {
		t.Error("Output should contain source agent")
	}
	if !strings.Contains(outputStr, "coder") {
		t.Error("Output should contain target agent")
	}
}
