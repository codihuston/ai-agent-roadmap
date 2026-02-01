# Observations: Building Agentic AI Systems

This document captures high-level observations and insights from building this agentic system POC - what you can control, what you can't, and where the real opportunities lie.

---

## Where You Get to "Play" (Your Differentiation Points)

### 1. Tools are YOUR Domain 🔧

The LLM is generic - it's the same Claude for everyone. But YOUR tools are unique:
- What APIs do you connect to?
- What databases can you query?
- What internal systems can you automate?
- What domain-specific calculations matter to your users?

**Example**: We built a calculator, but you could build:
- A tool that queries your company's inventory system
- A tool that creates Jira tickets
- A tool that deploys to your specific infrastructure
- A tool that searches your proprietary knowledge base

### 2. System Prompts are YOUR Secret Sauce 📝

The Architect and Coder have different personalities because of their system prompts. This is where you inject:
- Domain expertise ("You are a financial analyst...")
- Guardrails ("Never execute trades over $10k without confirmation")
- Personality ("Be concise" vs "Explain your reasoning")
- Workflow rules ("Always check inventory before promising delivery")

### 3. Orchestration Logic is YOUR Workflow 🔄

We built Architect → Coder. But you could build:
- Researcher → Writer → Editor → Publisher
- Planner → Executor → Validator → Reporter
- Intake → Triage → Specialist → Resolution

---

## What You CAN'T Really Change

| Layer | Your Control | Why |
|-------|-------------|-----|
| LLM reasoning | ❌ Low | It's a black box - you prompt, it responds |
| Token costs | ❌ Low | API pricing is what it is |
| Latency | ❌ Low | Network + LLM inference time |
| Hallucinations | ⚠️ Medium | Better prompts help, but can't eliminate |

---

## Complexity Considerations

### Code Maintenance

```
Simple ────────────────────────────────────────────── Complex

Single Agent          Multi-Agent           Dynamic Agent
with 2-3 tools        with fixed workflow   selection/routing
     │                      │                      │
     ▼                      ▼                      ▼
  ~500 LOC              ~2000 LOC              ~5000+ LOC
  Easy to debug         Moderate               Hard to trace
```

### Ops Considerations

| Concern | What to Think About |
|---------|---------------------|
| **Cost** | Each LLM call costs money. Multi-agent = multiple calls per request |
| **Latency** | Users wait while LLM thinks. Tool calls add more round trips |
| **Reliability** | LLM can fail, tools can fail, network can fail |
| **Observability** | How do you debug when the LLM makes a weird decision? |
| **Security** | Tools have real power - file writes, API calls, etc. |

---

## Key Takeaways from This POC

### 1. The Agent is Just Plumbing

It's a loop that shuttles messages between user, LLM, and tools. The "intelligence" is in the LLM.

```go
for {
    response := llm.Generate(messages)
    if response.HasToolCalls() {
        results := executeTool(response.ToolCalls)
        messages.Add(results)
    } else {
        return response.Text
    }
}
```

### 2. Tools are the Interface to the Real World

Without tools, the LLM can only talk. Tools let it DO things.

```
LLM alone:     "I would calculate 5+3 as 8"
LLM + tools:   [calls calculator] → "5+3 = 8"
```

### 3. Memory Matters

We had to reset memory each prompt to avoid token explosion. Real systems need smarter memory management:
- Summarization of old context
- Selective retrieval (RAG)
- Sliding window approaches

### 4. MCP is About Interoperability

Same tools can be used by different agents/systems via standard protocol:
- Your tools work with any MCP-compatible client
- You can use anyone's MCP servers
- Decouples tool implementation from agent implementation

### 5. Testing is Hard

You can't unit test "did the LLM make a good decision?" You test the plumbing, mock the LLM:
- Test that tool calls are dispatched correctly
- Test that results are fed back properly
- Test error handling and edge cases
- Property-based tests for invariants

---

## Where to Go From Here

| Direction | Complexity | Value |
|-----------|------------|-------|
| **Add more tools** | Low | High - immediate utility |
| **Better prompts** | Low | High - improves quality |
| **RAG (retrieval)** | Medium | High - grounds responses in your data |
| **Memory/context management** | Medium | Medium - longer conversations |
| **Agent routing** | High | Medium - "which agent handles this?" |
| **Fine-tuning** | High | Variable - domain-specific behavior |

