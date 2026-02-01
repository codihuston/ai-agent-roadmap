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


---

## Agentic Orchestration Patterns

Different orchestration patterns suit different use cases. Here's a breakdown of patterns with real-world product examples.

### 1. Sequential Pipeline

```
Agent A → Agent B → Agent C → Result
```

Each agent has a distinct phase. Simple, predictable, easy to debug.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Code generation | Cursor, GitHub Copilot Workspace | Plan → Code → Review flow |
| Content creation | Jasper, Copy.ai | Research → Draft → Edit → Polish |
| Document processing | DocuSign IAM, Eigen Technologies | Extract → Validate → Transform |
| CI/CD pipelines | Harness AI, Octopus Deploy | Build → Test → Deploy → Verify |

**When to use**: Clear sequential phases, each step depends on previous output.

### 2. Router/Dispatcher

```
         ┌→ Specialist A
Input → Router → Specialist B → Output
         └→ Specialist C
```

A classifier routes requests to specialized agents.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Customer support | Intercom Fin, Zendesk AI | Route to billing/technical/sales specialists |
| Multi-domain assistants | ChatGPT with plugins, Claude with tools | Route to code/search/math capabilities |
| Enterprise search | Glean, Coveo | Route to HR/IT/Finance knowledge bases |
| Healthcare triage | Ada Health, Babylon | Route symptoms to appropriate specialty |

**When to use**: Multiple specialized domains, need to classify intent first.

### 3. Supervisor/Worker

```
        ┌─ Worker A
Supervisor ├─ Worker B  → Supervisor aggregates
        └─ Worker C
```

Supervisor decomposes tasks, delegates, and synthesizes results.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Research & analysis | Perplexity, Elicit | Break research into sub-queries, synthesize |
| Complex coding tasks | Devin, OpenHands | Decompose feature into subtasks |
| Data analysis | Julius AI, Akkio | Split analysis across data sources |
| Report generation | Tome, Gamma | Supervisor coordinates section writers |

**When to use**: Complex tasks that can be decomposed, need aggregation of multiple results.

### 4. Debate/Adversarial

```
Agent A (propose) ←→ Agent B (critique) → Consensus
```

One proposes, another critiques. Iterate until agreement.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Code review | CodeRabbit, Codium PR-Agent | Generator vs reviewer |
| Fact-checking | Factiverse, ClaimBuster | Claim maker vs fact checker |
| Legal analysis | Harvey, CoCounsel | Argument vs counter-argument |
| Security review | Snyk AI, Semgrep | Code generator vs security auditor |

**When to use**: High-stakes decisions, need verification, adversarial robustness.

### 5. Hierarchical

```
Executive Agent
    ├─ Manager A → Workers
    └─ Manager B → Workers
```

Multiple delegation layers. Executives strategize, managers coordinate, workers execute.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Autonomous software dev | Devin, Factory AI | PM → Tech Lead → Developers |
| Marketing campaigns | Pencil, Smartly.io | Strategy → Creative → Execution |
| Game AI | Unity ML-Agents, NVIDIA ACE | Director → Scene managers → NPCs |
| Robotics | Boston Dynamics, Figure AI | Mission planner → Task coordinators → Actuators |

**When to use**: Large-scale projects, need organizational structure, different abstraction levels.

### 6. Swarm/Collaborative

```
Agent A ←→ Agent B
   ↕         ↕
Agent C ←→ Agent D
```

Peer-to-peer communication, emergent behavior, no central coordinator.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Multi-agent simulation | AutoGen, CrewAI | Research collaboration scenarios |
| Game NPCs | AI Dungeon, Character.ai (multi-char) | Characters interact naturally |
| Trading systems | QuantConnect, Alpaca | Multiple strategies negotiate |
| Scientific discovery | Coscientist, ChemCrow | Agents collaborate on hypotheses |

**When to use**: Emergent behavior desired, simulation, research. Hard to debug - use cautiously.

### 7. Map-Reduce

```
        ┌→ Agent (chunk 1) ─┐
Input → ├→ Agent (chunk 2) ─┼→ Aggregator → Output
        └→ Agent (chunk 3) ─┘
```

Parallelize across identical agents, then combine results.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Document analysis | Docugami, Instabase | Process pages in parallel |
| Codebase analysis | Sourcegraph Cody, Tabnine | Analyze files in parallel |
| Data extraction | Scale AI, Labelbox | Parallel annotation/extraction |
| Translation | DeepL, Smartling | Translate sections in parallel |

**When to use**: Large inputs that can be chunked, embarrassingly parallel workloads.

### 8. Reflection/Self-Critique

```
Agent → Output → Same Agent (critique mode) → Refined Output
```

Single agent reviews its own work with a different prompt/persona.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Writing improvement | Grammarly, ProWritingAid | Draft → Self-edit |
| Code refinement | Cursor (with review), Aider | Generate → Self-review |
| Math/reasoning | OpenAI o1, Claude thinking | Solve → Verify steps |
| Image generation | Midjourney (variations), DALL-E | Generate → Self-critique → Refine |

**When to use**: Quality improvement without multi-agent complexity, cheaper than debate pattern.

### 9. Plan-Execute-Verify

```
Planner → Executor → Verifier ─┐
    ↑                          │
    └──────── (retry) ─────────┘
```

