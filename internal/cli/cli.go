// Package cli provides the command-line interface for the agentic system.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"agentic-poc/internal/agent"
	"agentic-poc/internal/mcp"
	"agentic-poc/internal/memory"
	"agentic-poc/internal/orchestrator"
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// CLI provides the command-line interface for interacting with the agentic system.
// It supports both single-agent mode (for testing tool use) and multi-agent mode
// (for the Architect/Coder workflow).
//
// Validates: Requirement 9.1
type CLI struct {
	provider   provider.LLMProvider
	output     io.Writer
	input      *bufio.Scanner
	basePath   string
	mcpManager *mcp.MCPManager
	mcpOnly    bool // If true, only use MCP tools (no built-in tools)
}

// NewCLI creates a new CLI instance with the given LLM provider.
// By default, it uses os.Stdout for output and os.Stdin for input.
func NewCLI(llmProvider provider.LLMProvider) *CLI {
	return &CLI{
		provider: llmProvider,
		output:   os.Stdout,
		input:    bufio.NewScanner(os.Stdin),
		basePath: ".",
	}
}

// NewCLIWithIO creates a new CLI instance with custom input/output streams.
// This is useful for testing.
func NewCLIWithIO(llmProvider provider.LLMProvider, input io.Reader, output io.Writer) *CLI {
	return &CLI{
		provider: llmProvider,
		output:   output,
		input:    bufio.NewScanner(input),
		basePath: ".",
	}
}

// SetBasePath sets the base path for file operations.
func (c *CLI) SetBasePath(path string) {
	c.basePath = path
}

// SetMCPOnly sets whether to use only MCP tools (no built-in tools).
func (c *CLI) SetMCPOnly(mcpOnly bool) {
	c.mcpOnly = mcpOnly
}

// LoadMCPConfig loads MCP servers from the given config file path.
// Returns an error if mcpOnly is true and config cannot be loaded.
func (c *CLI) LoadMCPConfig(ctx context.Context, configPath string) error {
	cfg, err := mcp.LoadMCPConfig(configPath)
	if err != nil {
		if c.mcpOnly {
			return fmt.Errorf("mcp-only mode requires valid config: %w", err)
		}
		// Config file not found is not an error in normal mode
		return nil
	}

	c.mcpManager = mcp.NewMCPManager()
	if err := c.mcpManager.LoadFromConfig(ctx, cfg); err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	if c.mcpOnly && c.mcpManager.ToolCount() == 0 {
		return fmt.Errorf("mcp-only mode but no MCP tools loaded")
	}

	return nil
}

// Shutdown cleans up resources, including MCP connections.
func (c *CLI) Shutdown() {
	if c.mcpManager != nil {
		c.mcpManager.Shutdown()
	}
}

// printf is a helper to write formatted output.
func (c *CLI) printf(format string, args ...interface{}) {
	fmt.Fprintf(c.output, format, args...)
}

// println is a helper to write a line of output.
func (c *CLI) println(args ...interface{}) {
	fmt.Fprintln(c.output, args...)
}

// printToolCall displays information about a tool call.
// Validates: Requirement 9.5
func (c *CLI) printToolCall(tc provider.ToolCall) {
	c.printf("  [Tool Call] %s\n", tc.Name)
	for key, value := range tc.Arguments {
		c.printf("    %s: %v\n", key, value)
	}
}

// printAgentTransition displays information about an agent transition.
// Validates: Requirement 9.5
func (c *CLI) printAgentTransition(from, to string) {
	c.printf("\n>>> Agent Transition: %s -> %s\n\n", from, to)
}

// isExitCommand checks if the input is an exit command.
// Validates: Requirement 9.4
func isExitCommand(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return lower == "exit" || lower == "quit"
}

