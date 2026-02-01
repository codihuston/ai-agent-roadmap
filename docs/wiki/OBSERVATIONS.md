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


---
## Memory & Context Management

### The Memory Problem

LLMs have a context window limit. As conversations grow:
- Tokens accumulate → costs increase
- Eventually hit context limit → conversation breaks
- Old context gets lost → agent "forgets"

### Memory Strategies

#### 1. Sliding Window (Simplest)
```
Keep last N messages, drop oldest

Messages: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
Window=5: [_, _, _, _, _, 6, 7, 8, 9, 10]
```
- Pros: Simple, predictable cost
- Cons: Loses important early context

#### 2. Summarization
```
┌─────────────────────────────────────────┐
│ Old Messages (1-50)                     │
│ "User asked about inventory system,     │
│  we discussed API design, decided on    │
│  REST with pagination..."               │
└─────────────────────────────────────────┘
            ↓ Summarize
┌─────────────────────────────────────────┐
│ Summary: "Designing REST API for        │
│ inventory with pagination"              │
└─────────────────────────────────────────┘
            + Recent Messages (51-60)
```
- Pros: Preserves key context, reduces tokens
- Cons: Summarization can lose details, requires LLM call

#### 3. RAG (Retrieval-Augmented Generation)
```
User: "What did we decide about the database?"

    ┌──────────────┐
    │   Embed      │
    │   Query      │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐     ┌─────────────────────┐
    │   Vector     │────▶│ Relevant chunks:    │
    │   Database   │     │ - "Decided PostgreSQL│
    └──────────────┘     │   for ACID compliance"│
                         │ - "Schema v2 approved"│
                         └─────────────────────┘
                                   │
                                   ▼
                         ┌─────────────────────┐
                         │   LLM + Context     │
                         └─────────────────────┘
```
- Pros: Scales to unlimited history, semantic search
- Cons: Complexity, embedding costs, retrieval quality

#### 4. Hierarchical Memory
```
┌─────────────────────────────────────────────────────────┐
│                    Long-Term Memory                      │
│  (Vector DB: all conversations, documents, decisions)   │
└─────────────────────────────────────────────────────────┘
                          ↑ Store
                          │ Retrieve
┌─────────────────────────────────────────────────────────┐
│                   Working Memory                         │
│  (Current session: last N messages + retrieved context) │
└─────────────────────────────────────────────────────────┘
                          ↑
                          │
┌─────────────────────────────────────────────────────────┐
│                   Scratchpad                             │
│  (Current task: intermediate results, tool outputs)     │
└─────────────────────────────────────────────────────────┘
```

### Memory Architecture for Complex Systems

```
┌─────────────────────────────────────────────────────────────────┐
│                        Memory Layer                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Episodic    │  │  Semantic    │  │  Procedural          │  │
│  │  Memory      │  │  Memory      │  │  Memory              │  │
│  │              │  │              │  │                      │  │
│  │  "What       │  │  "What do    │  │  "How to do things"  │  │
│  │  happened"   │  │  we know"    │  │                      │  │
│  │              │  │              │  │  - Tool usage        │  │
│  │  - Convos    │  │  - Facts     │  │  - Workflows         │  │
│  │  - Events    │  │  - Entities  │  │  - Patterns          │  │
│  │  - Decisions │  │  - Relations │  │                      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation Example

```go
type MemoryManager struct {
    shortTerm   *ConversationMemory  // Current session
    longTerm    VectorStore          // Persistent storage
    summarizer  LLMProvider          // For compression
}

