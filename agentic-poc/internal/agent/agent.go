// Package agent implements the core agent loop and specialized agents.
package agent

import (
	"context"
	"errors"
	"fmt"

	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// Default values for agent configuration.
const (
	DefaultMaxIterations = 10
)

// ErrMaxIterationsExceeded is returned when the agent loop reaches the maximum number of iterations.
var ErrMaxIterationsExceeded = errors.New("max iterations exceeded")

// AgentConfig holds configuration for creating a new Agent.
type AgentConfig struct {
	Provider      provider.LLMProvider
	Tools         []tool.Tool
	SystemPrompt  string
	MaxIterations int
}

// AgentResult represents the result of an agent run.
type AgentResult struct {
	Response      string
	ToolCallsMade []provider.ToolCall
	Iterations    int
}

// Agent implements the Think -> Act -> Observe loop for interacting with an LLM.
type Agent struct {
	provider      provider.LLMProvider
	tools         map[string]tool.Tool
	systemPrompt  string
	maxIterations int
}

// NewAgent creates a new Agent with the given configuration.
// If MaxIterations is not set (0), it defaults to DefaultMaxIterations.
func NewAgent(cfg AgentConfig) *Agent {
	maxIter := cfg.MaxIterations
	if maxIter <= 0 {
		maxIter = DefaultMaxIterations
	}

	// Build tool map for quick lookup by name
	toolMap := make(map[string]tool.Tool)
	for _, t := range cfg.Tools {
		toolMap[t.Name()] = t
	}

	return &Agent{
		provider:      cfg.Provider,
		tools:         toolMap,
		systemPrompt:  cfg.SystemPrompt,
		maxIterations: maxIter,
	}
}

// RegisterTool adds a tool to the agent's tool registry.
func (a *Agent) RegisterTool(t tool.Tool) {
	a.tools[t.Name()] = t
}

// GetTools returns a slice of all registered tools.
func (a *Agent) GetTools() []tool.Tool {
	tools := make([]tool.Tool, 0, len(a.tools))
	for _, t := range a.tools {
		tools = append(tools, t)
	}
	return tools
}

// Run executes the agent loop with the given input and conversation memory.
// It implements the Think -> Act -> Observe loop:
// 1. Add user input to memory
// 2. Call LLM with history and tools
// 3. If no tool calls, return response
// 4. Execute tool calls, add results to memory
// 5. Repeat until max iterations or final response
func (a *Agent) Run(ctx context.Context, input string, mem *memory.ConversationMemory) (*AgentResult, error) {
	// Add user input to memory
	mem.AddMessage("user", input)

	// Track all tool calls made during this run
	allToolCalls := make([]provider.ToolCall, 0)

	// Build tool definitions for LLM
	toolDefs := a.buildToolDefinitions()

	for iteration := 1; iteration <= a.maxIterations; iteration++ {
		// Think: Call LLM with current context
		req := provider.GenerateRequest{
			Messages:     mem.GetMessages(),
			Tools:        toolDefs,
			SystemPrompt: a.systemPrompt,
		}

		resp, err := a.provider.Generate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		// Check if this is a final response (no tool calls)
		if !resp.HasToolCalls() {
			// Add assistant response to memory
			mem.AddMessage("assistant", resp.Text)

			return &AgentResult{
				Response:      resp.Text,
				ToolCallsMade: allToolCalls,
				Iterations:    iteration,
			}, nil
		}

		// Store the assistant's response with tool calls BEFORE executing tools
		// This is required by Claude API - tool_result must follow tool_use in the same conversation
		mem.AddAssistantMessageWithToolCalls(resp.Text, resp.ToolCalls)

		// Act: Execute tool calls
		for _, tc := range resp.ToolCalls {
			allToolCalls = append(allToolCalls, tc)

			result := a.executeTool(ctx, tc)

			// Observe: Add tool result to memory
			mem.AddToolResult(tc.ID, tc.Name, result)
		}
	}

	// Max iterations reached without final response
	return nil, fmt.Errorf("%w: reached %d iterations without final response", ErrMaxIterationsExceeded, a.maxIterations)
}

// buildToolDefinitions converts registered tools to ToolDefinitions for LLM requests.
func (a *Agent) buildToolDefinitions() []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, 0, len(a.tools))
	for _, t := range a.tools {
		defs = append(defs, tool.ToDefinition(t))
	}
	return defs
}

// executeTool dispatches a tool call to the correct tool and returns the result as a string.
// If the tool is not found or execution fails, it returns an error message instead of panicking.
func (a *Agent) executeTool(ctx context.Context, tc provider.ToolCall) string {
	t, exists := a.tools[tc.Name]
	if !exists {
		return fmt.Sprintf("error: unknown tool '%s'", tc.Name)
	}

	result, err := t.Execute(ctx, tc.Arguments)
	if err != nil {
		return fmt.Sprintf("error: tool execution failed: %v", err)
	}

	if !result.Success {
		return fmt.Sprintf("error: %s", result.Error)
	}

	return result.Output
}