Explicit verification with feedback loop for retry.

| Use Case | Example Products | Why This Pattern |
|----------|------------------|------------------|
| Autonomous browsing | MultiOn, Adept | Plan actions → Execute → Verify page state |
| Test generation | Testim, Mabl | Plan tests → Run → Verify results |
| Infrastructure | Pulumi AI, Terraform AI | Plan changes → Apply → Verify state |
| Data pipelines | Fivetran, Airbyte | Plan sync → Execute → Validate data |

**When to use**: Correctness critical, need explicit verification, can afford retries.

---

### Choosing the Right Pattern

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     Pattern Selection Guide                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Simple, linear workflow?          → Sequential Pipeline                 │
│  Multiple specialized domains?     → Router/Dispatcher                   │
│  Complex decomposable task?        → Supervisor/Worker                   │
│  High-stakes, need verification?   → Debate/Adversarial                  │
│  Large org-like structure?         → Hierarchical                        │
│  Emergent behavior needed?         → Swarm (careful!)                    │
│  Large parallelizable input?       → Map-Reduce                          │
│  Quality boost, low complexity?    → Reflection/Self-Critique            │
│  Must be correct, can retry?       → Plan-Execute-Verify                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Hybrid Approaches

Real systems often combine patterns:

- **Cursor**: Router (intent) → Sequential (plan/code/review) → Reflection (self-edit)
- **Devin**: Hierarchical (PM/dev) → Plan-Execute-Verify (coding loop)
- **Perplexity**: Router (query type) → Map-Reduce (search) → Supervisor (synthesize)

Start simple (sequential or reflection), add complexity only when needed.


---

## Underaddressed Problem Spaces in Agentic AI

These are gaps in the current ecosystem - problems that aren't well-solved yet, with analysis of what's in your control vs outside your domain.

### 1. Long-Running Task Reliability

**The Problem**: Agents fail mid-task. Network drops, LLM rate limits, tool errors. For a 30-minute autonomous task, failure probability compounds.

| Scale | Challenge |
|-------|-----------|
| Single request | Retry logic, timeout handling |
| Multi-step task | Checkpointing, resume from failure |
| Hours-long workflow | State persistence, partial rollback |
| Days/weeks | Human handoff, progress reporting |

**What You Can Control**:
- Checkpointing between steps (save state to disk/DB)
- Idempotent tool operations (safe to retry)
- Graceful degradation (partial results > no results)
- Progress persistence and resume logic

**Outside Your Domain**:
- LLM API reliability (you can only retry/failover)
- Network stability
- Rate limit policies

**Approach**: Build a task state machine with persistent checkpoints. Every tool call should be resumable. Think of it like a database transaction log.

```go
type TaskCheckpoint struct {
    TaskID      string
    Step        int
    State       map[string]interface{}
    Completed   []string  // completed tool calls
    Pending     []string  // remaining work
    CreatedAt   time.Time
}
```

---

### 2. Cost Predictability & Budgeting

**The Problem**: Token costs are unpredictable. A "simple" task might spiral into 50 LLM calls. Users/businesses need cost guarantees.

| Scale | Challenge |
|-------|-----------|
| Single query | Estimate before execution |
| Session | Budget caps per conversation |
| User/tenant | Monthly quotas, alerts |
| Enterprise | Chargeback, cost allocation |

**What You Can Control**:
- Pre-flight cost estimation (count tokens before sending)
- Hard budget limits with circuit breakers
- Cost-aware routing (cheaper model for simple tasks)
- Caching identical/similar requests
- Prompt optimization (shorter = cheaper)

**Outside Your Domain**:
- API pricing changes
- Token counting accuracy (varies by model)
- Quality vs cost tradeoff (cheaper models = worse results)

**Approach**: Build a cost governor into your orchestrator.

```go
type CostGovernor struct {
    BudgetRemaining float64
    EstimatedCost   func(req GenerateRequest) float64
    OnBudgetLow     func(remaining float64)
}

func (g *CostGovernor) CanProceed(req GenerateRequest) bool {
    estimated := g.EstimatedCost(req)
    return estimated < g.BudgetRemaining
}
```

---

### 3. Determinism & Reproducibility

**The Problem**: Same input → different output. Makes testing, debugging, and auditing nearly impossible.

| Scale | Challenge |
|-------|-----------|
| Development | Can't write reliable tests |
| Debugging | Can't reproduce user issues |
| Compliance | Can't prove what happened |
| Legal/audit | Need exact replay capability |