func (m *MemoryManager) GetContext(query string, maxTokens int) []Message {
    // 1. Get recent messages (always included)
    recent := m.shortTerm.GetLast(10)
    
    // 2. Retrieve relevant historical context
    relevant := m.longTerm.Search(query, limit: 5)
    
    // 3. Build context within token budget
    context := []Message{}
    tokens := 0
    
    // Add system context first
    for _, doc := range relevant {
        if tokens + doc.Tokens < maxTokens/2 {
            context = append(context, doc.ToMessage())
            tokens += doc.Tokens
        }
    }
    
    // Add recent messages
    for _, msg := range recent {
        if tokens + msg.Tokens < maxTokens {
            context = append(context, msg)
            tokens += msg.Tokens
        }
    }
    
    return context
}

func (m *MemoryManager) Store(messages []Message) {
    // Store in short-term
    m.shortTerm.Add(messages)
    
    // Periodically persist to long-term
    if m.shortTerm.Len() > 50 {
        summary := m.summarizer.Summarize(m.shortTerm.GetOldest(40))
        m.longTerm.Store(summary)
        m.shortTerm.DropOldest(40)
    }
}
```

### Vector Database Options

| Database | Type | Best For |
|----------|------|----------|
| **Pinecone** | Managed | Production, scale |
| **Weaviate** | Self-hosted/Managed | Flexibility |
| **Chroma** | Embedded | Prototyping, local |
| **pgvector** | PostgreSQL extension | Already using Postgres |
| **Qdrant** | Self-hosted | Performance |

### Memory Dimensions for Complex Systems

```
┌─────────────────────────────────────────────────────────────────┐
│                     Memory Dimensions                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  TIME ──────────────────────────────────────────────────────▶   │
│  │ Session    │ Day      │ Week     │ Month    │ All-time │    │
│                                                                  │
│  SCOPE ─────────────────────────────────────────────────────▶   │
│  │ Message   │ Task     │ Project  │ User     │ Global   │     │
│                                                                  │
│  TYPE ──────────────────────────────────────────────────────▶   │
│  │ Fact      │ Decision │ Preference│ Skill   │ Relation │     │
│                                                                  │
│  AGENT ─────────────────────────────────────────────────────▶   │
│  │ Private   │ Shared   │ Global   │                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Observability & Traceability

### Why It Matters

Agentic systems are **non-deterministic**. The same input can produce different outputs. When something goes wrong, you need to understand:
- What did the LLM decide?
- Why did it choose that tool?
- What was in the context?
- Where did it go wrong?

### The Observability Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                     Observability Layers                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Metrics   │  │   Logging   │  │   Tracing               │  │
│  │             │  │             │  │                         │  │
│  │  - Latency  │  │  - Requests │  │  - Request flow         │  │
│  │  - Tokens   │  │  - Errors   │  │  - Agent handoffs       │  │
│  │  - Costs    │  │  - Tool use │  │  - Tool execution       │  │
│  │  - Errors   │  │  - LLM resp │  │  - LLM calls            │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Distributed Tracing for Agents

```
Trace ID: trace-abc-123
│
├─► Span: user-request (2.3s)
│   │ user_id: user@example.com
│   │ input: "Calculate quarterly revenue"
│   │
│   ├─► Span: agent-loop-iteration-1 (0.8s)
│   │   │
│   │   ├─► Span: llm-call (0.7s)
│   │   │   │ model: claude-3-sonnet
│   │   │   │ input_tokens: 1,234
│   │   │   │ output_tokens: 89
│   │   │   │ decision: tool_call
│   │   │   │ tool: database_query
│   │   │
│   │   └─► Span: tool-execution (0.1s)
│   │       │ tool: database_query
│   │       │ args: {table: "revenue", quarter: "Q3"}
│   │       │ result: success
│   │       │ rows: 150
│   │
│   ├─► Span: agent-loop-iteration-2 (0.9s)
│   │   │
│   │   ├─► Span: llm-call (0.85s)
│   │   │   │ model: claude-3-sonnet
│   │   │   │ input_tokens: 1,456
│   │   │   │ output_tokens: 234
│   │   │   │ decision: tool_call
│   │   │   │ tool: calculator
│   │   │
│   │   └─► Span: tool-execution (0.05s)
│   │       │ tool: calculator
│   │       │ args: {op: "sum", values: [...]}
│   │       │ result: 1,234,567
│   │
│   └─► Span: agent-loop-iteration-3 (0.6s)
│       │
│       └─► Span: llm-call (0.6s)
│           │ decision: final_response
│           │ response: "Q3 revenue was $1,234,567"
│
└─► Result: success
```