---

## The Real Question

**What problem are you solving?**

- If it's "I want to automate X workflow" → Focus on **tools + orchestration**
- If it's "I want better answers about Y domain" → Focus on **RAG + prompts**
- If it's "I want a general assistant" → You're competing with ChatGPT (hard)

### The Sweet Spot

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   Specific Domain + Specific Tools + Good Prompts           │
│                         =                                   │
│              Differentiated Value                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Architecture Decision: Build vs Buy

| Component | Build | Buy/Use |
|-----------|-------|---------|
| LLM | ❌ Use API (Claude, GPT, etc.) | ✅ |
| Agent Loop | ✅ Simple to build | Or use LangChain, etc. |
| Tools | ✅ Your domain expertise | - |
| Orchestration | ✅ Your workflow | - |
| Memory | ⚠️ Can get complex | Consider vector DBs |
| MCP Server | ✅ If you want interop | Or use existing servers |

---

## Common Pitfalls

### 1. Over-engineering the Agent
The agent is just a loop. Don't add complexity until you need it.

### 2. Ignoring Token Costs
Multi-agent workflows can burn through tokens fast. Monitor and optimize.

### 3. Trusting LLM Output
Always validate. The LLM might return malformed JSON, call non-existent tools, or hallucinate parameters.

### 4. Forgetting Error Handling
Tools fail. APIs timeout. Handle gracefully and give the LLM useful error messages.

### 5. Prompt Drift
As you add features, prompts get longer and messier. Keep them organized and tested.

---

## Summary

Building agentic systems is less about AI and more about:
1. **Integration** - Connecting LLMs to your systems via tools
2. **Workflow** - Designing how agents collaborate
3. **Prompting** - Instructing the LLM effectively
4. **Operations** - Monitoring, debugging, and cost management

The LLM provides the reasoning. You provide everything else.


---

## Testing Strategies

### Testing the Plumbing (Easy)

This is what you CAN reliably test:

```go
// Mock the LLM, test the agent loop
mockLLM.Returns(ToolCall{Name: "calculator", Args: {...}})
result := agent.Run("what is 5+3")
assert(mockLLM.WasCalled())
assert(calculator.WasCalledWith(5, 3))
```

| What to Test | How |
|--------------|-----|
| Tool dispatch | Mock LLM → verify correct tool called |
| Error handling | Mock tool failure → verify graceful handling |
| Memory updates | Verify messages added in correct order |
| Max iterations | Verify loop terminates |
| JSON parsing | Test malformed LLM responses |

### Testing Prompts (Hard but Important)

Prompts are code. They should be tested, but it's tricky:

#### 1. Golden Tests (Snapshot Testing)
```
Input: "What is 5+3?"
Expected: LLM calls calculator with add, 5, 3
```
- Record expected behavior
- Alert when behavior changes
- Requires human review of changes

#### 2. Eval Datasets
```python
test_cases = [
    {"input": "calculate 10/2", "expected_tool": "calculator"},
    {"input": "read config.json", "expected_tool": "read_file"},
    {"input": "hello", "expected_tool": None},  # No tool needed
]
```
- Run against real LLM periodically
- Track accuracy over time
- Expensive but catches regressions

#### 3. Property-Based Prompt Testing
```
Property: Math questions should ALWAYS use calculator
Property: File questions should NEVER call calculator
Property: Response should NEVER contain "I don't know" for supported operations
```

#### 4. A/B Testing in Production
- Run two prompt versions
- Measure success rate, user satisfaction
- Gradually roll out winner

### Testing Multi-Agent Systems

| Level | What to Test |
|-------|--------------|
| Unit | Each agent in isolation with mocked LLM |
| Integration | Agent handoffs (Architect → Coder) |
| End-to-end | Full workflow with real LLM (expensive) |

---

## Language & Ecosystem Comparison

### Python 🐍
**Best for**: Rapid prototyping, ML integration, most examples/tutorials

| Pros | Cons |
|------|------|
| LangChain, LlamaIndex, etc. | Slower runtime |
| Huge ecosystem | Type safety is optional |
| Most AI libraries | Deployment can be messy |
| Easy to iterate | GIL limits concurrency |

**Best when**: You're experimenting, need ML integrations, or team knows Python.

