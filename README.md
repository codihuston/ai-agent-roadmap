# Agentic System POC

A Proof of Concept agentic system built in Go demonstrating core agentic patterns: single-agent tool use and multi-agent orchestration.

## Features

- **Single Agent Mode**: Interactive agent with calculator and file reader tools
- **Multi-Agent Mode**: Architect/Coder workflow for goal-driven task execution
- **Provider Abstraction**: Pluggable LLM provider interface (Claude implemented)
- **Tool System**: Extensible tool interface with built-in tools

## Requirements

- Go 1.21 or later
- Anthropic API key (for Claude provider)

## Installation

```bash
git clone <repository-url>
cd agentic-poc
go mod download
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | Yes | Your Anthropic API key for Claude |

## Usage

### Build

```bash
go build -o agent ./cmd/agent
```

### Single Agent Mode (Default)

Interactive mode with access to calculator and file reader tools:

```bash
./agent
# or explicitly:
./agent -mode single
```

Example interaction:
```
=== Single Agent Mode ===
Available tools: calculator, read_file
Type 'exit' or 'quit' to exit.

You: What is 15 + 27?

--- Intermediate Steps ---
  [Tool Call] calculator
    operation: add
    a: 15
    b: 27
  Iterations: 2
---------------------------

Assistant: The sum of 15 and 27 is 42.
```

### Multi-Agent Mode

Architect creates a plan, Coder executes it:

```bash
./agent -mode multi
# With custom base path for file operations:
./agent -mode multi -path /tmp/workspace
```

Example interaction:
```
=== Multi-Agent Mode (Architect/Coder) ===
Enter a goal for the system to accomplish.
Type 'exit' or 'quit' to exit.

Goal: Create a hello world file

>>> Agent Transition: user -> architect

--- Plan ---
Goal: Create a hello world file
Steps:
  1. Create hello.txt with greeting (action: write_file)
------------

>>> Agent Transition: architect -> coder

--- Actions Taken ---
  • write_file: map[content:Hello, World! path:hello.txt]
---------------------

Summary: Successfully created hello.txt
Success: true
```

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | `single` | Mode: `single` or `multi` |
| `-path` | `.` | Base path for file operations |
| `-help` | - | Show help message |

## Project Structure

```
agentic-poc/
├── cmd/agent/          # CLI entry point
├── internal/
│   ├── provider/       # LLM provider abstraction
│   ├── tool/           # Tool interface and implementations
│   ├── memory/         # Conversation history
│   ├── agent/          # Agent loop and specialized agents
│   ├── orchestrator/   # Multi-agent coordination
│   └── cli/            # Command-line interface
├── test/integration/   # End-to-end tests
└── docs/wiki/          # Development learnings
```

## Architecture

For detailed C4 diagrams covering all operational modes, see [docs/wiki/ARCHITECTURE.md](docs/wiki/ARCHITECTURE.md).

### Single Agent Flow
```
┌──────┐     ┌───────┐     ┌─────────┐     ┌───────┐
│ User │────▶│  CLI  │────▶│  Agent  │────▶│ Claude│
└──────┘     └───────┘     └────┬────┘     └───┬───┘
                                │              │
                                ▼              │
                           ┌─────────┐         │
                           │  Tools  │◀────────┘
                           │(calc,   │  tool_use
                           │ file)   │
                           └─────────┘
```

### Multi-Agent Flow
```
┌──────┐     ┌────────────┐     ┌───────────┐     ┌───────┐
│ User │────▶│Orchestrator│────▶│ Architect │────▶│ Claude│
└──────┘     └─────┬──────┘     └─────┬─────┘     └───────┘
                   │                  │
                   │                  ▼ Plan
                   │            ┌───────────┐     ┌───────┐
                   └───────────▶│   Coder   │────▶│ Claude│
                                └─────┬─────┘     └───────┘
                                      │
                                      ▼
                                ┌───────────┐
                                │FileReader │
                                │FileWriter │
                                └───────────┘
```

### MCP Server Mode
```
┌──────┐     ┌───────┐     ┌─────────┐     ┌────────────┐
│ User │────▶│  CLI  │────▶│  Agent  │────▶│ MCPManager │
└──────┘     └───────┘     └────┬────┘     └──────┬─────┘
                                │                 │
                                │                 │ JSON-RPC
                                │                 ▼
                                │          ┌────────────┐
                                │          │ MCP Server │
                                │          │(subprocess)│
                                │          └──────┬─────┘
                                │                 │
                                ▼                 ▼
                           ┌─────────┐      ┌─────────┐
                           │ Claude  │      │  Tools  │
                           └─────────┘      └─────────┘
```

## Running Tests

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# Integration tests only
go test -v ./test/integration/
```

## Development

See [docs/wiki/LEARNINGS.md](docs/wiki/LEARNINGS.md) for design decisions, challenges, and observations encountered during development.

## License

MIT