---
inclusion: always
---

# Go Development Standards

## Git Workflow

Use conventional commits with trunk-based development:

```
<type>(<scope>): <description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Examples:
- `feat(agent): implement Think-Act-Observe loop`
- `fix(tool): handle division by zero in calculator`
- `test(memory): add property tests for message ordering`
- `docs(wiki): document LLM response parsing challenges`

Commit directly to `main`. Keep commits small and focused. No long-lived branches.

## Before Every Commit

1. Run tests: `go test ./...`
2. Format code: `go fmt ./...`
3. Vet code: `go vet ./...`

Never commit without passing tests and formatted code.

## Task Completion

Commit after each task is completed. One task = one commit (or a few small commits if the task is large).

Update `docs/wiki/LEARNINGS.md` with any design decisions or challenges encountered during the task before committing.

## Code Style

Write idiomatic Go:

- Use `gofmt` formatting (no exceptions)
- Prefer composition over inheritance
- Return errors, don't panic (except truly unrecoverable)
- Use interfaces for abstraction, concrete types for implementation
- Keep functions short and focused
- Name things clearly: `NewAgent()` not `CreateNewAgentInstance()`
- Use table-driven tests
- Document exported types and functions

## DRY & SOLID

**DRY**: Extract common patterns into shared functions. Don't copy-paste.

**SOLID in Go**:
- **S**: One responsibility per package/type
- **O**: Use interfaces to extend behavior without modifying existing code
- **L**: Implementations must satisfy interface contracts fully
- **I**: Small, focused interfaces (`io.Reader` not `DoEverything`)
- **D**: Depend on interfaces, not concrete types

## Testing

Write tests alongside implementation:

- Unit tests: `*_test.go` next to source files
- Property tests: Use `gopter` for invariant verification
- Integration tests: `test/integration/`

Test the behavior, not the implementation. Mock external dependencies (LLM APIs), not internal logic.

## Documentation

Document design decisions and challenges in `docs/wiki/LEARNINGS.md`:

- One document, ordered from beginner concepts to advanced
- Include code examples where helpful
- Update as you encounter new challenges
- Categories: parsing, tool calling, memory, orchestration, LLM quirks

## Project Structure

```
agentic-poc/
├── cmd/agent/          # Entry point only
├── internal/           # All implementation
│   ├── provider/       # LLM abstraction
│   ├── tool/           # Tool interface + implementations
│   ├── memory/         # Conversation history
│   ├── agent/          # Agent loop + specialized agents
│   ├── orchestrator/   # Multi-agent coordination
│   └── cli/            # User interface
├── test/integration/   # End-to-end tests
└── docs/wiki/          # Learnings document
```

## Error Handling

Wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to call LLM: %w", err)
}
```

Return errors up the stack. Let callers decide how to handle them.
