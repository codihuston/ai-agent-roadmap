# Learnings Document

This document captures design decisions, challenges, and observations encountered during the development of the agentic system POC.

## Table of Contents

1. [JSON Serialization](#json-serialization)
2. [Property-Based Testing](#property-based-testing)
3. [LLM Provider Abstraction](#llm-provider-abstraction)
4. [Tool Interface and Implementations](#tool-interface-and-implementations)

---

## JSON Serialization

### Challenge: nil vs empty map in JSON round-trips

**Context**: When deserializing JSON into Go structs, optional fields like `Parameters map[string]interface{}` can end up as `nil` if not present in the JSON, but the original struct may have had an empty map `map[string]interface{}{}`.

**Problem**: `reflect.DeepEqual(nil, map[string]interface{}{})` returns `false`, causing round-trip tests to fail even though the semantic meaning is the same.

**Solution**: Normalize `nil` maps to empty maps during parsing:

```go
// In ParsePlan, after unmarshaling:
for i := range plan.Steps {
    if plan.Steps[i].Parameters == nil {
        plan.Steps[i].Parameters = make(map[string]interface{})
    }
}
```

**Trade-off**: This adds a small overhead but ensures consistent behavior and makes testing simpler.

---

## Property-Based Testing

### Challenge: Generating valid test data with gopter

**Context**: Using `gopter` for property-based testing requires careful generator construction to avoid high discard rates.

**Problem**: Using `SuchThat` filters on slice generators (e.g., requiring at least 1 element) causes many generated values to be discarded, leading to "gave up" errors.

**Solution**: Use `FlatMap` to first generate the count, then generate exactly that many elements:

```go
// Instead of:
gen.SliceOf(genStep()).SuchThat(func(s []Step) bool { return len(s) >= 1 })

// Use:
gen.IntRange(1, 5).FlatMap(func(count interface{}) gopter.Gen {
    n := count.(int)
    stepGens := make([]gopter.Gen, n)
    for i := 0; i < n; i++ {
        stepGens[i] = genStep()
    }
    return gopter.CombineGens(stepGens...)
}, reflect.TypeOf([]Step{}))
```

**Lesson**: Design generators to produce valid data directly rather than filtering invalid data.

### Challenge: Non-empty string generation

**Context**: Required fields like `Goal`, `Description`, and `Action` must be non-empty strings.

**Problem**: `gen.AlphaString()` can generate empty strings, and using `SuchThat` to filter them increases discard rate.

**Solution**: Map empty strings to a default value:

```go
func genNonEmptyAlphaString() gopter.Gen {
    return gen.AlphaString().Map(func(s string) string {
        if len(s) == 0 {
            return "a" // Ensure non-empty
        }
        return s
    })
}
```

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design


---

## LLM Provider Abstraction

### Design Decision: Interface-based provider abstraction

**Context**: The system needs to support multiple LLM providers (Claude, Gemini, OpenAI) without changing agent code.

**Decision**: Define a minimal `LLMProvider` interface with two methods:

```go
type LLMProvider interface {
    Generate(ctx context.Context, req GenerateRequest) (*LLMResponse, error)
    Name() string
}
```

**Rationale**: 
- `Generate` is the core operation - send messages, get response
- `Name` enables logging and debugging to identify which provider is in use
- Keeping the interface small follows the Interface Segregation Principle

### Design Decision: Functional options for provider configuration

**Context**: ClaudeProvider needs configurable model, HTTP client, and base URL.

**Decision**: Use functional options pattern instead of a config struct:

```go
provider, err := NewClaudeProviderWithKey("key",
    WithModel("claude-3-opus"),
    WithHTTPClient(customClient),
    WithBaseURL("https://custom.api.com"),
)
```

**Rationale**:
- Optional parameters without requiring a config struct
- Sensible defaults without nil checks
- Easy to extend with new options
- Clear, readable API at call sites

### Challenge: Claude API message format conversion

**Context**: Claude's API uses a different message format than our internal representation. Tool results must be sent as `tool_result` content blocks with `tool_use_id`.

**Problem**: Our `Message` struct has `ToolCallID` and `ToolName` fields, but Claude expects these in a specific content block format.

**Solution**: Convert messages during request building:

```go
func (c *ClaudeProvider) convertMessage(msg Message) (claudeMsg, error) {
    // Handle tool result messages specially
    if msg.ToolCallID != "" {
        return claudeMsg{
            Role: "user",  // Tool results are always "user" role in Claude
            Content: []contentPart{{
                Type:      "tool_result",
                ToolUseID: msg.ToolCallID,
                Content:   msg.Content,
            }},
        }, nil
    }
    // Regular text messages...
}
```

**Lesson**: Keep internal message format simple and provider-agnostic; handle API-specific conversions in the provider implementation.

### Challenge: Error wrapping with context

**Context**: Requirement 1.3 states errors must be handled gracefully with descriptive messages.

**Decision**: All errors are wrapped with `claude:` prefix and operation context:

```go
return nil, fmt.Errorf("claude: failed to send request: %w", err)
return nil, fmt.Errorf("claude: authentication failed: %s", errResp.Error.Message)
```

**Rationale**:
- Errors clearly identify which provider failed
- Original error is preserved with `%w` for unwrapping
- HTTP status codes are translated to meaningful messages (401 → "authentication failed")

### Testing Strategy: Mock HTTP server

**Context**: Testing the provider without making real API calls.

**Decision**: Use `httptest.NewServer` to create mock servers that return controlled responses:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Verify headers, return mock response
}))
defer server.Close()

provider, _ := NewClaudeProviderWithKey("key", WithBaseURL(server.URL))
```

**Benefits**:
- Tests run fast without network calls
- Can test error scenarios (rate limits, auth failures)
- Can verify request format (headers, body structure)
- No API key required for testing

---

## Tool Interface and Implementations

### Design Decision: Tool interface with ToolResult from provider package

**Context**: Tools need to return results that can be sent back to the LLM. The `ToolResult` type was already defined in `internal/provider/types.go`.

**Decision**: Import `ToolResult` from the provider package rather than defining a duplicate:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error)
}
```

**Rationale**:
- Avoids type duplication and conversion overhead
- `ToolResult` is already designed for LLM communication
- Single source of truth for the result structure

### Design Decision: Helper functions for Tool to ToolDefinition conversion

**Context**: Agents need to pass tool definitions to the LLM provider.

**Decision**: Provide `ToDefinition` and `ToDefinitions` helper functions:

```go
func ToDefinition(t Tool) provider.ToolDefinition
func ToDefinitions(tools []Tool) []provider.ToolDefinition
```

**Rationale**:
- Keeps conversion logic in one place
- Agents don't need to know about ToolDefinition structure
- Easy to extend if conversion logic changes

### Challenge: Type conversion for numeric arguments

**Context**: JSON unmarshaling produces `float64` for all numbers, but callers might pass `int` or `int64`.

**Problem**: Calculator tool receives arguments from JSON parsing (always `float64`) or direct Go calls (might be `int`).

**Solution**: Create a `toFloat64` helper that handles multiple numeric types:

```go
func toFloat64(v interface{}) (float64, error) {
    switch n := v.(type) {
    case float64:
        return n, nil
    case int:
        return float64(n), nil
    case int64:
        return float64(n), nil
    default:
        return 0, fmt.Errorf("expected number, got %T", v)
    }
}
```

**Lesson**: Always handle type flexibility when dealing with `interface{}` arguments from JSON.

### Design Decision: Security for file tools with basePath

**Context**: FileReaderTool and FileWriterTool need to prevent directory traversal attacks.

**Decision**: All file paths are resolved relative to a `basePath` and validated:

```go
// Clean the path to prevent directory traversal attacks
fullPath = filepath.Clean(fullPath)

// Verify the path is still within basePath
rel, err := filepath.Rel(absBase, absPath)
if err != nil || len(rel) > 0 && rel[0] == '.' {
    return &provider.ToolResult{
        Success: false,
        Error:   "path escapes base directory",
    }, nil
}
```

**Rationale**:
- Prevents `../../../etc/passwd` style attacks
- Tools can only access files within their designated workspace
- Security is enforced at the tool level, not relying on callers

### Design Decision: FinishPlanTool captures plan for orchestrator

**Context**: The Architect agent needs to output a plan that the orchestrator can retrieve.

**Decision**: FinishPlanTool stores the captured plan internally with thread-safe access:

```go
type FinishPlanTool struct {
    capturedPlan string
    mu           sync.RWMutex
}

func (f *FinishPlanTool) GetCapturedPlan() string
func (f *FinishPlanTool) ClearCapturedPlan()
func (f *FinishPlanTool) HasCapturedPlan() bool
```

**Rationale**:
- Orchestrator creates the tool, passes it to Architect, then retrieves the plan
- Thread-safe for potential concurrent access
- Clear separation: tool captures, orchestrator retrieves

### Testing Strategy: Property-based tests for Calculator

**Context**: Property 7 requires mathematical correctness for all valid operations.

**Decision**: Use gopter to test mathematical properties:

```go
// Property: Addition produces mathematically correct results
properties.Property("add returns a + b", prop.ForAll(
    func(a, b float64) bool {
        result, _ := calc.Execute(ctx, args)
        got, _ := strconv.ParseFloat(result.Output, 64)
        return floatEquals(got, a+b)
    },
    genFiniteFloat(),
    genFiniteFloat(),
))
```

**Key decisions**:
- Filter out NaN and Inf values (not valid calculator inputs)
- Use relative tolerance for float comparison
- Test commutativity as an additional mathematical property
- Explicitly test division by zero returns error

---

## Conversation Memory

### Design Decision: Thread-safe ConversationMemory with sync.RWMutex

**Context**: ConversationMemory maintains conversation history that may be accessed concurrently by multiple goroutines (e.g., agent loop and monitoring).

**Decision**: Use `sync.RWMutex` for thread-safe access:

```go
type ConversationMemory struct {
    messages []provider.Message
    mu       sync.RWMutex
}
```

**Rationale**:
- `RWMutex` allows multiple concurrent readers (GetMessages, Len)
- Writers (AddMessage, AddToolResult, Clear) get exclusive access
- Better performance than `Mutex` for read-heavy workloads

### Design Decision: GetMessages returns a copy

**Context**: Callers might modify the returned slice, which could corrupt internal state.

**Decision**: Always return a copy of the messages slice:

```go
func (m *ConversationMemory) GetMessages() []provider.Message {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    result := make([]provider.Message, len(m.messages))
    copy(result, m.messages)
    return result
}
```

**Rationale**:
- Prevents external modification of internal state
- Callers can safely modify the returned slice
- Small memory overhead is acceptable for safety

### Design Decision: Separate AddMessage and AddToolResult methods

**Context**: Tool results have additional metadata (ToolCallID, ToolName) that regular messages don't have.

**Decision**: Provide two separate methods instead of one generic method:

```go
func (m *ConversationMemory) AddMessage(role, content string)
func (m *ConversationMemory) AddToolResult(toolCallID, toolName, result string)
```

**Rationale**:
- Clear API - callers know exactly what parameters are needed
- AddToolResult automatically sets Role to "tool"
- Prevents errors from forgetting to set tool-specific fields
- Type-safe at compile time

### Testing Strategy: Property-based tests for ordering preservation

**Context**: Property 1 requires that messages are returned in the exact order they were added.

**Decision**: Test multiple scenarios with gopter:

1. **Message ordering**: Add sequence of messages, verify order preserved
2. **Content preservation**: Verify exact byte-for-byte content match
3. **Tool result ordering**: Same as messages but for tool results
4. **Mixed ordering**: Interleaved messages and tool results
5. **Clear resets**: Verify Clear() removes all messages
6. **Copy isolation**: Verify GetMessages() returns independent copy

**Key insight**: Testing with random sequences catches edge cases that example-based tests miss (e.g., empty strings, special characters, very long sequences).

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design, mock HTTP servers
- **LLM Providers**: Interface design, Claude API format, error handling
- **Tool Calling**: Interface design, type conversion, security, plan capture
- **Memory**: Thread safety, defensive copying, API design


---

## Agent Core Loop

### Design Decision: Agent struct with tool map for O(1) lookup

**Context**: The agent needs to dispatch tool calls to the correct tool implementation quickly.

**Decision**: Store tools in a `map[string]tool.Tool` keyed by tool name:

```go
type Agent struct {
    provider      provider.LLMProvider
    tools         map[string]tool.Tool
    systemPrompt  string
    maxIterations int
}
```

**Rationale**:
- O(1) lookup by tool name during execution
- Easy to check if a tool exists
- Simple to add/remove tools dynamically with RegisterTool()

### Design Decision: AgentConfig struct for constructor

**Context**: Agent has multiple configuration options (provider, tools, system prompt, max iterations).

**Decision**: Use a config struct instead of multiple constructor parameters:

```go
type AgentConfig struct {
    Provider      provider.LLMProvider
    Tools         []tool.Tool
    SystemPrompt  string
    MaxIterations int
}

func NewAgent(cfg AgentConfig) *Agent
```

**Rationale**:
- Clear, self-documenting API
- Easy to add new configuration options without breaking existing code
- Optional fields have natural zero values (empty string, 0)
- Default MaxIterations (10) applied when not set

### Design Decision: Sentinel error for max iterations

**Context**: Callers need to distinguish "max iterations exceeded" from other errors.

**Decision**: Define a sentinel error that can be checked with `errors.Is`:

```go
var ErrMaxIterationsExceeded = errors.New("max iterations exceeded")

// In Run():
return nil, fmt.Errorf("%w: reached %d iterations without final response", 
    ErrMaxIterationsExceeded, a.maxIterations)
```

**Rationale**:
- Callers can use `errors.Is(err, ErrMaxIterationsExceeded)` for specific handling
- Error message includes context (iteration count)
- Follows Go error handling best practices

### Design Decision: Tool errors don't crash the agent

**Context**: Property 9 requires that tool execution errors don't crash the system.

**Decision**: executeTool returns error messages as strings, never panics:

```go
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
```

**Rationale**:
- Agent continues operation even when tools fail
- LLM receives error information and can decide how to proceed
- Three error cases handled: unknown tool, execution error, result failure
- Consistent error message format for LLM parsing

### Design Decision: Track all tool calls in AgentResult

**Context**: Callers may want to know what tools were used during a run.

**Decision**: AgentResult includes all tool calls made:

```go
type AgentResult struct {
    Response      string
    ToolCallsMade []provider.ToolCall
    Iterations    int
}
```

**Rationale**:
- Enables observability and debugging
- Useful for logging and auditing
- Supports Property 14 (Coder returns action summary)

### Challenge: Memory management across iterations

**Context**: The agent loop modifies conversation memory across multiple iterations.

**Problem**: Need to ensure memory is updated correctly for both tool results and final responses.

**Solution**: 
1. Add user input at the start of Run()
2. Add tool results after each tool execution
3. Add assistant response only when loop terminates with final response

```go
// Start of Run()
mem.AddMessage("user", input)

// After tool execution
mem.AddToolResult(tc.ID, tc.Name, result)

// On final response (no tool calls)
mem.AddMessage("assistant", resp.Text)
```

**Lesson**: Be explicit about when memory is modified to avoid duplicate or missing messages.

### Testing Strategy: Mock LLM provider for deterministic tests

**Context**: Testing the agent loop requires controlling LLM responses.

**Decision**: Create a mockLLMProvider that returns predefined responses:

```go
type mockLLMProvider struct {
    responses []provider.LLMResponse
    errors    []error
    callCount int
    requests  []provider.GenerateRequest
}
```

**Key features**:
- `responses`: Sequence of responses to return
- `errors`: Sequence of errors to return (nil for success)
- `callCount`: Track how many times Generate was called
- `requests`: Capture all requests for verification

**Benefits**:
- Deterministic tests - same input always produces same output
- Can test error scenarios
- Can verify request contents (messages, tools, system prompt)
- Can test multi-iteration scenarios by providing multiple responses

### Testing Strategy: Property-based tests for loop termination

**Context**: Properties 10, 11, 12 define loop termination behavior.

**Decision**: Use gopter to test with generated scenarios:

1. **Property 10** (continues while tool calls exist):
   - Generate random number of tool call rounds (0-8)
   - Verify agent makes correct number of LLM calls
   - Verify tool is called correct number of times

2. **Property 11** (terminates on final response):
   - Generate random response text
   - Verify only one LLM call made
   - Verify response matches exactly

3. **Property 12** (max iterations error):
   - Generate random max iterations (1-15)
   - Always return tool calls (never final response)
   - Verify ErrMaxIterationsExceeded returned
   - Verify exact number of iterations made

**Key insight**: Property tests catch edge cases like:
- Zero tool call rounds (immediate final response)
- Single iteration max (fails immediately)
- Empty response text

---

## Specialized Agents

### Design Decision: Factory functions for specialized agents

**Context**: The system needs Architect and Coder agents with different configurations (tools, system prompts).

**Decision**: Create factory functions that return pre-configured agents:

```go
func NewArchitectAgent(provider LLMProvider) (*Agent, *FinishPlanTool)
func NewCoderAgent(provider LLMProvider, basePath string) *Agent
```

**Rationale**:
- Encapsulates agent configuration in one place
- Callers don't need to know which tools each agent needs
- System prompts are defined as constants for easy review/modification
- Architect returns the FinishPlanTool so orchestrator can retrieve captured plans

### Design Decision: Architect returns FinishPlanTool reference

**Context**: The orchestrator needs to retrieve the plan captured by the Architect agent.

**Decision**: NewArchitectAgent returns both the agent and the FinishPlanTool:

```go
func NewArchitectAgent(provider LLMProvider) (*Agent, *FinishPlanTool) {
    finishPlanTool := tool.NewFinishPlanTool()
    agent := NewAgent(AgentConfig{
        Provider: provider,
        Tools:    []tool.Tool{finishPlanTool},
        // ...
    })
    return agent, finishPlanTool
}
```

**Rationale**:
- Orchestrator creates the agent and keeps reference to the tool
- After agent runs, orchestrator calls `finishPlanTool.GetCapturedPlan()`
- Clean separation of concerns - agent doesn't know about orchestrator
- Tool is created fresh each time, avoiding state leakage between runs

### Design Decision: Coder agent takes basePath parameter

**Context**: The Coder agent needs to read/write files within a specific workspace.

**Decision**: basePath is a required parameter for NewCoderAgent:

```go
func NewCoderAgent(provider LLMProvider, basePath string) *Agent
```

**Rationale**:
- Security: all file operations are sandboxed to basePath
- Flexibility: different Coder instances can work in different directories
- Explicit: caller must think about where files will be created
- Consistent with FileReaderTool and FileWriterTool design

### Design Decision: Detailed system prompts with clear instructions

**Context**: LLMs need clear instructions to behave as expected.

**Decision**: System prompts include:
1. Role description (who the agent is)
2. Responsibilities (what it should do)
3. Available tools and their usage
4. Guidelines for behavior
5. Expected output format

**Example (Architect)**:
```go
const ArchitectSystemPrompt = `You are an Architect agent responsible for breaking down high-level goals into detailed implementation plans.

Your role is to:
1. Analyze the user's goal...
2. Break down the goal into clear, actionable steps...
...

When you have completed your plan, you MUST call the finish_plan tool with:
- goal: The original goal being addressed
- steps: An array of steps...

Guidelines for creating plans:
- Be specific and detailed...
...

Always call finish_plan when your plan is complete. Do not provide the plan as text - use the tool.`
```

**Rationale**:
- Clear instructions reduce LLM confusion
- Explicit tool usage requirements ensure consistent behavior
- Guidelines help produce higher quality output
- "MUST" language emphasizes required behavior

### Testing Strategy: Integration-style tests with real file operations

**Context**: Coder agent tests need to verify actual file operations.

**Decision**: Use temporary directories for realistic testing:

```go
func TestCoderAgent_ExecutesWriteFile(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "coder_test")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    // ... test with real file operations ...
    
    // Verify file was actually created
    content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
    // ...
}
```

**Rationale**:
- Tests verify actual behavior, not just mock interactions
- Catches real issues (permissions, path handling, directory creation)
- Cleanup with `defer os.RemoveAll` ensures no test artifacts remain
- More confidence that code works in production

### Testing Strategy: Table-driven tests for agent configuration

**Context**: Need to verify agent configuration is correct.

**Decision**: Use simple verification tests for factory functions:

```go
func TestNewArchitectAgent(t *testing.T) {
    agent, finishPlanTool := NewArchitectAgent(mockProvider)
    
    // Verify agent is created
    if agent == nil { t.Fatal("expected agent") }
    
    // Verify tool is returned
    if finishPlanTool == nil { t.Fatal("expected tool") }
    
    // Verify system prompt
    if agent.systemPrompt != ArchitectSystemPrompt { t.Error("wrong prompt") }
    
    // Verify tools registered
    tools := agent.GetTools()
    // ...
}
```

**Rationale**:
- Quick verification that factory functions work correctly
- Catches configuration errors early
- Documents expected behavior

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design, mock HTTP servers, mock LLM providers, integration tests with temp directories
- **LLM Providers**: Interface design, Claude API format, error handling
- **Tool Calling**: Interface design, type conversion, security, plan capture
- **Memory**: Thread safety, defensive copying, API design
- **Agent Loop**: Think-Act-Observe pattern, error handling, iteration limits, memory management
- **Specialized Agents**: Factory functions, system prompts, tool configuration, orchestrator integration


---

## CLI Interface

### Design Decision: Dependency injection for input/output streams

**Context**: The CLI needs to be testable without requiring actual stdin/stdout interaction.

**Decision**: Accept `io.Reader` and `io.Writer` interfaces for input/output:

```go
type CLI struct {
    provider provider.LLMProvider
    output   io.Writer
    input    *bufio.Scanner
    basePath string
}

func NewCLI(llmProvider provider.LLMProvider) *CLI
func NewCLIWithIO(llmProvider provider.LLMProvider, input io.Reader, output io.Writer) *CLI
```

**Rationale**:
- `NewCLI` uses os.Stdin/os.Stdout for production use
- `NewCLIWithIO` allows injecting `bytes.Buffer` and `strings.Reader` for testing
- Follows Dependency Inversion Principle - depend on interfaces, not concrete types
- Tests can verify output content and simulate user input

### Design Decision: Separate methods for single and multi-agent modes

**Context**: The CLI supports two distinct modes of operation.

**Decision**: Provide separate public methods for each mode:

```go
func (c *CLI) RunSingleAgentMode() error
func (c *CLI) RunMultiAgentMode() error
```

**Rationale**:
- Clear separation of concerns
- Each method is focused and easier to test
- main.go can select mode based on command-line flag
- Easier to add new modes in the future

### Design Decision: Case-insensitive exit commands

**Context**: Users might type "exit", "EXIT", "Exit", "quit", etc.

**Decision**: Normalize input to lowercase before checking:

```go
func isExitCommand(input string) bool {
    lower := strings.ToLower(strings.TrimSpace(input))
    return lower == "exit" || lower == "quit"
}
```

**Rationale**:
- Better user experience - any case works
- Handles accidental whitespace
- Simple, predictable behavior

### Design Decision: Display intermediate steps for observability

**Context**: Requirement 9.5 requires showing tool calls and agent transitions.

**Decision**: Print tool calls and transitions with clear formatting:

```go
func (c *CLI) printToolCall(tc provider.ToolCall) {
    c.printf("  [Tool Call] %s\n", tc.Name)
    for key, value := range tc.Arguments {
        c.printf("    %s: %v\n", key, value)
    }
}

func (c *CLI) printAgentTransition(from, to string) {
    c.printf("\n>>> Agent Transition: %s -> %s\n\n", from, to)
}
```

**Rationale**:
- Users can see what the agent is doing
- Helpful for debugging and understanding agent behavior
- Clear visual markers (`[Tool Call]`, `>>>`) make output scannable
- Arguments are displayed for full transparency

### Design Decision: Graceful handling of EOF

**Context**: Users might pipe input or close stdin unexpectedly.

**Decision**: Treat EOF as a graceful exit:

```go
if !c.input.Scan() {
    if err := c.input.Err(); err != nil {
        return fmt.Errorf("input error: %w", err)
    }
    c.println("\nGoodbye!")
    return nil
}
```

**Rationale**:
- Supports piped input (`echo "hello" | agent`)
- Ctrl+D (EOF) exits cleanly
- Actual errors are still reported
- Consistent "Goodbye!" message for all exit paths

### Design Decision: Skip empty input lines

**Context**: Users might accidentally press Enter without typing anything.

**Decision**: Continue the loop on empty input:

```go
input := strings.TrimSpace(c.input.Text())
if input == "" {
    continue
}
```

**Rationale**:
- Avoids sending empty prompts to the LLM
- Better user experience - no error for accidental Enter
- Consistent behavior in both modes

### Design Decision: main.go uses flag package for CLI arguments

**Context**: Need to select between single and multi-agent modes.

**Decision**: Use standard library `flag` package:

```go
mode := flag.String("mode", "single", "Mode to run: 'single' or 'multi'")
basePath := flag.String("path", ".", "Base path for file operations")
help := flag.Bool("help", false, "Show help message")
```

**Rationale**:
- Standard library - no external dependencies
- Familiar interface for Go developers
- Automatic help generation with `-h`
- Default values clearly specified

### Testing Strategy: Mock provider with controlled responses

**Context**: CLI tests need to verify behavior without real LLM calls.

**Decision**: Create a simple mock provider for testing:

```go
type mockProvider struct {
    responses []*provider.LLMResponse
    callIndex int
    calls     []provider.GenerateRequest
}
```

**Key features**:
- Returns predefined responses in sequence
- Tracks all calls for verification
- Simple implementation focused on CLI testing needs

### Testing Strategy: Table-driven tests for exit commands

**Context**: Need to verify all variations of exit commands work.

**Decision**: Use table-driven tests:

```go
tests := []struct {
    input    string
    expected bool
}{
    {"exit", true},
    {"EXIT", true},
    {"quit", true},
    {"hello", false},
    // ...
}
```

**Rationale**:
- Easy to add new test cases
- Clear documentation of expected behavior
- Follows Go testing conventions

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design, mock HTTP servers, mock LLM providers, integration tests with temp directories, table-driven tests
- **LLM Providers**: Interface design, Claude API format, error handling
- **Tool Calling**: Interface design, type conversion, security, plan capture
- **Memory**: Thread safety, defensive copying, API design
- **Agent Loop**: Think-Act-Observe pattern, error handling, iteration limits, memory management
- **Specialized Agents**: Factory functions, system prompts, tool configuration, orchestrator integration
- **CLI**: Dependency injection, mode separation, user input handling, observability


---

## Integration Testing

### Design Decision: Separate integration test directory

**Context**: Integration tests verify end-to-end workflows across multiple components.

**Decision**: Place integration tests in `test/integration/` rather than alongside unit tests:

```
agentic-poc/
├── internal/
│   └── agent/
│       └── agent_test.go      # Unit tests
└── test/
    └── integration/
        └── workflow_test.go   # Integration tests
```

**Rationale**:
- Clear separation between unit and integration tests
- Integration tests can be run separately (`go test ./test/integration/...`)
- Follows Go project conventions
- Easier to manage test dependencies and setup

### Design Decision: Multiple mock provider implementations

**Context**: Different integration tests need different mocking strategies.

**Decision**: Create specialized mock providers for different scenarios:

```go
// Simple sequential responses
type mockLLMProvider struct {
    responses []provider.LLMResponse
    callCount int
}

// Multi-agent workflow with agent detection
type sequentialMockProvider struct {
    responseSequences [][]provider.LLMResponse
    currentSequence   int
}

// Request tracking for verification
type planTrackingMockProvider struct {
    architectResponses []provider.LLMResponse
    coderResponses     []provider.LLMResponse
    coderRequests      *[]provider.GenerateRequest
}
```

**Rationale**:
- Each mock is focused on specific testing needs
- `sequentialMockProvider` detects agent type from system prompt
- `planTrackingMockProvider` captures requests for verification
- Avoids complex single mock with many configuration options

### Challenge: Detecting which agent is calling the provider

**Context**: In multi-agent workflows, the same provider is used by both Architect and Coder agents.

**Problem**: Need to return different responses based on which agent is making the call.

**Solution**: Detect agent type from the system prompt:

```go
func (m *sequentialMockProvider) Generate(ctx context.Context, req provider.GenerateRequest) (*provider.LLMResponse, error) {
    if strings.Contains(req.SystemPrompt, "Architect") {
        // Return architect responses
    } else if strings.Contains(req.SystemPrompt, "Coder") {
        // Return coder responses
    }
}
```

**Rationale**:
- System prompts are unique to each agent type
- No need to modify agent code for testing
- Works with existing agent implementations

### Design Decision: Use real tools in integration tests

**Context**: Integration tests should verify actual behavior, not just mock interactions.

**Decision**: Use real tool implementations (Calculator, FileReader, FileWriter) with mocked LLM:

```go
// Real calculator tool
calcTool := tool.NewCalculatorTool()
agentInstance := agent.NewAgent(agent.AgentConfig{
    Provider: mockProvider,  // Mocked LLM
    Tools:    []tool.Tool{calcTool},  // Real tool
})
```

**Rationale**:
- Verifies tool execution actually works
- Catches integration issues between agent and tools
- Only the LLM is mocked (external dependency)
- File operations create real files in temp directories

### Design Decision: Temporary directories for file operations

**Context**: Integration tests that write files need isolated workspaces.

**Decision**: Use `t.TempDir()` for automatic cleanup:

```go
func TestMultiAgentFlow_ArchitectCoderWorkflow(t *testing.T) {
    tmpDir := t.TempDir()  // Automatically cleaned up
    
    orch := orchestrator.NewOrchestrator(mockProvider, tmpDir)
    // ... test creates files in tmpDir ...
    
    // Verify file was created
    content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
}
```

**Rationale**:
- `t.TempDir()` handles cleanup automatically
- Each test gets isolated directory
- No test artifacts left behind
- Can verify actual file contents

### Testing Strategy: Verify workflow state transitions

**Context**: Orchestrator should transition through phases correctly.

**Decision**: Test state at key points:

```go
// Initial state
state := orch.State()
if state.Phase != orchestrator.PhaseIdle { ... }

// After run
result, err := orch.Run(ctx, goal)

// Final state
state = orch.State()
if state.Phase != orchestrator.PhaseComplete { ... }
```

**Rationale**:
- Verifies Property 16 (state reflects current phase)
- Catches state management bugs
- Documents expected state transitions

### Testing Strategy: Verify plan is passed to Coder

**Context**: Property 15 requires Architect's plan to be passed to Coder.

**Decision**: Use request-tracking mock to capture Coder's input:

```go
type planTrackingMockProvider struct {
    coderRequests *[]provider.GenerateRequest
}

// After test
firstCoderReq := coderRequests[0]
for _, msg := range firstCoderReq.Messages {
    if msg.Role == "user" && strings.Contains(msg.Content, "Create config file") {
        foundPlan = true
    }
}
```

**Rationale**:
- Verifies data flow between agents
- Catches serialization/deserialization issues
- Documents expected message format

### Testing Strategy: CLI mode switching tests

**Context**: CLI should support both single and multi-agent modes.

**Decision**: Test mode selection with table-driven tests:

```go
tests := []struct {
    name       string
    runMode    func(*cli.CLI) error
    wantHeader string
}{
    {"single agent mode", func(c *cli.CLI) error { return c.RunSingleAgentMode() }, "Single Agent Mode"},
    {"multi agent mode", func(c *cli.CLI) error { return c.RunMultiAgentMode() }, "Multi-Agent Mode"},
}
```

**Rationale**:
- Verifies both modes work correctly
- Easy to add new modes
- Clear documentation of expected behavior

### Testing Strategy: Graceful exit command tests

**Context**: Both "exit" and "quit" should work in both modes, case-insensitive.

**Decision**: Comprehensive table-driven tests:

```go
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
        t.Run(mode.name+"_"+cmd, func(t *testing.T) { ... })
    }
}
```

**Rationale**:
- Tests all combinations (12 test cases)
- Catches case-sensitivity bugs
- Verifies consistent behavior across modes

---

## Categories

- **Parsing**: JSON serialization, nil vs empty handling
- **Testing**: Property-based testing with gopter, generator design, mock HTTP servers, mock LLM providers, integration tests with temp directories, table-driven tests, workflow verification
- **LLM Providers**: Interface design, Claude API format, error handling
- **Tool Calling**: Interface design, type conversion, security, plan capture
- **Memory**: Thread safety, defensive copying, API design
- **Agent Loop**: Think-Act-Observe pattern, error handling, iteration limits, memory management
- **Specialized Agents**: Factory functions, system prompts, tool configuration, orchestrator integration
- **CLI**: Dependency injection, mode separation, user input handling, observability
- **Integration Testing**: Mock provider strategies, agent detection, real tool usage, state verification