### Go 🐹
**Best for**: Production systems, performance, simplicity

| Pros | Cons |
|------|------|
| Fast, compiled | Fewer AI-specific libraries |
| Great concurrency | More boilerplate |
| Single binary deploy | Smaller community for AI |
| Strong typing | Less "batteries included" |

**Best when**: You're building production infrastructure, need performance, or prefer simplicity.

### TypeScript/JavaScript 🟨
**Best for**: Web integration, full-stack teams

| Pros | Cons |
|------|------|
| Vercel AI SDK | Node.js quirks |
| Full-stack friendly | Less performant |
| Good async model | Type system less strict |
| NPM ecosystem | |

**Best when**: Building web apps, team is JS-native, need browser integration.

### Rust 🦀
**Best for**: Performance-critical, safety-critical systems

| Pros | Cons |
|------|------|
| Maximum performance | Steep learning curve |
| Memory safety | Slower development |
| Great for tools | Fewer AI libraries |

**Best when**: Building MCP servers, performance-critical tools, or you love Rust.

### Recommendation

```
Learning/Prototyping → Python
Production Backend   → Go or Python
Web Integration      → TypeScript
Performance Tools    → Rust or Go
```

---

## Scaling to Larger Systems

### Is Multi-Agent Worth It?

**When YES:**
- Tasks are genuinely separable (planning vs execution)
- Different tools/permissions per role
- You need audit trails per agent
- Specialized prompts improve quality significantly

**When NO:**
- Single agent can handle it (simpler is better)
- Latency is critical (each agent = more LLM calls)
- Cost is a concern (2 agents = 2x+ token usage)
- Tasks are tightly coupled

### Coordination Patterns

#### 1. Sequential Pipeline
```
Agent A → Agent B → Agent C → Result
```
- Simple to reason about
- Each agent has clear input/output
- Failure handling is straightforward

#### 2. Supervisor Pattern
```
        ┌─── Agent A
Supervisor ├─── Agent B
        └─── Agent C
```
- Supervisor routes tasks
- Can retry with different agent
- More complex but flexible

#### 3. Collaborative (Hardest)
```
Agent A ←→ Agent B
   ↕         ↕
Agent C ←→ Agent D
```
- Agents communicate directly
- Very hard to debug
- Usually overkill

### What You Need for Larger Systems

| Capability | Why |
|------------|-----|
| **Message Queue** | Decouple agents, handle backpressure |
| **State Management** | Track workflow progress, enable resume |
| **Observability** | Trace requests across agents |
| **Rate Limiting** | Don't blow your API budget |
| **Circuit Breakers** | Handle LLM outages gracefully |

---

## Security: Secrets & Tokens

### The Problem

Tools often need secrets:
- API keys for external services
- Database credentials
- OAuth tokens for user actions

**The LLM should NEVER see these secrets.**

### Pattern 1: Server-Side Secret Injection

```
┌─────────┐     ┌─────────┐     ┌──────────┐
│   LLM   │────▶│  Agent  │────▶│   Tool   │
└─────────┘     └─────────┘     └────┬─────┘
                                     │
    LLM sees: "call github_api"      │ Tool injects
    LLM does NOT see: token          ▼ secret here
                                ┌──────────┐
                                │ GitHub   │
                                │ API      │
                                └──────────┘
```

```go
// Tool implementation - secret never in LLM context
func (t *GitHubTool) Execute(ctx context.Context, args map[string]interface{}) {
    token := os.Getenv("GITHUB_TOKEN")  // Injected at runtime
    // LLM only provided: repo, action
    // Token added server-side
}
```

### Pattern 2: Scoped Tokens

Give tools minimal permissions:

```
❌ Bad:  Tool has admin database access
✅ Good: Tool has read-only access to specific tables
```

### Pattern 3: User Context Tokens

For multi-tenant systems:

```go
type ToolContext struct {
    UserID      string
    Permissions []string
    Token       string  // User-specific, scoped token
}

func (t *Tool) Execute(ctx ToolContext, args map[string]interface{}) {
    // Use ctx.Token for this user's permissions
    // Validate ctx.Permissions before action
}
```

### What NOT to Do

```
❌ Include API keys in system prompt
❌ Let LLM generate authentication headers
❌ Store secrets in conversation memory
❌ Log full tool arguments (may contain secrets)
❌ Return secrets in tool results
```

