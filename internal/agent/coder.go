// Package agent implements the core agent loop and specialized agents.
package agent

import (
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// CoderSystemPrompt is the system prompt for the Coder agent.
// It instructs the agent to execute plans step by step.
const CoderSystemPrompt = `You are a Coder agent responsible for executing implementation plans step by step.

Your role is to:
1. Read and understand the plan provided to you
2. Execute each step in order using the available tools
3. Use read_file to examine existing files when needed
4. Use write_file to create or modify files as specified in the plan
5. Report on the completion of each step

Available tools:
- read_file: Read the contents of a file at a specified path
- write_file: Write content to a file at a specified path (creates directories as needed)

Guidelines for executing plans:
- Follow the plan steps in order
- Read files before modifying them if you need to understand their current state
- Write complete, working code when creating files
- Handle errors gracefully and report any issues
- Provide a summary of actions taken when complete

When you have completed all steps in the plan, provide a summary of what was accomplished.`

// NewCoderAgent creates a new Agent configured as a Coder.
// The Coder agent is responsible for executing plans by reading and writing files.
// The basePath parameter specifies the root directory for file operations.
//
// Validates: Requirements 6.1, 6.2, 6.3, 6.4
func NewCoderAgent(llmProvider provider.LLMProvider, basePath string) *Agent {
	fileReader := tool.NewFileReaderTool(basePath)
	fileWriter := tool.NewFileWriterTool(basePath)

	agent := NewAgent(AgentConfig{
		Provider:      llmProvider,
		Tools:         []tool.Tool{fileReader, fileWriter},
		SystemPrompt:  CoderSystemPrompt,
		MaxIterations: DefaultMaxIterations,
	})

	return agent
}
