# Implementation Tasks

## Task 1: Project Setup and Core Data Models
- [x] 1.1 Initialize Go module with `go mod init agentic-poc`
- [x] 1.2 Create project directory structure (cmd/, internal/, docs/)
- [x] 1.3 Implement core data models in `internal/provider/types.go`: Message, ToolCall, ToolResult, LLMResponse, ToolDefinition, GenerateRequest
- [x] 1.4 Implement Plan data models in `internal/agent/plan.go`: PlanStep, Plan with ToJSON() and ParsePlan()
- [x] 1.5 Write unit tests for Plan JSON serialization/deserialization
- [x] 1.6 (PBT) Write property test for Plan round-trip (Property 18)

## Task 2: LLM Provider Abstraction
- [x] 2.1 Define LLMProvider interface in `internal/provider/provider.go`
- [x] 2.2 Implement ClaudeProvider in `internal/provider/claude.go` with API key from env
- [x] 2.3 Implement Generate() method with proper request/response handling
- [x] 2.4 Implement error wrapping with context for all provider errors
- [x] 2.5 Write unit tests for ClaudeProvider (mock HTTP responses)

## Task 3: Tool Interface and Built-in Tools
- [x] 3.1 Define Tool interface in `internal/tool/tool.go`
- [x] 3.2 Implement CalculatorTool in `internal/tool/calculator.go` (add, subtract, multiply, divide)
- [x] 3.3 Write unit tests for CalculatorTool with known inputs
- [x] 3.4 (PBT) Write property test for Calculator correctness (Property 7)
- [x] 3.5 Implement FileReaderTool in `internal/tool/file_reader.go`
- [x] 3.6 Write unit tests for FileReaderTool (existing file, missing file, permission error)
- [x] 3.7 Implement FileWriterTool in `internal/tool/file_writer.go`
- [x] 3.8 Write unit tests for FileWriterTool
- [x] 3.9 Implement FinishPlanTool in `internal/tool/finish_plan.go`

## Task 4: Conversation Memory
- [x] 4.1 Implement ConversationMemory in `internal/memory/conversation.go`
- [x] 4.2 Implement AddMessage(), AddToolResult(), GetMessages(), Clear() methods
- [x] 4.3 Write unit tests for ConversationMemory operations
- [x] 4.4 (PBT) Write property test for message ordering preservation (Property 1)

## Task 5: Agent Core Loop
- [x] 5.1 Implement Agent struct and NewAgent() in `internal/agent/agent.go`
- [x] 5.2 Implement Run() method with Think -> Act -> Observe loop
- [x] 5.3 Implement tool call parsing from LLM response
- [x] 5.4 Implement tool execution dispatch to correct tool
- [x] 5.5 Implement tool result feedback to LLM
- [x] 5.6 Implement max iterations guard
- [x] 5.7 Write unit tests for agent loop with mocked LLM
- [x] 5.8 (PBT) Write property test for loop termination behavior (Properties 10, 11, 12)

## Task 6: Specialized Agents
- [x] 6.1 Implement NewArchitectAgent() in `internal/agent/architect.go` with system prompt
- [x] 6.2 Implement NewCoderAgent() in `internal/agent/coder.go` with system prompt
- [x] 6.3 Write unit tests for architect agent plan generation (mocked LLM)
- [x] 6.4 Write unit tests for coder agent execution (mocked LLM)

## Task 7: Orchestrator
- [x] 7.1 Implement Orchestrator struct and NewOrchestrator() in `internal/orchestrator/orchestrator.go`
- [x] 7.2 Implement WorkflowState and phase transitions
- [x] 7.3 Implement Run() method coordinating Architect -> Coder flow
- [x] 7.4 Implement error handling and workflow failure states
- [x] 7.5 Write unit tests for orchestrator workflow with mocked LLM provider (Validates: Requirements 7.1-7.6, Properties 15, 16, 17)
  - [x] 7.5.1 Test successful Architect -> Coder workflow with mocked LLM responses
  - [x] 7.5.2 Test workflow state transitions (idle -> planning -> executing -> complete)
  - [x] 7.5.3 Test architect failure sets phase to Failed and returns error
  - [x] 7.5.4 Test missing plan (architect doesn't call finish_plan) returns error
  - [x] 7.5.5 Test coder failure sets phase to Failed and returns partial result with plan

## Task 8: CLI Interface
- [x] 8.1 Implement CLI struct in `internal/cli/cli.go` with provider, input/output streams (Validates: Requirement 9.1)
- [x] 8.2 Implement RunSingleAgentMode() with interactive loop using Calculator and FileReader tools (Validates: Requirement 9.2)
- [x] 8.3 Implement RunMultiAgentMode() that invokes Orchestrator with user goal (Validates: Requirement 9.3)
- [x] 8.4 Implement intermediate step display showing tool calls and agent transitions (Validates: Requirement 9.5)
- [x] 8.5 Implement graceful exit on "exit" or "quit" commands (Validates: Requirement 9.4)
- [x] 8.6 Create main.go entry point in `cmd/agent/main.go` with mode selection flag (Validates: Requirements 9.2, 9.3)

## Task 9: Integration Testing
- [x] 9.1 Create integration test for single agent flow with mocked LLM in `test/integration/workflow_test.go` (Validates: Requirements 3, 4)
- [x] 9.2 Create integration test for multi-agent Architect->Coder flow with mocked LLM (Validates: Requirements 5, 6, 7)
- [x] 9.3 Create integration test for CLI mode switching between single and multi-agent modes (Validates: Requirements 9.2, 9.3)

## Task 10: Documentation
- [x] 10.1 Create docs/wiki/LEARNINGS.md with consolidated documentation structure
- [x] 10.2 Document challenges encountered during development in LEARNINGS.md
- [x] 10.3 Document trade-off decisions in LEARNINGS.md
- [x] 10.4 Document observations about LLM providers and patterns in LEARNINGS.md
- [x] 10.5 Write project README.md in `agentic-poc/README.md` with setup instructions, environment variables, and usage examples (Validates: Requirement 10)

## Task 11: MCP Integration (Future Phase)
- [ ]* 11.1 Define MCPClient interface in `internal/mcp/client.go`
- [ ]* 11.2 Implement StdioMCPClient in `internal/mcp/stdio.go`
- [ ]* 11.3 Implement MCPToolWrapper in `internal/mcp/wrapper.go`
- [ ]* 11.4 Implement MCPManager in `internal/mcp/manager.go`
- [ ]* 11.5 Implement config loading in `internal/mcp/config.go`
- [ ]* 11.6 Write unit tests for MCP components
- [ ]* 11.7 (PBT) Write property test for MCP tool interface compliance (Property 21)
- [ ]* 11.8 Create example mcp.json configuration file