### Token Usage & Cost Management

#### Tracking Token Usage

```go
type TokenTracker struct {
    InputTokens  int
    OutputTokens int
    Cost         float64
}

func (t *TokenTracker) Add(response LLMResponse) {
    t.InputTokens += response.Usage.InputTokens
    t.OutputTokens += response.Usage.OutputTokens
    t.Cost += calculateCost(response.Usage)
}
```

#### Cost Control Strategies

| Strategy | Implementation |
|----------|----------------|
| **Per-request limits** | Max tokens per LLM call |
| **Per-user limits** | Daily/monthly token budgets |
| **Circuit breaker** | Stop if cost exceeds threshold |
| **Caching** | Cache common tool results |
| **Prompt optimization** | Shorter prompts = fewer tokens |

#### Example: Budget Guard

```go
func (a *Agent) Run(input string) (*Result, error) {
    if a.tokenBudget.Remaining() < MIN_TOKENS {
        return nil, ErrBudgetExceeded
    }
    
    for iteration := 0; iteration < a.maxIterations; iteration++ {
        response, err := a.llm.Generate(messages)
        a.tokenBudget.Deduct(response.Usage)
        
        if a.tokenBudget.Remaining() < MIN_TOKENS {
            return nil, ErrBudgetExceeded
        }
        // ... continue loop
    }
}
```

---

## Production Checklist

### Before Going Live

- [ ] **Secrets**: No secrets in prompts or logs
- [ ] **Rate limits**: Per-user and global limits
- [ ] **Cost tracking**: Know your spend per request
- [ ] **Error handling**: Graceful degradation
- [ ] **Timeouts**: Don't wait forever for LLM
- [ ] **Logging**: Trace requests, but sanitize sensitive data
- [ ] **Monitoring**: Alert on error rates, latency, cost spikes
- [ ] **Testing**: Golden tests for critical prompts

### Observability Essentials

```
Request ID: abc-123
User: user@example.com
Agent: single-agent
Iterations: 3
Tool Calls: [calculator, calculator]
Input Tokens: 1,234
Output Tokens: 567
Cost: $0.02
Latency: 2.3s
Status: success
```

---

## Final Thoughts

Building agentic systems is an exercise in **controlled chaos**:

1. **You control**: Tools, prompts, orchestration, security
2. **You influence**: LLM behavior via prompts
3. **You can't control**: LLM reasoning, hallucinations, costs (fully)

The art is in designing systems that work well when the LLM behaves AND fail gracefully when it doesn't.

```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   Good Agentic System = Good Tools + Good Prompts +        │
│                         Good Error Handling +              │
│                         Good Observability                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
```
### Testing the Plumbing (Easy)

This is what you CAN reliably test:

```go
// Mock the LLM, test the agent loop
mockLLM.Returns(ToolCall{Name: "calculator", Args: {...}})
result := agent.Run("what is 5+3")
assert(mockLLM.WasCalled())
assert(calculator.WasCalledWith(5, 3))
```

| What to Test | How |
|--------------|-----|
| Tool dispatch | Mock LLM → verify correct tool called |
| Error handling | Mock tool failure → verify graceful handling |
| Memory updates | Verify messages added in correct order |
| Max iterations | Verify loop terminates |
| JSON parsing | Test malformed LLM responses |

### Testing Prompts (Hard but Important)

Prompts are code. They should be tested, but it's tricky:

#### 1. Golden Tests (Snapshot Testing)
```
Input: "What is 5+3?"
Expected: LLM calls calculator with add, 5, 3
```
- Record expected behavior
- Alert when behavior changes
- Requires human review of changes

#### 2. Eval Datasets
```python
test_cases = [
    {"input": "calculate 10/2", "expected_tool": "calculator"},
    {"input": "read config.json", "expected_tool": "read_file"},
    {"input": "hello", "expected_tool": None},  # No tool needed
]
```
- Run against real LLM periodically
- Track accuracy over time
- Expensive but catches regressions

#### 3. Property-Based Prompt Testing
```
Property: Math questions should ALWAYS use calculator
Property: File questions should NEVER call calculator
Property: Response should NEVER contain "I don't know" for supported operations
```

#### 4. A/B Testing in Production
- Run two prompt versions
- Measure success rate, user satisfaction
- Gradually roll out winner

