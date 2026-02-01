// Package agent implements the core agent loop and specialized agents.
package agent

import (
	"agentic-poc/internal/provider"
	"agentic-poc/internal/tool"
)

// ArchitectSystemPrompt is the system prompt for the Architect agent.
// It instructs the agent to create detailed implementation plans.
const ArchitectSystemPrompt = `You are an Architect agent responsible for breaking down high-level goals into detailed implementation plans.

Your role is to:
1. Analyze the user's goal and understand what needs to be accomplished
2. Break down the goal into clear, actionable steps
3. For each step, specify what action should be taken (e.g., write_file, read_file)
4. Include any necessary parameters for each action

When you have completed your plan, you MUST call the finish_plan tool with:
- goal: The original goal being addressed
- steps: An array of steps, where each step has:
  - description: What this step accomplishes
  - action: The action to perform (e.g., "write_file", "read_file")
  - parameters: Any parameters needed for the action (optional)

Guidelines for creating plans:
- Be specific and detailed in step descriptions
- Order steps logically (dependencies first)
- Use appropriate actions for each step
- Keep steps atomic and focused on a single task
- Consider error handling and edge cases

Always call finish_plan when your plan is complete. Do not provide the plan as text - use the tool.`

// NewArchitectAgent creates a new Agent configured as an Architect.
// The Architect agent is responsible for breaking down high-level goals into detailed plans.
// It returns both the Agent and the FinishPlanTool so the caller can retrieve the captured plan.
//
// Validates: Requirements 5.1, 5.2, 5.3
func NewArchitectAgent(llmProvider provider.LLMProvider) (*Agent, *tool.FinishPlanTool) {
	finishPlanTool := tool.NewFinishPlanTool()

	agent := NewAgent(AgentConfig{
		Provider:      llmProvider,
		Tools:         []tool.Tool{finishPlanTool},
		SystemPrompt:  ArchitectSystemPrompt,
		MaxIterations: DefaultMaxIterations,
	})

	return agent, finishPlanTool
}