### Implementation: Trace Context

```go
type TraceContext struct {
    TraceID     string
    SpanID      string
    ParentSpan  string
    StartTime   time.Time
    Attributes  map[string]interface{}
}

type AgentTracer struct {
    exporter TraceExporter  // Jaeger, Zipkin, OTLP, etc.
}

func (t *AgentTracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
    span := &Span{
        TraceID:   getTraceID(ctx),
        SpanID:    generateSpanID(),
        ParentID:  getSpanID(ctx),
        Name:      name,
        StartTime: time.Now(),
    }
    return context.WithValue(ctx, spanKey, span), span
}

// Usage in agent loop
func (a *Agent) Run(ctx context.Context, input string) (*Result, error) {
    ctx, span := a.tracer.StartSpan(ctx, "agent-run")
    defer span.End()
    
    span.SetAttribute("input", input)
    span.SetAttribute("max_iterations", a.maxIterations)
    
    for i := 0; i < a.maxIterations; i++ {
        ctx, iterSpan := a.tracer.StartSpan(ctx, fmt.Sprintf("iteration-%d", i))
        
        // LLM call with tracing
        ctx, llmSpan := a.tracer.StartSpan(ctx, "llm-call")
        response, err := a.provider.Generate(ctx, request)
        llmSpan.SetAttribute("input_tokens", response.Usage.InputTokens)
        llmSpan.SetAttribute("output_tokens", response.Usage.OutputTokens)
        llmSpan.SetAttribute("has_tool_calls", len(response.ToolCalls) > 0)
        llmSpan.End()
        
        // Tool execution with tracing
        for _, tc := range response.ToolCalls {
            ctx, toolSpan := a.tracer.StartSpan(ctx, "tool-"+tc.Name)
            result := a.executeTool(ctx, tc)
            toolSpan.SetAttribute("success", result.Success)
            toolSpan.End()
        }
        
        iterSpan.End()
    }
    
    return result, nil
}
```

### Key Metrics to Track

```go
type AgentMetrics struct {
    // Latency
    RequestDuration    prometheus.Histogram
    LLMCallDuration    prometheus.Histogram
    ToolCallDuration   prometheus.Histogram
    
    // Volume
    RequestsTotal      prometheus.Counter
    ToolCallsTotal     prometheus.CounterVec  // by tool name
    IterationsTotal    prometheus.Histogram
    
    // Cost
    InputTokensTotal   prometheus.Counter
    OutputTokensTotal  prometheus.Counter
    EstimatedCost      prometheus.Counter
    
    // Errors
    ErrorsTotal        prometheus.CounterVec  // by error type
    ToolFailures       prometheus.CounterVec  // by tool name
    
    // Quality (if you can measure it)
    UserRatings        prometheus.Histogram
    TaskCompletionRate prometheus.Gauge
}
```

### Logging Best Practices

```go
// Structured logging with context
type LogEntry struct {
    Timestamp   time.Time              `json:"ts"`
    Level       string                 `json:"level"`
    TraceID     string                 `json:"trace_id"`
    SpanID      string                 `json:"span_id"`
    Component   string                 `json:"component"`
    Event       string                 `json:"event"`
    Attributes  map[string]interface{} `json:"attrs"`
}

// Example log entries
{"ts":"2024-01-15T10:30:00Z","level":"info","trace_id":"abc123","component":"agent","event":"request_start","attrs":{"user":"user@example.com","input_length":45}}

{"ts":"2024-01-15T10:30:01Z","level":"info","trace_id":"abc123","component":"llm","event":"call_complete","attrs":{"model":"claude-3","input_tokens":1234,"output_tokens":89,"tool_calls":["calculator"]}}

{"ts":"2024-01-15T10:30:01Z","level":"info","trace_id":"abc123","component":"tool","event":"execution","attrs":{"tool":"calculator","duration_ms":50,"success":true}}

{"ts":"2024-01-15T10:30:02Z","level":"info","trace_id":"abc123","component":"agent","event":"request_complete","attrs":{"iterations":2,"total_tokens":1500,"duration_ms":2300}}
```