### Testing Multi-Agent Systems

| Level | What to Test |
|-------|--------------|
| Unit | Each agent in isolation with mocked LLM |
| Integration | Agent handoffs (Architect → Coder) |
| End-to-end | Full workflow with real LLM (expensive) |

---

## Language & Ecosystem Comparison

### Python 🐍
**Best for**: Rapid prototyping, ML integration, most examples/tutorials

| Pros | Cons |
|------|------|
| LangChain, LlamaIndex, etc. | Slower runtime |
| Huge ecosystem | Type safety is optional |
| Most AI libraries | Deployment can be messy |
| Easy to iterate | GIL limits concurrency |

**Best when**: You're experimenting, need ML integrations, or team knows Python.

### Go 🐹
**Best for**: Production systems, performance, simplicity

| Pros | Cons |
|------|------|
| Fast, compiled | Fewer AI-specific libraries |
| Great concurrency | More boilerplate |
| Single binary deploy | Smaller community for AI |
| Strong typing | Less "batteries included" |

**Best when**: You're building production infrastructure, need performance, or prefer simplicity.

### TypeScript/JavaScript 🟨
**Best for**: Web integration, full-stack teams

| Pros | Cons |
|------|------|
| Vercel AI SDK | Node.js quirks |
| Full-stack friendly | Less performant |
| Good async model | Type system less strict |
| NPM ecosystem | |

**Best when**: Building web apps, team is JS-native, need browser integration.

### Rust 🦀
**Best for**: Performance-critical, safety-critical systems

| Pros | Cons |
|------|------|
| Maximum performance | Steep learning curve |
| Memory safety | Slower development |
| Great for tools | Fewer AI libraries |

**Best when**: Building MCP servers, performance-critical tools, or you love Rust.

### Recommendation

```
Learning/Prototyping → Python
Production Backend   → Go or Python
Web Integration      → TypeScript
Performance Tools    → Rust or Go
```

---

## Scaling to Larger Systems

### Is Multi-Agent Worth It?

**When YES:**
- Tasks are genuinely separable (planning vs execution)
- Different tools/permissions per role
- You need audit trails per agent
- Specialized prompts improve quality significantly

**When NO:**
- Single agent can handle it (simpler is better)
- Latency is critical (each agent = more LLM calls)
- Cost is a concern (2 agents = 2x+ token usage)
- Tasks are tightly coupled

### Coordination Patterns

#### 1. Sequential Pipeline
```
Agent A → Agent B → Agent C → Result
```
- Simple to reason about
- Each agent has clear input/output
- Failure handling is straightforward

#### 2. Supervisor Pattern
```
        ┌─── Agent A
Supervisor ├─── Agent B
        └─── Agent C
```
- Supervisor routes tasks
- Can retry with different agent
- More complex but flexible

#### 3. Collaborative (Hardest)
```
Agent A ←→ Agent B
   ↕         ↕
Agent C ←→ Agent D
```
- Agents communicate directly
- Very hard to debug
- Usually overkill

### What You Need for Larger Systems

| Capability | Why |
|------------|-----|
| **Message Queue** | Decouple agents, handle backpressure |
| **State Management** | Track workflow progress, enable resume |
| **Observability** | Trace requests across agents |
| **Rate Limiting** | Don't blow your API budget |
| **Circuit Breakers** | Handle LLM outages gracefully |

---

## Security: Secrets & Tokens

### The Problem

Tools often need secrets:
- API keys for external services
- Database credentials
- OAuth tokens for user actions

**The LLM should NEVER see these secrets.**

### Pattern 1: Server-Side Secret Injection

```
┌─────────┐     ┌─────────┐     ┌──────────┐
│   LLM   │────▶│  Agent  │────▶│   Tool   │
└─────────┘     └─────────┘     └────┬─────┘
                                     │
    LLM sees: "call github_api"      │ Tool injects
    LLM does NOT see: token          ▼ secret here
                                ┌──────────┐
                                │ GitHub   │
                                │ API      │
                                └──────────┘
```

```go
// Tool implementation - secret never in LLM context
func (t *GitHubTool) Execute(ctx context.Context, args map[string]interface{}) {
    token := os.Getenv("GITHUB_TOKEN")  // Injected at runtime
    // LLM only provided: repo, action
    // Token added server-side
}
```

