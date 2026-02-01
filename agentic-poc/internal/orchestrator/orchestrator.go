// Package orchestrator coordinates multi-agent workflows.
package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"agentic-poc/internal/agent"
	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
)

// WorkflowPhase represents the current phase of the orchestrator workflow.
type WorkflowPhase string

const (
	// PhaseIdle indicates the orchestrator is not running a workflow.
	PhaseIdle WorkflowPhase = "idle"
	// PhasePlanning indicates the Architect agent is creating a plan.
	PhasePlanning WorkflowPhase = "planning"
	// PhaseExecuting indicates the Coder agent is executing the plan.
	PhaseExecuting WorkflowPhase = "executing"
	// PhaseComplete indicates the workflow completed successfully.
	PhaseComplete WorkflowPhase = "complete"
	// PhaseFailed indicates the workflow failed due to an error.
	PhaseFailed WorkflowPhase = "failed"
)

// WorkflowState represents the current state of the orchestrator workflow.
type WorkflowState struct {
	Phase        WorkflowPhase
	CurrentAgent string
	Plan         *agent.Plan
	Error        string
}

// OrchestratorResult represents the result of an orchestrator run.
type OrchestratorResult struct {
	Success      bool
	Plan         *agent.Plan
	ActionsTaken []string
	Summary      string
	Error        string
}

// Orchestrator coordinates the multi-agent workflow between Architect and Coder agents.
// It manages the workflow state and ensures proper sequencing of agent execution.
//
// Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
type Orchestrator struct {
	provider provider.LLMProvider
	basePath string
	state    WorkflowState
	mu       sync.RWMutex
}

// NewOrchestrator creates a new Orchestrator with the given LLM provider and base path.
// The basePath is used for file operations by the Coder agent.
func NewOrchestrator(llmProvider provider.LLMProvider, basePath string) *Orchestrator {
	return &Orchestrator{
		provider: llmProvider,
		basePath: basePath,
		state: WorkflowState{
			Phase: PhaseIdle,
		},
	}
}

// State returns a copy of the current workflow state.
// This method is thread-safe.
func (o *Orchestrator) State() WorkflowState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state
}

// setPhase updates the workflow phase and optionally the current agent.
func (o *Orchestrator) setPhase(phase WorkflowPhase, currentAgent string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state.Phase = phase
	o.state.CurrentAgent = currentAgent
}

// setPlan stores the plan in the workflow state.
func (o *Orchestrator) setPlan(plan *agent.Plan) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state.Plan = plan
}

// setError sets the error state and transitions to failed phase.
func (o *Orchestrator) setError(err string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state.Phase = PhaseFailed
	o.state.Error = err
}

// Run executes the multi-agent workflow with the given goal.
// It coordinates the Architect -> Coder flow:
// 1. Set phase to Planning, invoke Architect agent
// 2. Capture plan from FinishPlanTool
// 3. Set phase to Executing, invoke Coder agent with plan
// 4. Return result with actions taken
//
// Validates: Properties 15, 16, 17
func (o *Orchestrator) Run(ctx context.Context, goal string) (*OrchestratorResult, error) {
	// Reset state for new run
	o.mu.Lock()
	o.state = WorkflowState{Phase: PhaseIdle}
	o.mu.Unlock()

	// Phase 1: Planning with Architect agent
	o.setPhase(PhasePlanning, "architect")

	architectAgent, finishPlanTool := agent.NewArchitectAgent(o.provider)
	architectMemory := memory.NewConversationMemory()

	architectResult, err := architectAgent.Run(ctx, goal, architectMemory)
	if err != nil {
		errMsg := fmt.Sprintf("architect agent failed: %v", err)
		o.setError(errMsg)
		return &OrchestratorResult{
			Success: false,
			Error:   errMsg,
		}, fmt.Errorf("architect agent failed: %w", err)
	}

	// Check if a plan was captured
	if !finishPlanTool.HasCapturedPlan() {
		errMsg := "architect agent did not produce a plan"
		o.setError(errMsg)
		return &OrchestratorResult{
			Success: false,
			Error:   errMsg,
		}, fmt.Errorf(errMsg)
	}

	// Parse the captured plan
	planJSON := finishPlanTool.GetCapturedPlan()
	plan, err := agent.ParsePlan(planJSON)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse architect plan: %v", err)
		o.setError(errMsg)
		return &OrchestratorResult{
			Success: false,
			Error:   errMsg,
		}, fmt.Errorf("failed to parse architect plan: %w", err)
	}

	o.setPlan(plan)

	// Phase 2: Executing with Coder agent
	o.setPhase(PhaseExecuting, "coder")

	coderAgent := agent.NewCoderAgent(o.provider, o.basePath)
	coderMemory := memory.NewConversationMemory()

	// Prepare the plan as input for the Coder agent
	planInput, err := plan.ToJSON()
	if err != nil {
		errMsg := fmt.Sprintf("failed to serialize plan for coder: %v", err)
		o.setError(errMsg)
		return &OrchestratorResult{
			Success: false,
			Plan:    plan,
			Error:   errMsg,
		}, fmt.Errorf("failed to serialize plan for coder: %w", err)
	}

	coderPrompt := fmt.Sprintf("Execute the following plan:\n\n%s", planInput)
	coderResult, err := coderAgent.Run(ctx, coderPrompt, coderMemory)
	if err != nil {
		errMsg := fmt.Sprintf("coder agent failed: %v", err)
		o.setError(errMsg)
		return &OrchestratorResult{
			Success: false,
			Plan:    plan,
			Error:   errMsg,
		}, fmt.Errorf("coder agent failed: %w", err)
	}

	// Phase 3: Complete
	o.setPhase(PhaseComplete, "")

	// Collect actions taken from coder's tool calls
	actionsTaken := make([]string, 0, len(coderResult.ToolCallsMade))
	for _, tc := range coderResult.ToolCallsMade {
		actionsTaken = append(actionsTaken, fmt.Sprintf("%s: %v", tc.Name, tc.Arguments))
	}

	// Also include architect's tool calls
	architectActions := make([]string, 0, len(architectResult.ToolCallsMade))
	for _, tc := range architectResult.ToolCallsMade {
		architectActions = append(architectActions, fmt.Sprintf("%s: %v", tc.Name, tc.Arguments))
	}

	allActions := append(architectActions, actionsTaken...)

	return &OrchestratorResult{
		Success:      true,
		Plan:         plan,
		ActionsTaken: allActions,
		Summary:      coderResult.Response,
	}, nil
}