### What to Log (and What NOT to)

```
✅ DO LOG:
- Request/trace IDs
- User IDs (for debugging)
- Tool names and success/failure
- Token counts
- Latencies
- Error types and messages
- Agent decisions (tool calls vs final response)

❌ DO NOT LOG:
- Full prompt content (may contain PII)
- API keys or secrets
- Full tool arguments (may contain sensitive data)
- Full LLM responses (token cost, storage cost)
- User input verbatim (PII risk)

⚠️ LOG CAREFULLY (sanitize first):
- Summarized user intent
- Tool argument keys (not values)
- Error details (redact sensitive info)
```

### Debugging Workflow

```
1. User reports: "Agent gave wrong answer"
   │
   ▼
2. Find trace ID from user session
   │
   ▼
3. Load full trace
   │
   ├─► Check: What tools were called?
   │   └─► Were the right tools selected?
   │
   ├─► Check: What was in the context?
   │   └─► Was relevant info retrieved?
   │
   ├─► Check: What did LLM decide?
   │   └─► Was the reasoning sound?
   │
   └─► Check: Did tools return correct data?
       └─► Was there a tool bug?
```

### Observability Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| **Jaeger/Zipkin** | Distributed tracing | Self-hosted |
| **Datadog/New Relic** | Full observability | Managed, $$$ |
| **Prometheus + Grafana** | Metrics | Self-hosted, free |
| **OpenTelemetry** | Standard instrumentation | Vendor-neutral |
| **LangSmith** | LLM-specific tracing | LangChain ecosystem |
| **Weights & Biases** | ML experiment tracking | Good for evals |

### Dashboard Essentials

```
┌─────────────────────────────────────────────────────────────────┐
│                    Agent Dashboard                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Requests/min    │  │ Avg Latency     │  │ Error Rate      │  │
│  │     127         │  │    2.3s         │  │    0.5%         │  │
│  │   ▲ 12%         │  │   ▼ 5%          │  │   ▼ 0.1%        │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Tokens Today    │  │ Cost Today      │  │ Avg Iterations  │  │
│  │   1.2M          │  │    $45.67       │  │    2.3          │  │
│  │   ▲ 8%          │  │   ▲ 8%          │  │   ─ 0%          │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Tool Usage (last 24h)                                     │   │
│  │ calculator ████████████████████ 45%                       │   │
│  │ database   ██████████████ 32%                             │   │
│  │ file_read  ████████ 18%                                   │   │
│  │ api_call   ██ 5%                                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Recent Errors                                             │   │
│  │ 10:30 tool_timeout  database_query  trace-abc123          │   │
│  │ 10:28 llm_error     rate_limited    trace-def456          │   │
│  │ 10:15 tool_error    api_call        trace-ghi789          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Alerting Rules

```yaml
# Example Prometheus alerting rules
groups:
  - name: agent-alerts
    rules:
      - alert: HighErrorRate
        expr: rate(agent_errors_total[5m]) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Agent error rate above 5%"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95, agent_request_duration_seconds) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P95 latency above 10 seconds"
          
      - alert: CostSpike
        expr: increase(agent_cost_dollars[1h]) > 100
        labels:
          severity: warning
        annotations:
          summary: "Hourly cost exceeded $100"
          
      - alert: LLMRateLimited
        expr: rate(llm_rate_limit_errors[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "LLM API rate limited"
```