### Pattern 2: Scoped Tokens

Give tools minimal permissions:

```
❌ Bad:  Tool has admin database access
✅ Good: Tool has read-only access to specific tables
```

### Pattern 3: User Context Tokens

For multi-tenant systems:

```go
type ToolContext struct {
    UserID      string
    Permissions []string
    Token       string  // User-specific, scoped token
}

func (t *Tool) Execute(ctx ToolContext, args map[string]interface{}) {
    // Use ctx.Token for this user's permissions
    // Validate ctx.Permissions before action
}
```

### What NOT to Do

```
❌ Include API keys in system prompt
❌ Let LLM generate authentication headers
❌ Store secrets in conversation memory
❌ Log full tool arguments (may contain secrets)
❌ Return secrets in tool results
```

### Token Usage & Cost Management

#### Tracking Token Usage

```go
type TokenTracker struct {
    InputTokens  int
    OutputTokens int
    Cost         float64
}

func (t *TokenTracker) Add(response LLMResponse) {
    t.InputTokens += response.Usage.InputTokens
    t.OutputTokens += response.Usage.OutputTokens
    t.Cost += calculateCost(response.Usage)
}
```

#### Cost Control Strategies

| Strategy | Implementation |
|----------|----------------|
| **Per-request limits** | Max tokens per LLM call |
| **Per-user limits** | Daily/monthly token budgets |
| **Circuit breaker** | Stop if cost exceeds threshold |
| **Caching** | Cache common tool results |
| **Prompt optimization** | Shorter prompts = fewer tokens |

#### Example: Budget Guard

```go
func (a *Agent) Run(input string) (*Result, error) {
    if a.tokenBudget.Remaining() < MIN_TOKENS {
        return nil, ErrBudgetExceeded
    }
    
    for iteration := 0; iteration < a.maxIterations; iteration++ {
        response, err := a.llm.Generate(messages)
        a.tokenBudget.Deduct(response.Usage)
        
        if a.tokenBudget.Remaining() < MIN_TOKENS {
            return nil, ErrBudgetExceeded
        }
        // ... continue loop
    }
}
```

---

## Production Checklist

### Before Going Live

- [ ] **Secrets**: No secrets in prompts or logs
- [ ] **Rate limits**: Per-user and global limits
- [ ] **Cost tracking**: Know your spend per request
- [ ] **Error handling**: Graceful degradation
- [ ] **Timeouts**: Don't wait forever for LLM
- [ ] **Logging**: Trace requests, but sanitize sensitive data
- [ ] **Monitoring**: Alert on error rates, latency, cost spikes
- [ ] **Testing**: Golden tests for critical prompts

### Observability Essentials

```
Request ID: abc-123
User: user@example.com
Agent: single-agent
Iterations: 3
Tool Calls: [calculator, calculator]
Input Tokens: 1,234
Output Tokens: 567
Cost: $0.02
Latency: 2.3s
Status: success
```

---

## Final Thoughts

Building agentic systems is an exercise in **controlled chaos**:

1. **You control**: Tools, prompts, orchestration, security
2. **You influence**: LLM behavior via prompts
3. **You can't control**: LLM reasoning, hallucinations, costs (fully)

The art is in designing systems that work well when the LLM behaves AND fail gracefully when it doesn't.

```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   Good Agentic System = Good Tools + Good Prompts +        │
│                         Good Error Handling +              │
│                         Good Observability                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
```


---

## Testing Strategies

### Testing the Plumbing (Easy)

This is what you CAN reliably test:

```go
// Mock the LLM, test the agent loop
mockLLM.Returns(ToolCall{Name: "calculator", Args: {...}})
result := agent.Run("what is 5+3")
assert(mockLLM.WasCalled())
assert(calculator.WasCalledWith(5, 3))
```

| What to Test | How |
|--------------|-----|
| Tool dispatch | Mock LLM → verify correct tool called |
| Error handling | Mock tool failure → verify graceful handling |
| Memory updates | Verify messages added in correct order |
| Max iterations | Verify loop terminates |
| JSON parsing | Test malformed LLM responses |

### Testing Prompts (Hard but Important)

Prompts are code. They should be tested, but it's tricky:

