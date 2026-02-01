package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// **Feature: agentic-system-poc, Properties 10, 11, 12: Agent Loop Termination Behavior**
// **Validates: Requirements 4.2, 4.3, 4.4**

// TestProperty_AgentLoopTermination tests the three core properties of agent loop termination:
// - Property 10: Agent continues loop while tool calls exist
// - Property 11: Agent terminates immediately on final response (no tool calls)
// - Property 12: Agent returns error when max iterations exceeded
func TestProperty_AgentLoopTermination(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property 10: For any LLMResponse containing tool calls, the Agent loop SHALL execute
	// those tools and make another LLM request, continuing until a response without tool
	// calls is received or max iterations is reached.
	// **Validates: Requirements 4.2**
	properties.Property("Agent continues loop while tool calls exist", prop.ForAll(
		func(numToolCallRounds int) bool {
			// Create a mock provider that returns tool calls for numToolCallRounds iterations
			// then returns a final response
			responses := make([]provider.LLMResponse, numToolCallRounds+1)
			for i := 0; i < numToolCallRounds; i++ {
				responses[i] = provider.LLMResponse{
					ToolCalls: []provider.ToolCall{
						{ID: "call_" + string(rune('0'+i)), Name: "test_tool", Arguments: map[string]interface{}{}},
					},
				}
			}
			responses[numToolCallRounds] = provider.LLMResponse{Text: "Final response"}

			mockProvider := &mockLLMProvider{responses: responses}
			mockTool := &mockTool{
				name:   "test_tool",
				result: &provider.ToolResult{Success: true, Output: "ok"},
			}

			agent := NewAgent(AgentConfig{
				Provider:      mockProvider,
				Tools:         []tool.Tool{mockTool},
				MaxIterations: numToolCallRounds + 5, // Ensure we don't hit max iterations
			})

			mem := memory.NewConversationMemory()
			result, err := agent.Run(context.Background(), "test input", mem)

			if err != nil {
				return false
			}

			// Verify the agent made the correct number of LLM calls
			// (numToolCallRounds + 1 for the final response)
			if mockProvider.callCount != numToolCallRounds+1 {
				return false
			}

			// Verify the tool was called the correct number of times
			if mockTool.callCount != numToolCallRounds {
				return false
			}

			// Verify we got the final response
			if result.Response != "Final response" {
				return false
			}

			// Verify iterations count
			if result.Iterations != numToolCallRounds+1 {
				return false
			}

			return true
		},
		gen.IntRange(0, 8), // Test 0-8 rounds of tool calls
	))

	// Property 11: For any LLMResponse without tool calls, the Agent SHALL immediately
	// return that response as the final result without making additional LLM requests.
	// **Validates: Requirements 4.3**
	properties.Property("Agent terminates immediately on final response", prop.ForAll(
		func(responseText string) bool {
			if responseText == "" {
				responseText = "default response" // Ensure non-empty
			}

			mockProvider := &mockLLMProvider{
				responses: []provider.LLMResponse{
					{Text: responseText}, // No tool calls
				},
			}

			agent := NewAgent(AgentConfig{
				Provider:      mockProvider,
				MaxIterations: 10,
			})

			mem := memory.NewConversationMemory()
			result, err := agent.Run(context.Background(), "test input", mem)

			if err != nil {
				return false
			}

			// Verify only one LLM call was made
			if mockProvider.callCount != 1 {
				return false
			}

			// Verify the response matches
			if result.Response != responseText {
				return false
			}

			// Verify only one iteration
			if result.Iterations != 1 {
				return false
			}

			// Verify no tool calls were made
			if len(result.ToolCallsMade) != 0 {
				return false
			}

			return true
		},
		gen.AnyString(),
	))

	// Property 12: For any agent run that reaches the configured maxIterations without
	// receiving a final response, the Agent SHALL return an error indicating max iterations exceeded.
	// **Validates: Requirements 4.4**
	properties.Property("Agent returns error when max iterations exceeded", prop.ForAll(
		func(maxIterations int) bool {
			if maxIterations < 1 {
				maxIterations = 1 // Ensure at least 1 iteration
			}

			// Create responses that always have tool calls (never a final response)
			responses := make([]provider.LLMResponse, maxIterations+5)
			for i := range responses {
				responses[i] = provider.LLMResponse{
					ToolCalls: []provider.ToolCall{
						{ID: "call", Name: "test_tool", Arguments: map[string]interface{}{}},
					},
				}
			}

			mockProvider := &mockLLMProvider{responses: responses}
			mockTool := &mockTool{
				name:   "test_tool",
				result: &provider.ToolResult{Success: true, Output: "ok"},
			}

			agent := NewAgent(AgentConfig{
				Provider:      mockProvider,
				Tools:         []tool.Tool{mockTool},
				MaxIterations: maxIterations,
			})

			mem := memory.NewConversationMemory()
			result, err := agent.Run(context.Background(), "test input", mem)

			// Should return nil result
			if result != nil {
				return false
			}

			// Should return an error
			if err == nil {
				return false
			}

			// Error should be ErrMaxIterationsExceeded
			if !errors.Is(err, ErrMaxIterationsExceeded) {
				return false
			}

			// Verify the agent made exactly maxIterations LLM calls
			if mockProvider.callCount != maxIterations {
				return false
			}

			return true
		},
		gen.IntRange(1, 15),
	))

	properties.TestingRun(t)
}

