# Implementation Tasks

## Task 1: Project Setup and Core Data Models
- [ ] 1.1 Initialize Go module with `go mod init agentic-poc`
- [ ] 1.2 Create project directory structure (cmd/, internal/, docs/)
- [ ] 1.3 Implement core data models in `internal/provider/types.go`: Message, ToolCall, ToolResult, LLMResponse, ToolDefinition, GenerateRequest
- [ ] 1.4 Implement Plan data models in `internal/agent/plan.go`: PlanStep, Plan with ToJSON() and ParsePlan()
- [ ] 1.5 Write unit tests for Plan JSON serialization/deserialization
- [ ] 1.6 (PBT) Write property test for Plan round-trip (Property 18)

## Task 2: LLM Provider Abstraction
- [ ] 2.1 Define LLMProvider interface in `internal/provider/provider.go`
- [ ] 2.2 Implement ClaudeProvider in `internal/provider/claude.go` with API key from env
- [ ] 2.3 Implement Generate() method with proper request/response handling
- [ ] 2.4 Implement error wrapping with context for all provider errors
- [ ] 2.5 Write unit tests for ClaudeProvider (mock HTTP responses)

## Task 3: Tool Interface and Built-in Tools
- [ ] 3.1 Define Tool interface in `internal/tool/tool.go`
- [ ] 3.2 Implement CalculatorTool in `internal/tool/calculator.go` (add, subtract, multiply, divide)
- [ ] 3.3 Write unit tests for CalculatorTool with known inputs
- [ ] 3.4 (PBT) Write property test for Calculator correctness (Property 7)
- [ ] 3.5 Implement FileReaderTool in `internal/tool/file_reader.go`
- [ ] 3.6 Write unit tests for FileReaderTool (existing file, missing file, permission error)
- [ ] 3.7 Implement FileWriterTool in `internal/tool/file_writer.go`
- [ ] 3.8 Write unit tests for FileWriterTool
- [ ] 3.9 Implement FinishPlanTool in `internal/tool/finish_plan.go`

## Task 4: Conversation Memory
- [ ] 4.1 Implement ConversationMemory in `internal/memory/conversation.go`
- [ ] 4.2 Implement AddMessage(), AddToolResult(), GetMessages(), Clear() methods
- [ ] 4.3 Write unit tests for ConversationMemory operations
- [ ] 4.4 (PBT) Write property test for message ordering preservation (Property 1)

## Task 5: Agent Core Loop
- [ ] 5.1 Implement Agent struct and NewAgent() in `internal/agent/agent.go`
- [ ] 5.2 Implement Run() method with Think -> Act -> Observe loop
- [ ] 5.3 Implement tool call parsing from LLM response
- [ ] 5.4 Implement tool execution dispatch to correct tool
- [ ] 5.5 Implement tool result feedback to LLM
- [ ] 5.6 Implement max iterations guard
- [ ] 5.7 Write unit tests for agent loop with mocked LLM
- [ ] 5.8 (PBT) Write property test for loop termination behavior (Properties 10, 11, 12)

## Task 6: Specialized Agents
- [ ] 6.1 Implement NewArchitectAgent() in `internal/agent/architect.go` with system prompt
- [ ] 6.2 Implement NewCoderAgent() in `internal/agent/coder.go` with system prompt
- [ ] 6.3 Write unit tests for architect agent plan generation (mocked LLM)
- [ ] 6.4 Write unit tests for coder agent execution (mocked LLM)

## Task 7: Orchestrator
- [ ] 7.1 Implement Orchestrator struct and NewOrchestrator() in `internal/orchestrator/orchestrator.go`
- [ ] 7.2 Implement WorkflowState and phase transitions
- [ ] 7.3 Implement Run() method coordinating Architect -> Coder flow
- [ ] 7.4 Implement error handling and workflow failure states
- [ ] 7.5 Write unit tests for orchestrator workflow (mocked agents)

## Task 8: CLI Interface
- [ ] 8.1 Implement CLI struct in `internal/cli/cli.go`
- [ ] 8.2 Implement RunSingleAgentMode() with interactive loop
- [ ] 8.3 Implement RunMultiAgentMode() with goal input
- [ ] 8.4 Implement intermediate step display (tool calls, agent transitions)
- [ ] 8.5 Implement graceful exit on "exit" or "quit"
- [ ] 8.6 Create main.go entry point in `cmd/agent/main.go`

## Task 9: Integration Testing
- [ ] 9.1 Create integration test for single agent flow in `test/integration/workflow_test.go`
- [ ] 9.2 Create integration test for multi-agent flow
- [ ] 9.3 Create integration test for CLI mode switching

## Task 10: Documentation
- [ ] 10.1 Create docs/wiki/README.md with wiki structure
- [ ] 10.2 Create docs/wiki/challenges/ directory with template
- [ ] 10.3 Create docs/wiki/tradeoffs/ directory with template
- [ ] 10.4 Create docs/wiki/observations/ directory with template
- [ ] 10.5 Write project README.md with setup and usage instructions

## Task 11: MCP Integration (Future Phase)
- [ ]* 11.1 Define MCPClient interface in `internal/mcp/client.go`
- [ ]* 11.2 Implement StdioMCPClient in `internal/mcp/stdio.go`
- [ ]* 11.3 Implement MCPToolWrapper in `internal/mcp/wrapper.go`
- [ ]* 11.4 Implement MCPManager in `internal/mcp/manager.go`
- [ ]* 11.5 Implement config loading in `internal/mcp/config.go`
- [ ]* 11.6 Write unit tests for MCP components
- [ ]* 11.7 (PBT) Write property test for MCP tool interface compliance (Property 21)
- [ ]* 11.8 Create example mcp.json configuration file