#### 1. Golden Tests (Snapshot Testing)
```
Input: "What is 5+3?"
Expected: LLM calls calculator with add, 5, 3
```
- Record expected behavior
- Alert when behavior changes
- Requires human review of changes

#### 2. Eval Datasets
```python
test_cases = [
    {"input": "calculate 10/2", "expected_tool": "calculator"},
    {"input": "read config.json", "expected_tool": "read_file"},
    {"input": "hello", "expected_tool": None},  # No tool needed
]
```
- Run against real LLM periodically
- Track accuracy over time
- Expensive but catches regressions

#### 3. Property-Based Prompt Testing
```
Property: Math questions should ALWAYS use calculator
Property: File questions should NEVER call calculator
Property: Response should NEVER contain "I don't know" for supported operations
```

#### 4. A/B Testing in Production
- Run two prompt versions
- Measure success rate, user satisfaction
- Gradually roll out winner

### Testing Multi-Agent Systems

| Level | What to Test |
|-------|--------------|
| Unit | Each agent in isolation with mocked LLM |
| Integration | Agent handoffs (Architect → Coder) |
| End-to-end | Full workflow with real LLM (expensive) |

---

## Language & Ecosystem Comparison

### Python 🐍
**Best for**: Rapid prototyping, ML integration, most examples/tutorials

| Pros | Cons |
|------|------|
| LangChain, LlamaIndex, etc. | Slower runtime |
| Huge ecosystem | Type safety is optional |
| Most AI libraries | Deployment can be messy |
| Easy to iterate | GIL limits concurrency |

**Best when**: You're experimenting, need ML integrations, or team knows Python.

### Go 🐹
**Best for**: Production systems, performance, simplicity

| Pros | Cons |
|------|------|
| Fast, compiled | Fewer AI-specific libraries |
| Great concurrency | More boilerplate |
| Single binary deploy | Smaller community for AI |
| Strong typing | Less "batteries included" |

**Best when**: You're building production infrastructure, need performance, or prefer simplicity.

### TypeScript/JavaScript 🟨
**Best for**: Web integration, full-stack teams

| Pros | Cons |
|------|------|
| Vercel AI SDK | Node.js quirks |
| Full-stack friendly | Less performant |
| Good async model | Type system less strict |
| NPM ecosystem | |

**Best when**: Building web apps, team is JS-native, need browser integration.

### Rust 🦀
**Best for**: Performance-critical, safety-critical systems

| Pros | Cons |
|------|------|
| Maximum performance | Steep learning curve |
| Memory safety | Slower development |
| Great for tools | Fewer AI libraries |

**Best when**: Building MCP servers, performance-critical tools, or you love Rust.

### Recommendation

```
Learning/Prototyping → Python
Production Backend   → Go or Python
Web Integration      → TypeScript
Performance Tools    → Rust or Go
```

---

## Scaling to Larger Systems

### Is Multi-Agent Worth It?

**When YES:**
- Tasks are genuinely separable (planning vs execution)
- Different tools/permissions per role
- You need audit trails per agent
- Specialized prompts improve quality significantly

**When NO:**
- Single agent can handle it (simpler is better)
- Latency is critical (each agent = more LLM calls)
- Cost is a concern (2 agents = 2x+ token usage)
- Tasks are tightly coupled

### Coordination Patterns

#### 1. Sequential Pipeline
```
Agent A → Agent B → Agent C → Result
```
- Simple to reason about
- Each agent has clear input/output
- Failure handling is straightforward

#### 2. Supervisor Pattern
```
        ┌─── Agent A
Supervisor ├─── Agent B
        └─── Agent C
```
- Supervisor routes tasks
- Can retry with different agent
- More complex but flexible

#### 3. Collaborative (Hardest)
```
Agent A ←→ Agent B
   ↕         ↕
Agent C ←→ Agent D
```
- Agents communicate directly
- Very hard to debug
- Usually overkill

### What You Need for Larger Systems

| Capability | Why |
|------------|-----|
| **Message Queue** | Decouple agents, handle backpressure |
| **State Management** | Track workflow progress, enable resume |
| **Observability** | Trace requests across agents |
| **Rate Limiting** | Don't blow your API budget |
| **Circuit Breakers** | Handle LLM outages gracefully |

---

## Security: Secrets & Tokens

### The Problem

Tools often need secrets:
- API keys for external services
- Database credentials
- OAuth tokens for user actions