// TestProperty_ToolCallsAreExecuted verifies that all tool calls in a response are executed.
// This is a supporting property for Property 10.
func TestProperty_ToolCallsAreExecuted(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("All tool calls in a response are executed", prop.ForAll(
		func(numToolCalls int) bool {
			if numToolCalls < 1 {
				numToolCalls = 1
			}

			// Create tool calls
			toolCalls := make([]provider.ToolCall, numToolCalls)
			for i := 0; i < numToolCalls; i++ {
				toolCalls[i] = provider.ToolCall{
					ID:        "call_" + string(rune('a'+i)),
					Name:      "test_tool",
					Arguments: map[string]interface{}{},
				}
			}

			mockProvider := &mockLLMProvider{
				responses: []provider.LLMResponse{
					{ToolCalls: toolCalls},
					{Text: "Final response"},
				},
			}

			mockTool := &mockTool{
				name:   "test_tool",
				result: &provider.ToolResult{Success: true, Output: "ok"},
			}

			agent := NewAgent(AgentConfig{
				Provider:      mockProvider,
				Tools:         []tool.Tool{mockTool},
				MaxIterations: 10,
			})

			mem := memory.NewConversationMemory()
			result, err := agent.Run(context.Background(), "test input", mem)

			if err != nil {
				return false
			}

			// Verify all tool calls were executed
			if mockTool.callCount != numToolCalls {
				return false
			}

			// Verify all tool calls are recorded in result
			if len(result.ToolCallsMade) != numToolCalls {
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// TestProperty_ToolResultsFedBackToLLM verifies that tool results are included in subsequent LLM requests.
// This supports Property 6: Tool Results Are Fed Back to LLM.
func TestProperty_ToolResultsFedBackToLLM(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("Tool results are included in next LLM request", prop.ForAll(
		func(toolOutput string) bool {
			if toolOutput == "" {
				toolOutput = "default output"
			}

			mockProvider := &mockLLMProvider{
				responses: []provider.LLMResponse{
					{
						ToolCalls: []provider.ToolCall{
							{ID: "call_1", Name: "test_tool", Arguments: map[string]interface{}{}},
						},
					},
					{Text: "Final response"},
				},
			}

			mockTool := &mockTool{
				name:   "test_tool",
				result: &provider.ToolResult{Success: true, Output: toolOutput},
			}

			agent := NewAgent(AgentConfig{
				Provider:      mockProvider,
				Tools:         []tool.Tool{mockTool},
				MaxIterations: 10,
			})

			mem := memory.NewConversationMemory()
			_, err := agent.Run(context.Background(), "test input", mem)

			if err != nil {
				return false
			}

			// Verify there were 2 LLM requests
			if len(mockProvider.requests) != 2 {
				return false
			}

			// Verify the second request contains the tool result
			secondRequest := mockProvider.requests[1]
			foundToolResult := false
			for _, msg := range secondRequest.Messages {
				if msg.Role == "tool" && msg.Content == toolOutput {
					foundToolResult = true
					break
				}
			}

			return foundToolResult
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}