// RunSingleAgentMode runs the CLI in single-agent mode with an interactive loop.
// The agent has access to Calculator and FileReader tools, plus any MCP tools.
// If mcpOnly is true, only MCP tools are used.
//
// Validates: Requirement 9.2
func (c *CLI) RunSingleAgentMode() error {
	c.println("=== Single Agent Mode ===")

	var tools []tool.Tool

	if c.mcpOnly {
		// MCP-only mode: use only tools from MCP servers
		c.println("Mode: MCP-only (tools loaded from MCP servers)")
		if c.mcpManager == nil {
			return fmt.Errorf("mcp-only mode but no MCP manager configured")
		}
		tools = c.mcpManager.GetTools()
		if len(tools) == 0 {
			return fmt.Errorf("mcp-only mode but no MCP tools available")
		}
		c.println("Available MCP tools:")
		for _, t := range tools {
			c.printf("  - %s: %s\n", t.Name(), t.Description())
		}
	} else {
		// Normal mode: use built-in tools
		c.println("Mode: Built-in tools")
		tools = []tool.Tool{
			tool.NewCalculatorTool(),
			tool.NewFileReaderTool(c.basePath),
		}
		c.println("Available tools: calculator, read_file")

		// Also add MCP tools if available
		if c.mcpManager != nil {
			mcpTools := c.mcpManager.GetTools()
			if len(mcpTools) > 0 {
				c.printf("Additional MCP tools: %d\n", len(mcpTools))
				for _, t := range mcpTools {
					c.printf("  - %s\n", t.Name())
				}
				tools = append(tools, mcpTools...)
			}
		}
	}

	c.println("Type 'exit' or 'quit' to exit.")
	c.println()

	// Create the agent with a clear system prompt
	agentInstance := agent.NewAgent(agent.AgentConfig{
		Provider: c.provider,
		Tools:    tools,
		SystemPrompt: `You are a helpful assistant with access to two tools:
1. calculator - Use this for ANY math operations (add, subtract, multiply, divide). Always use the calculator tool for arithmetic.
2. read_file - Use this to read file contents when asked about files.

When the user asks a math question, use the calculator tool. Do not try to calculate in your head.
When the user asks to read a file, use the read_file tool.
Keep responses concise and helpful.`,
		MaxIterations: 10,
	})

	// Interactive loop - fresh memory for each prompt
	for {
		c.printf("You: ")

		if !c.input.Scan() {
			// EOF or error
			if err := c.input.Err(); err != nil {
				return fmt.Errorf("input error: %w", err)
			}
			c.println("\nGoodbye!")
			return nil
		}

		input := strings.TrimSpace(c.input.Text())
		if input == "" {
			continue
		}

		// Check for exit command
		if isExitCommand(input) {
			c.println("Goodbye!")
			return nil
		}

		// Create fresh memory for each prompt to avoid token accumulation
		mem := memory.NewConversationMemory()

		// Run the agent
		ctx := context.Background()
		result, err := agentInstance.Run(ctx, input, mem)
		if err != nil {
			c.printf("Error: %v\n\n", err)
			continue
		}

		// Display tool calls made (intermediate steps)
		if len(result.ToolCallsMade) > 0 {
			c.println("\n--- Intermediate Steps ---")
			for _, tc := range result.ToolCallsMade {
				c.printToolCall(tc)
			}
			c.printf("  Iterations: %d\n", result.Iterations)
			c.println("---------------------------")
		}

		// Display the response
		c.printf("\nAssistant: %s\n\n", result.Response)
	}
}

// RunMultiAgentMode runs the CLI in multi-agent mode with the Architect/Coder workflow.
// The user provides a goal, and the orchestrator coordinates the agents.
//
// Validates: Requirement 9.3
func (c *CLI) RunMultiAgentMode() error {
	c.println("=== Multi-Agent Mode (Architect/Coder) ===")
	c.println("Enter a goal for the system to accomplish.")
	c.println("The Architect will create a plan, and the Coder will execute it.")
	c.println("Type 'exit' or 'quit' to exit.")
	c.println()

	// Create the orchestrator
	orch := orchestrator.NewOrchestrator(c.provider, c.basePath)

	// Interactive loop
	for {
		c.printf("Goal: ")

		if !c.input.Scan() {
			// EOF or error
			if err := c.input.Err(); err != nil {
				return fmt.Errorf("input error: %w", err)
			}
			c.println("\nGoodbye!")
			return nil
		}

		input := strings.TrimSpace(c.input.Text())
		if input == "" {
			continue
		}

		// Check for exit command
		if isExitCommand(input) {
			c.println("Goodbye!")
			return nil
		}

		// Display agent transition
		c.printAgentTransition("user", "architect")

		// Run the orchestrator
		ctx := context.Background()

		// Create a callback to display state changes
		c.println("Starting workflow...")

		result, err := orch.Run(ctx, input)

		// Display final state
		state := orch.State()
		c.printf("\nWorkflow Phase: %s\n", state.Phase)

		if err != nil {
			c.printf("Error: %v\n\n", err)
			continue
		}

		// Display the plan
		if result.Plan != nil {
			c.println("\n--- Plan ---")
			c.printf("Goal: %s\n", result.Plan.Goal)
			c.println("Steps:")
			for i, step := range result.Plan.Steps {
				c.printf("  %d. %s (action: %s)\n", i+1, step.Description, step.Action)
			}
			c.println("------------")
		}

		// Display agent transition
		if state.Phase == orchestrator.PhaseComplete {
			c.printAgentTransition("architect", "coder")
		}

		// Display actions taken
		if len(result.ActionsTaken) > 0 {
			c.println("\n--- Actions Taken ---")
			for _, action := range result.ActionsTaken {
				c.printf("  • %s\n", action)
			}
			c.println("---------------------")
		}

		// Display summary
		c.printf("\nSummary: %s\n", result.Summary)
		c.printf("Success: %v\n\n", result.Success)
	}
}