**The LLM should NEVER see these secrets.**

### Pattern 1: Server-Side Secret Injection

```
┌─────────┐     ┌─────────┐     ┌──────────┐
│   LLM   │────▶│  Agent  │────▶│   Tool   │
└─────────┘     └─────────┘     └────┬─────┘
                                     │
    LLM sees: "call github_api"      │ Tool injects
    LLM does NOT see: token          ▼ secret here
                                ┌──────────┐
                                │ GitHub   │
                                │ API      │
                                └──────────┘
```

```go
// Tool implementation - secret never in LLM context
func (t *GitHubTool) Execute(ctx context.Context, args map[string]interface{}) {
    token := os.Getenv("GITHUB_TOKEN")  // Injected at runtime
    // LLM only provided: repo, action
    // Token added server-side
}
```

### Pattern 2: Scoped Tokens

Give tools minimal permissions:

```
❌ Bad:  Tool has admin database access
✅ Good: Tool has read-only access to specific tables
```

### Pattern 3: User Context Tokens

For multi-tenant systems:

```go
type ToolContext struct {
    UserID      string
    Permissions []string
    Token       string  // User-specific, scoped token
}

func (t *Tool) Execute(ctx ToolContext, args map[string]interface{}) {
    // Use ctx.Token for this user's permissions
    // Validate ctx.Permissions before action
}
```

### What NOT to Do

```
❌ Include API keys in system prompt
❌ Let LLM generate authentication headers
❌ Store secrets in conversation memory
❌ Log full tool arguments (may contain secrets)
❌ Return secrets in tool results
```

### Token Usage & Cost Management

#### Tracking Token Usage

```go
type TokenTracker struct {
    InputTokens  int
    OutputTokens int
    Cost         float64
}

func (t *TokenTracker) Add(response LLMResponse) {
    t.InputTokens += response.Usage.InputTokens
    t.OutputTokens += response.Usage.OutputTokens
    t.Cost += calculateCost(response.Usage)
}
```

#### Cost Control Strategies

| Strategy | Implementation |
|----------|----------------|
| **Per-request limits** | Max tokens per LLM call |
| **Per-user limits** | Daily/monthly token budgets |
| **Circuit breaker** | Stop if cost exceeds threshold |
| **Caching** | Cache common tool results |
| **Prompt optimization** | Shorter prompts = fewer tokens |

#### Example: Budget Guard

```go
func (a *Agent) Run(input string) (*Result, error) {
    if a.tokenBudget.Remaining() < MIN_TOKENS {
        return nil, ErrBudgetExceeded
    }
    
    for iteration := 0; iteration < a.maxIterations; iteration++ {
        response, err := a.llm.Generate(messages)
        a.tokenBudget.Deduct(response.Usage)
        
        if a.tokenBudget.Remaining() < MIN_TOKENS {
            return nil, ErrBudgetExceeded
        }
        // ... continue loop
    }
}
```

---

## Production Checklist

### Before Going Live

- [ ] **Secrets**: No secrets in prompts or logs
- [ ] **Rate limits**: Per-user and global limits
- [ ] **Cost tracking**: Know your spend per request
- [ ] **Error handling**: Graceful degradation
- [ ] **Timeouts**: Don't wait forever for LLM
- [ ] **Logging**: Trace requests, but sanitize sensitive data
- [ ] **Monitoring**: Alert on error rates, latency, cost spikes
- [ ] **Testing**: Golden tests for critical prompts

### Observability Essentials

```
Request ID: abc-123
User: user@example.com
Agent: single-agent
Iterations: 3
Tool Calls: [calculator, calculator]
Input Tokens: 1,234
Output Tokens: 567
Cost: $0.02
Latency: 2.3s
Status: success
```

---

## Final Thoughts

Building agentic systems is an exercise in **controlled chaos**:

1. **You control**: Tools, prompts, orchestration, security
2. **You influence**: LLM behavior via prompts
3. **You can't control**: LLM reasoning, hallucinations, costs (fully)

The art is in designing systems that work well when the LLM behaves AND fail gracefully when it doesn't.

```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   Good Agentic System = Good Tools + Good Prompts +        │
│                         Good Error Handling +              │
│                         Good Observability                 │
│                                                            │
└────────────────────────────────────────────────────────────┘
```