**What You Can Control**:
- Seed parameters (where supported)
- Temperature = 0 (reduces but doesn't eliminate variance)
- Full request/response logging
- Snapshot exact prompts and context
- Version control system prompts

**Outside Your Domain**:
- LLM internal randomness
- Model updates/changes by provider
- Non-deterministic tool outputs (live APIs, time-based data)

**Approach**: Accept non-determinism, design around it.

- Log everything needed to understand decisions (not reproduce exactly)
- Test behavior ranges, not exact outputs
- Use property-based tests for invariants
- Build "close enough" comparison for outputs

---

### 4. Multi-Tenant Isolation & Security

**The Problem**: Agents have real power (file access, API calls, code execution). In multi-tenant systems, one user's agent shouldn't affect another's.

| Scale | Challenge |
|-------|-----------|
| Single user | Sandbox tool execution |
| Multi-user | Isolate contexts, prevent data leakage |
| Enterprise | Tenant-specific tools, permissions |
| Platform | Malicious prompt injection across tenants |

**What You Can Control**:
- Tool permission scoping per user/tenant
- Sandboxed execution environments (containers, VMs)
- Context isolation (separate memory per tenant)
- Input sanitization before tool execution
- Output filtering before returning to user

**Outside Your Domain**:
- LLM-level prompt injection vulnerabilities
- Zero-day exploits in sandbox tech
- Provider-side data handling

**Approach**: Defense in depth.

```go
type TenantContext struct {
    TenantID    string
    Permissions []Permission
    Sandbox     SandboxConfig
    Tools       []Tool  // tenant-specific tool subset
}

func (t *Tool) Execute(ctx TenantContext, args map[string]interface{}) (Result, error) {
    if !ctx.HasPermission(t.RequiredPermission) {
        return Result{}, ErrPermissionDenied
    }
    // Execute in isolated sandbox
    return t.sandbox.Run(ctx.Sandbox, args)
}
```

---

### 5. Human-in-the-Loop at Scale

**The Problem**: Fully autonomous agents make mistakes. Humans need to intervene, but intervention doesn't scale.

| Scale | Challenge |
|-------|-----------|
| Single task | Simple approve/reject |
| Batch tasks | Prioritize which need review |
| High volume | Sample-based review, escalation rules |
| Enterprise | Workflow integration, SLAs |

**What You Can Control**:
- Confidence scoring (low confidence → human review)
- Escalation rules (certain actions always need approval)
- Async approval workflows
- Batch review interfaces
- Audit trails for accountability

**Outside Your Domain**:
- Human availability and response time
- Organizational approval processes
- Regulatory requirements for human oversight

**Approach**: Build confidence-based escalation.

```go
type EscalationPolicy struct {
    AlwaysEscalate  []string  // tool names that always need approval
    ConfidenceThreshold float64
    MaxAutoApprovals    int    // after N auto-approvals, force human review
}

func (p *EscalationPolicy) NeedsHumanReview(action ToolCall, confidence float64) bool {
    if slices.Contains(p.AlwaysEscalate, action.Name) {
        return true
    }
    return confidence < p.ConfidenceThreshold
}
```

---

### 6. Observability for Non-Deterministic Systems

**The Problem**: Traditional observability assumes deterministic systems. "Request X always produces Y." Agents don't work that way.

| Scale | Challenge |
|-------|-----------|
| Single request | Why did it choose that tool? |
| Session | Why did behavior change mid-conversation? |
| Aggregate | What's "normal" when every request is different? |
| Debugging | Reproduce and understand failures |

**What You Can Control**:
- Decision logging (what options, why chosen)
- Context snapshots at each step
- Tool call traces with full arguments
- LLM response logging (with PII scrubbing)
- Anomaly detection on behavioral patterns

**Outside Your Domain**:
- LLM internal reasoning (black box)
- Why the model "changed its mind"
- Explaining emergent multi-agent behavior

**Approach**: Log decisions, not just actions.

```go
type DecisionLog struct {
    Timestamp   time.Time
    Context     string   // summarized context
    Options     []string // what tools/actions were available
    Chosen      string   // what was selected
    Confidence  float64  // if available
    Reasoning   string   // LLM's stated reasoning (if using CoT)
}
```

---

### 7. Graceful Degradation & Fallbacks

**The Problem**: When the primary LLM fails or is slow, what happens? Most systems just error out.

| Scale | Challenge |
|-------|-----------|
| Single failure | Retry with backoff |
| Provider outage | Failover to alternate provider |
| Degraded quality | Use cheaper/faster model as fallback |
| Complete failure | Meaningful error to user |

**What You Can Control**:
- Multi-provider support (Claude, GPT, local models)
- Automatic failover logic
- Quality-aware routing (try best model first, fall back)
- Cached responses for common queries
- Graceful error messages

**Outside Your Domain**:
- Provider availability
- Model quality differences
- API compatibility across providers

**Approach**: Provider abstraction with fallback chain.

```go
type FallbackChain struct {
    Providers []LLMProvider  // ordered by preference
    Timeout   time.Duration
}

func (f *FallbackChain) Generate(ctx context.Context, req GenerateRequest) (*LLMResponse, error) {
    var lastErr error
    for _, provider := range f.Providers {
        ctx, cancel := context.WithTimeout(ctx, f.Timeout)
        resp, err := provider.Generate(ctx, req)
        cancel()
        if err == nil {
            return resp, nil
        }
        lastErr = err
        log.Warn("provider failed, trying next", "provider", provider.Name(), "error", err)
    }
    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
```

---

### 8. Context Window Management

**The Problem**: Context windows are finite. Long conversations, large codebases, extensive tool outputs - they all compete for limited space.

| Scale | Challenge |
|-------|-----------|
| Single conversation | Fit history in window |
| Large documents | Chunk and retrieve relevant parts |
| Codebase-wide | Can't fit entire repo in context |
| Multi-session | Maintain continuity across sessions |

**What You Can Control**:
- Smart summarization of old context
- RAG for selective retrieval
- Chunking strategies for large inputs
- Priority-based context inclusion
- Token budgeting across context components

**Outside Your Domain**:
- Context window size limits
- Quality degradation with summarization
- Retrieval accuracy

**Approach**: Context budget allocation.

```go
type ContextBudget struct {
    TotalTokens     int
    SystemPrompt    int  // reserved
    RecentHistory   int  // reserved for last N messages
    RetrievedContext int // for RAG results
    CurrentInput    int  // user's current message
}

func (b *ContextBudget) Allocate(components []ContextComponent) []ContextComponent {
    // Prioritize: system > current input > recent > retrieved
    // Truncate lower priority items if over budget
}
```

---

### 9. Evaluation & Quality Measurement

**The Problem**: How do you know if your agent is "good"? Traditional metrics don't apply. User satisfaction is subjective and delayed.

| Scale | Challenge |
|-------|-----------|
| Single response | Was this helpful? |
| Task completion | Did it achieve the goal? |
| Aggregate quality | Is the system improving over time? |
| Comparison | Is prompt A better than prompt B? |

**What You Can Control**:
- Task completion tracking (did tool calls succeed?)
- User feedback collection (thumbs up/down, ratings)
- A/B testing infrastructure
- Golden test suites for regression
- LLM-as-judge for automated evaluation

**Outside Your Domain**:
- Subjective quality perception
- Long-term impact measurement
- Ground truth for open-ended tasks

**Approach**: Multi-signal quality tracking.

```go
type QualitySignals struct {
    TaskCompleted    bool
    ToolCallsSucceeded int
    ToolCallsFailed    int
    Iterations       int
    UserRating       *int     // nil if not provided
    LLMJudgeScore    *float64 // automated evaluation
    Latency          time.Duration
    TokensUsed       int
}
```

---

### 10. Agent Identity & Continuity

**The Problem**: Agents are stateless by default. Each conversation starts fresh. Users expect continuity - "remember what we discussed yesterday."

| Scale | Challenge |
|-------|-----------|
| Single session | Maintain context within conversation |
| Cross-session | Remember user preferences, past decisions |
| Long-term | Build up knowledge about user/project |
| Multi-agent | Share relevant context between agents |

**What You Can Control**:
- User profile storage (preferences, history)
- Project-level memory (decisions, context)
- Cross-session retrieval (RAG on past conversations)
- Agent "personality" persistence

**Outside Your Domain**:
- True understanding/memory (it's retrieval, not memory)
- Privacy regulations on data retention
- Storage costs at scale

**Approach**: Layered memory with explicit persistence.

```go
type AgentMemory struct {
    Session    *ConversationMemory  // current conversation
    User       *UserProfile         // preferences, history
    Project    *ProjectContext      // decisions, codebase knowledge
    Global     *KnowledgeBase       // shared across all users
}

func (m *AgentMemory) GetRelevantContext(query string) []Message {
    // Combine from all layers, prioritize by relevance
}
```

---

### Summary: What's In Your Control

| Problem Space | Your Control | Outside Domain |
|---------------|--------------|----------------|
| Long-running reliability | Checkpointing, retry, idempotency | LLM/network reliability |
| Cost predictability | Budgets, estimation, caching | API pricing |
| Determinism | Logging, testing strategies | LLM randomness |
| Multi-tenant security | Sandboxing, permissions | LLM vulnerabilities |
| Human-in-the-loop | Escalation rules, workflows | Human availability |
| Observability | Decision logging, tracing | LLM reasoning |
| Graceful degradation | Fallbacks, multi-provider | Provider availability |
| Context management | Summarization, RAG, budgeting | Window size limits |
| Quality measurement | Metrics, A/B testing, feedback | Subjective quality |
| Agent continuity | Persistent memory, retrieval | True understanding |

**The pattern**: You control the plumbing, orchestration, and data flow. The LLM itself is a black box you work around, not through.

Focus your engineering effort on the "Your Control" column - that's where you can differentiate and add value.


---

## Emerging CS Fundamentals for Agentic Systems

Classic computer science concepts are being adapted and new patterns are emerging specifically for agentic AI. These are becoming foundational knowledge for building robust agent systems.

### Data Structures

#### 1. Vector Indexes (ANN - Approximate Nearest Neighbor)

Not new, but now essential infrastructure. Powers semantic search and RAG.

```
Query: "How do I deploy to production?"
         │
         ▼ Embed
    [0.12, -0.34, 0.56, ...]
         │
         ▼ ANN Search
    ┌─────────────────────────────────────┐
    │  Vector Index (HNSW, IVF, etc.)     │
    │  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐    │
    │  │ • │─│ • │─│ • │─│ • │─│ • │    │
    │  └───┘ └───┘ └─┬─┘ └───┘ └───┘    │
    │                │ ← nearest         │
    └────────────────┼───────────────────┘
                     ▼
    "Deployment docs section 3.2..."
```

**Key Algorithms**:
- **HNSW (Hierarchical Navigable Small World)**: Multi-layer graph, O(log n) search. Best general-purpose.
- **IVF (Inverted File Index)**: Cluster-based, good for very large datasets.
- **PQ (Product Quantization)**: Compression for memory efficiency.

**When to use**: Any time you need "find similar" - context retrieval, memory search, deduplication.

#### 2. Merkle DAGs for Agent State

Git-like structures for tracking agent decision trees. Each node is content-addressed (hash of contents).

```
                    ┌─────────────────┐
                    │ State: abc123   │
                    │ "User asked X"  │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              ▼                              ▼
    ┌─────────────────┐            ┌─────────────────┐
    │ State: def456   │            │ State: ghi789   │
    │ "Called tool A" │            │ "Called tool B" │
    │ (branch 1)      │            │ (branch 2)      │
    └────────┬────────┘            └─────────────────┘
             │
             ▼
    ┌─────────────────┐
    │ State: jkl012   │
    │ "Got result"    │
    └─────────────────┘
```

**Benefits**:
- Immutable history (can't tamper with past decisions)
- Efficient branching (explore alternatives)
- Easy rollback (just point to earlier hash)
- Deduplication (same state = same hash)

**When to use**: Complex planning, speculative execution, audit trails.

#### 3. Persistent Data Structures

Immutable structures that share unchanged parts between versions. Efficient state snapshots.

```
Version 1:        Version 2 (after update):
    ┌───┐             ┌───┐
    │ A │             │ A │ ← shared
    ├───┤             ├───┤
    │ B │             │ B │ ← shared
    ├───┤             ├───┤
    │ C │             │ C'│ ← new (only this changed)
    └───┘             └───┘
    
Memory: Only C' is new allocation
```

**Implementations**:
- Persistent vectors (Clojure-style, 32-way tries)
- Persistent hash maps (HAMT - Hash Array Mapped Trie)
- Immutable.js, Immer (JS), pyrsistent (Python)

**When to use**: Checkpointing agent state, undo/redo, concurrent access without locks.

#### 4. Bloom Filters for Context Deduplication

Probabilistic structure: "definitely not seen" or "probably seen." Space-efficient.

```go
type ContextDeduplicator struct {
    filter *bloom.BloomFilter
}

func (d *ContextDeduplicator) ShouldInclude(content string) bool {
    hash := hashContent(content)
    if d.filter.Test(hash) {
        return false  // probably already in context
    }
    d.filter.Add(hash)
    return true  // definitely new
}
```

**Properties**:
- False positives possible (says "seen" when not)
- False negatives impossible (never says "not seen" when it was)
- Very space efficient (bits, not full content)

**When to use**: Deduplicating retrieved context, avoiding repeated tool calls, cycle detection.

#### 5. Ring Buffers for Sliding Window Memory

Fixed-size circular buffer. O(1) add, O(1) evict oldest.

```
Capacity: 5

Add messages: A, B, C, D, E
┌───┬───┬───┬───┬───┐
│ A │ B │ C │ D │ E │
└───┴───┴───┴───┴───┘
  ↑                 ↑
  tail              head

Add F (overwrites A):
┌───┬───┬───┬───┬───┐
│ F │ B │ C │ D │ E │
└───┴───┴───┴───┴───┘
      ↑           ↑
      tail        head
```

```go
type MessageRingBuffer struct {
    messages []Message
    head     int
    tail     int
    size     int
    capacity int
}

func (r *MessageRingBuffer) Add(msg Message) {
    r.messages[r.head] = msg
    r.head = (r.head + 1) % r.capacity
    if r.size == r.capacity {
        r.tail = (r.tail + 1) % r.capacity
    } else {
        r.size++
    }
}
```

**When to use**: Fixed-size conversation history, recent context window, streaming data.

---

### Algorithms

#### 1. Monte Carlo Tree Search (MCTS)

From game AI (AlphaGo), now used for agent planning. Explore action trees probabilistically.

```
                    ┌─────────────┐
                    │ Current     │
                    │ State       │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    ┌─────────┐       ┌─────────┐       ┌─────────┐
    │ Action A│       │ Action B│       │ Action C│
    │ Win: 3/10│      │ Win: 7/10│      │ Win: 2/10│
    └────┬────┘       └────┬────┘       └─────────┘
         │                 │
    ┌────┴────┐       ┌────┴────┐
    ▼         ▼       ▼         ▼
  ┌───┐     ┌───┐   ┌───┐     ┌───┐
  │2/5│     │1/5│   │4/5│     │3/5│
  └───┘     └───┘   └───┘     └───┘
```

**Four phases**:
1. **Selection**: Walk tree using UCB1 (balance exploration/exploitation)
2. **Expansion**: Add new child node for unexplored action
3. **Simulation**: Random rollout to terminal state
4. **Backpropagation**: Update win/visit counts up the tree

**For agents**:
- Actions = tool calls or response options
- Simulation = LLM predicts outcome (or cheap heuristic)
- Used in OpenAI o1-style reasoning

**When to use**: Complex planning with many options, when you can simulate outcomes.

#### 2. Beam Search with Pruning

Generate multiple candidate paths, keep top K, prune rest.

```
Step 1:     Step 2:           Step 3:
            
   A ──┬── A1 (0.9) ──┬── A1a (0.85) ✓ keep
       │              └── A1b (0.70) ✗ prune
       ├── A2 (0.7) ──┬── A2a (0.80) ✓ keep
       │              └── A2b (0.60) ✗ prune
       └── A3 (0.5) ✗ prune early

Beam width K=2: Keep top 2 at each step
```

```go
type BeamSearch struct {
    BeamWidth int
    ScoreFunc func(path []Action) float64
}

func (b *BeamSearch) Search(initial State) []Action {
    beams := [][]Action{{}}
    
    for step := 0; step < maxSteps; step++ {
        candidates := [][]Action{}
        for _, path := range beams {
            for _, action := range possibleActions(path) {
                candidates = append(candidates, append(path, action))
            }
        }
        // Sort by score, keep top K
        sort.Slice(candidates, func(i, j int) bool {
            return b.ScoreFunc(candidates[i]) > b.ScoreFunc(candidates[j])
        })
        beams = candidates[:min(b.BeamWidth, len(candidates))]
    }
    return beams[0]  // best path
}
```

**When to use**: Plan generation, when greedy is too short-sighted but full search is too expensive.

#### 3. A* / Graph Search for Tool Selection

Model tool chains as a graph, find optimal path to goal.

```
Start: "User wants to analyze sales data"

    ┌──────────────┐
    │    START     │
    └──────┬───────┘
           │
    ┌──────┴───────┐
    ▼              ▼
┌────────┐    ┌────────┐
│ query  │    │ read   │
│ db     │    │ file   │
│ cost:2 │    │ cost:1 │
└───┬────┘    └───┬────┘
    │             │
    └──────┬──────┘
           ▼
    ┌────────────┐
    │ calculator │
    │ cost: 1    │
    └─────┬──────┘
          │
          ▼
    ┌────────────┐
    │   GOAL     │
    │ "analysis" │
    └────────────┘

A* finds: read_file → calculator (cost: 2)
vs: query_db → calculator (cost: 3)
```

**Heuristic function**: Estimate remaining cost to goal (e.g., "how many more tools needed?")

**When to use**: Tool chain optimization, finding efficient paths through capability space.

#### 4. Semantic Caching with LSH

Locality-Sensitive Hashing: Similar inputs hash to same bucket. Enables "fuzzy" cache lookups.

```
Query 1: "What's the weather in Seattle?"
Query 2: "Seattle weather today?"
Query 3: "How's the weather in Seattle?"

Traditional cache: 3 misses (exact match fails)
LSH cache: 1 miss, 2 hits (semantically similar)

┌─────────────────────────────────────────────┐
│              LSH Hash Buckets               │
├─────────────────────────────────────────────┤
│ Bucket 0x3F2A: [weather queries...]         │
│ Bucket 0x7B1C: [code questions...]          │
│ Bucket 0x2D4E: [math problems...]           │
└─────────────────────────────────────────────┘
```

```go
type SemanticCache struct {
    lsh       *LSHIndex
    cache     map[string]CachedResponse
    threshold float64
}

func (c *SemanticCache) Get(query string) (*CachedResponse, bool) {
    embedding := embed(query)
    candidates := c.lsh.Query(embedding)
    
    for _, candidate := range candidates {
        if cosineSimilarity(embedding, candidate.Embedding) > c.threshold {
            return candidate.Response, true
        }
    }
    return nil, false
}
```

**When to use**: Reducing LLM calls for similar queries, cost optimization.

#### 5. UCB1 (Upper Confidence Bound) for Action Selection

Balance exploration vs exploitation when choosing actions.

```
UCB1 = average_reward + C * sqrt(ln(total_trials) / action_trials)
       └─────┬─────┘   └──────────────┬──────────────────────┘
         exploit                    explore
         (known good)               (uncertainty bonus)
```

```go
func (a *ActionSelector) SelectAction(actions []Action) Action {
    totalTrials := a.getTotalTrials()
    
    bestScore := -1.0
    var bestAction Action
    
    for _, action := range actions {
        trials := a.getTrials(action)
        if trials == 0 {
            return action  // always try untried actions
        }
        
        avgReward := a.getAverageReward(action)
        exploration := math.Sqrt(math.Log(float64(totalTrials)) / float64(trials))
        ucb := avgReward + a.C * exploration
        
        if ucb > bestScore {
            bestScore = ucb
            bestAction = action
        }
    }
    return bestAction
}
```

**When to use**: Tool selection when you have feedback, A/B testing prompts, adaptive routing.

---

### Emerging Patterns

#### 1. Chain-of-Thought as Structured Data

Treat reasoning traces as parseable data, not just text.

```go
type ThoughtChain struct {
    Steps []ThoughtStep `json:"steps"`
}

type ThoughtStep struct {
    Type       string `json:"type"`  // "observation", "reasoning", "conclusion"
    Content    string `json:"content"`
    Confidence float64 `json:"confidence,omitempty"`
    References []string `json:"references,omitempty"`
}

// Parse from LLM output
func ParseThoughtChain(llmOutput string) (*ThoughtChain, error) {
    // Extract structured reasoning from <thinking> tags or JSON
}

// Validate reasoning
func (tc *ThoughtChain) Validate() error {
    // Check logical consistency
    // Verify references exist
    // Flag low-confidence steps
}
```

**Benefits**:
- Can validate reasoning before acting
- Can branch on specific steps
- Enables reasoning about reasoning

#### 2. Constrained Decoding / Tool Use Grammars

Force LLM output to conform to a grammar. Guarantees valid structure.

```
Grammar for tool calls:

tool_call    := "{" "name" ":" tool_name "," "args" ":" args_obj "}"
tool_name    := "\"" ("calculator" | "read_file" | "search") "\""
args_obj     := "{" (arg_pair ("," arg_pair)*)? "}"
arg_pair     := "\"" identifier "\"" ":" value

LLM can ONLY output strings matching this grammar.
No more "I'll call the calculator" without actually calling it.
```

**Implementations**:
- Outlines (Python)
- Guidance (Microsoft)
- LMQL
- llama.cpp grammar sampling

**When to use**: When you need guaranteed parseable output, tool calling, structured extraction.

#### 3. Speculative Execution for Agents

Run multiple possible next steps in parallel, discard unused.

```
User: "What's the weather and my calendar for tomorrow?"

Sequential:                    Speculative:
                              
weather_api() ─┐               weather_api() ──┐
               │ 2s                             ├── 1s (parallel)
calendar_api() ┘               calendar_api() ─┘

Total: 2s                      Total: 1s
```

```go
func (a *Agent) SpeculativeExecute(possibleTools []ToolCall) []ToolResult {
    results := make(chan ToolResult, len(possibleTools))
    
    for _, tool := range possibleTools {
        go func(t ToolCall) {
            result := a.executeTool(t)
            results <- result
        }(tool)
    }
    
    // Collect all results
    allResults := []ToolResult{}
    for range possibleTools {
        allResults = append(allResults, <-results)
    }
    return allResults
    // LLM will use what it needs, ignore rest
}
```

**Trade-off**: More compute/API calls for lower latency. Worth it for independent tools.

#### 4. Actor Model for Multi-Agent

Erlang/Akka-style actors. Each agent is an actor with a mailbox.

```
┌─────────────────────────────────────────────────────────────┐
│                     Actor System                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    message    ┌──────────────┐            │
│  │   Planner    │──────────────▶│   Executor   │            │
│  │   Actor      │               │   Actor      │            │
│  │ ┌──────────┐ │               │ ┌──────────┐ │            │
│  │ │ Mailbox  │ │               │ │ Mailbox  │ │            │
│  │ └──────────┘ │               │ └──────────┘ │            │
│  └──────────────┘               └──────┬───────┘            │
│         ▲                              │                     │
│         │         result               │                     │
│         └──────────────────────────────┘                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

```go
type AgentActor struct {
    mailbox chan Message
    state   AgentState
    llm     LLMProvider
}

func (a *AgentActor) Run() {
    for msg := range a.mailbox {
        switch msg.Type {
        case "task":
            result := a.processTask(msg.Payload)
            msg.ReplyTo <- result
        case "query":
            a.handleQuery(msg)
        }
    }
}

// Send message to another agent
func (a *AgentActor) Send(target *AgentActor, msg Message) {
    target.mailbox <- msg
}
```

**Benefits**:
- Natural isolation (each agent has own state)
- Location transparency (agents can be distributed)
- Fault tolerance (supervisor hierarchies)

#### 5. Event Sourcing for Agent State

Store events, not state. Rebuild state by replaying.

```
Event Log:
┌────────────────────────────────────────────────────────────┐
│ 1. TaskStarted{id: "abc", input: "analyze data"}           │
│ 2. ToolCalled{tool: "read_file", args: {...}}              │
│ 3. ToolSucceeded{tool: "read_file", result: "..."}         │
│ 4. LLMCalled{prompt_hash: "xyz", tokens: 1234}             │
│ 5. LLMResponded{response_hash: "def", tool_calls: [...]}   │
│ 6. ToolCalled{tool: "calculator", args: {...}}             │
│ 7. TaskCompleted{id: "abc", result: "..."}                 │
└────────────────────────────────────────────────────────────┘

Current state = replay(events)
State at step 3 = replay(events[:3])
```

```go
type EventStore struct {
    events []Event
}

type Event interface {
    Apply(state *AgentState) *AgentState
}

func (s *EventStore) Append(event Event) {
    s.events = append(s.events, event)
}

func (s *EventStore) Rebuild() *AgentState {
    state := &AgentState{}
    for _, event := range s.events {
        state = event.Apply(state)
    }
    return state
}

func (s *EventStore) StateAt(index int) *AgentState {
    state := &AgentState{}
    for _, event := range s.events[:index] {
        state = event.Apply(state)
    }
    return state
}
```

**Benefits**:
- Complete audit trail
- Time travel debugging
- Easy replay for testing
- Natural fit for agent decisions

#### 6. Saga Pattern for Long-Running Tasks

Sequence of steps with compensating actions for rollback.

```
Forward actions:              Compensating actions:
                              
1. Create PR ─────────────── Delete PR
       │
       ▼
2. Run tests ─────────────── (no compensation needed)
       │
       ▼
3. Request review ────────── Cancel review request
       │
       ▼
4. Merge PR ──────────────── Revert merge
       │
       ▼
   SUCCESS

If step 3 fails:
- Run compensating action for step 1 (delete PR)
- Steps 2 has no compensation
- Result: Clean rollback
```

```go
type Saga struct {
    Steps []SagaStep
}

type SagaStep struct {
    Name       string
    Execute    func(ctx context.Context) error
    Compensate func(ctx context.Context) error
}

func (s *Saga) Run(ctx context.Context) error {
    completed := []SagaStep{}
    
    for _, step := range s.Steps {
        if err := step.Execute(ctx); err != nil {
            // Rollback in reverse order
            for i := len(completed) - 1; i >= 0; i-- {
                if completed[i].Compensate != nil {
                    completed[i].Compensate(ctx)
                }
            }
            return fmt.Errorf("saga failed at %s: %w", step.Name, err)
        }
        completed = append(completed, step)
    }
    return nil
}
```

**When to use**: Multi-step tasks with side effects, need clean rollback on failure.

#### 7. Circuit Breaker for LLM Calls

Fail fast when provider is unhealthy. Prevent cascade failures.

```
States:
┌────────┐  failures > threshold  ┌────────┐
│ CLOSED │───────────────────────▶│  OPEN  │
│(normal)│                        │ (fail  │
└────────┘                        │  fast) │
    ▲                             └───┬────┘
    │                                 │
    │    success                      │ timeout
    │                                 │
┌───┴────────┐◀───────────────────────┘
│ HALF-OPEN  │
│ (testing)  │
└────────────┘
```

```go
type CircuitBreaker struct {
    state           State
    failures        int
    threshold       int
    timeout         time.Duration
    lastFailureTime time.Time
}

func (cb *CircuitBreaker) Call(fn func() (*LLMResponse, error)) (*LLMResponse, error) {
    if cb.state == Open {
        if time.Since(cb.lastFailureTime) > cb.timeout {
            cb.state = HalfOpen
        } else {
            return nil, ErrCircuitOpen
        }
    }
    
    resp, err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailureTime = time.Now()
        if cb.failures >= cb.threshold {
            cb.state = Open
        }
        return nil, err
    }
    
    cb.failures = 0
    cb.state = Closed
    return resp, nil
}
```

#### 8. Token Bucket for Rate Limiting

Control rate of LLM calls to stay within quotas.

```
Bucket capacity: 10 tokens
Refill rate: 1 token/second

Time 0:  [••••••••••] 10 tokens
Call:    [•••••••••·]  9 tokens (1 consumed)
Call:    [••••••••··]  8 tokens
...
Time 10: [••••••••••] 10 tokens (refilled)
```

```go
type TokenBucket struct {
    tokens     float64
    capacity   float64
    refillRate float64  // tokens per second
    lastRefill time.Time
    mu         sync.Mutex
}

func (tb *TokenBucket) Take(n float64) bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    // Refill based on elapsed time
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    tb.tokens = min(tb.capacity, tb.tokens + elapsed * tb.refillRate)
    tb.lastRefill = now
    
    if tb.tokens >= n {
        tb.tokens -= n
        return true
    }
    return false
}
```

---

### Research Frontier

These are emerging areas, not yet production-ready but worth watching.

#### 1. World Models

Agents that build internal simulations to predict outcomes before acting.

```
Traditional:  Act → Observe outcome → Learn
World Model:  Imagine action → Predict outcome → Decide → Act

┌─────────────────────────────────────────────────────────────┐
│                    World Model                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Current State ──▶ Simulate Action A ──▶ Predicted State A  │
│        │                                                     │
│        └─────────▶ Simulate Action B ──▶ Predicted State B  │
│                                                              │
│  Choose action with best predicted outcome                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Research**: Dreamer, MuZero, IRIS

#### 2. Constitutional AI / Self-Critique

Agent critiques its own outputs against principles before responding.

```
Generate ──▶ Critique against principles ──▶ Revise ──▶ Output

Principles:
- Be helpful
- Be harmless  
- Be honest
- Don't reveal private information
- Verify facts before stating
```

**Research**: Anthropic Constitutional AI, Self-Refine

#### 3. Tool Learning

Agents that learn to use new tools from documentation alone.

```
Input: Tool documentation (API spec, examples)
Output: Ability to use tool correctly

No fine-tuning required - in-context learning of tool usage
```

**Research**: Toolformer, Gorilla, ToolLLM

#### 4. Hierarchical Task Networks (HTN) for LLM Planning

Decompose high-level goals into primitive actions using predefined methods.

```
Goal: "Deploy application"
         │
         ▼ decompose
┌─────────────────────────────────────┐
│ Method: deploy-to-production        │
│ Subtasks:                           │
│   1. run-tests                      │
│   2. build-artifact                 │
│   3. push-to-registry               │
│   4. update-kubernetes              │
└─────────────────────────────────────┘
         │
         ▼ decompose each
┌─────────────────────────────────────┐
│ Primitive actions (tool calls)      │
└─────────────────────────────────────┘
```

**Research**: LLM+P, SayCan, Inner Monologue

---

### Summary: What to Learn

| Category | Must Know | Good to Know | Research/Future |
|----------|-----------|--------------|-----------------|
| **Data Structures** | Vector indexes, Ring buffers | Persistent structures, Bloom filters | Merkle DAGs |
| **Algorithms** | Beam search, A* | MCTS, UCB1 | World models |
| **Patterns** | RAG pipeline, Circuit breaker | Event sourcing, Saga | Constitutional AI |
| **Concurrency** | Token bucket, Rate limiting | Actor model | Distributed consensus |

**The trend**: Classic CS is being adapted for non-deterministic, LLM-based systems. The fundamentals matter more than ever - you're just applying them in new contexts.
