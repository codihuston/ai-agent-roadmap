# Architecture Documentation

This document provides C4 diagrams for the three main operational modes of the agentic system.

## Table of Contents
MCP
1. [System Context](#system-context)
2. [Single Agent Mode](#single-agent-mode)
3. [Multi-Agent Mode](#multi-agent-mode)
4. [ Server Mode](#mcp-server-mode)

---

## System Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              System Context                                  │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐                                          ┌──────────────────┐
    │          │                                          │                  │
    │   User   │─────────── Uses ──────────────────────▶  │  Agentic System  │
    │          │                                          │                  │
    └──────────┘                                          └────────┬─────────┘
                                                                   │
                                                                   │ Calls
                                                                   ▼
                                                          ┌──────────────────┐
                                                          │                  │
                                                          │   Claude API     │
                                                          │   (Anthropic)    │
                                                          │                  │
                                                          └──────────────────┘
```

**Description**: The user interacts with the Agentic System via CLI. The system uses Claude API for LLM capabilities.

---

## Single Agent Mode

### Container Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Single Agent Mode - Containers                       │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐          ┌─────────────────────────────────────────────────┐
    │          │          │              Agentic System                      │
    │   User   │          │  ┌─────────┐    ┌─────────┐    ┌─────────────┐  │
    │          │──stdin──▶│  │   CLI   │───▶│  Agent  │───▶│   Tools     │  │
    │          │◀─stdout──│  │         │◀───│         │◀───│ (Calculator │  │
    └──────────┘          │  └─────────┘    └────┬────┘    │  FileReader)│  │
                          │                      │         └─────────────┘  │
                          │                      │                          │
                          └──────────────────────┼──────────────────────────┘
                                                 │
                                                 │ HTTP/JSON
                                                 ▼
                                          ┌──────────────┐
                                          │  Claude API  │
                                          └──────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Single Agent Mode - Components                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                                   CLI                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  RunSingleAgentMode()                                                 │   │
│  │  - Creates Agent with tools                                           │   │
│  │  - Reads user input                                                   │   │
│  │  - Displays tool calls and responses                                  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                  Agent                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Run(input, memory) → AgentResult                                     │   │
│  │                                                                       │   │
│  │  Think → Act → Observe Loop:                                          │   │
│  │  1. Send messages to LLM                                              │   │
│  │  2. Parse tool calls from response                                    │   │
│  │  3. Execute tools                                                     │   │
│  │  4. Feed results back to LLM                                          │   │
│  │  5. Repeat until final response                                       │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
          │                                           │
          ▼                                           ▼
┌─────────────────────┐                    ┌─────────────────────┐
│   ClaudeProvider    │                    │       Tools         │
│  ┌───────────────┐  │                    │  ┌───────────────┐  │
│  │ Generate()    │  │                    │  │ Calculator    │  │
│  │ - Build req   │  │                    │  │ - add         │  │
│  │ - Send HTTP   │  │                    │  │ - subtract    │  │
│  │ - Parse resp  │  │                    │  │ - multiply    │  │
│  └───────────────┘  │                    │  │ - divide      │  │
└─────────────────────┘                    │  └───────────────┘  │
                                           │  ┌───────────────┐  │
                                           │  │ FileReader    │  │
                                           │  │ - read_file   │  │
                                           │  └───────────────┘  │
                                           └─────────────────────┘
```

### Sequence Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Single Agent Mode - Sequence                           │
└─────────────────────────────────────────────────────────────────────────────┘

User          CLI           Agent         ClaudeProvider      Calculator
 │             │              │                 │                  │
 │──"5+3"────▶│              │                 │                  │
 │             │──Run()─────▶│                 │                  │
 │             │              │──Generate()───▶│                  │
 │             │              │                 │──HTTP POST─────▶│ Claude API
 │             │              │                 │◀─tool_use───────│
 │             │              │◀─ToolCalls─────│                  │
 │             │              │                 │                  │
 │             │              │──Execute()────────────────────────▶│
 │             │              │◀─"8"───────────────────────────────│
 │             │              │                 │                  │
 │             │              │──Generate()───▶│                  │
 │             │              │                 │──HTTP POST─────▶│ Claude API
 │             │              │◀─"5+3=8"───────│                  │
 │             │◀─Result──────│                 │                  │
 │◀─"5+3=8"───│              │                 │                  │
 │             │              │                 │                  │
```

---

## Multi-Agent Mode

### Container Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Multi-Agent Mode - Containers                         │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐          ┌─────────────────────────────────────────────────┐
    │          │          │              Agentic System                      │
    │   User   │          │                                                  │
    │          │──stdin──▶│  ┌─────────┐    ┌──────────────┐                │
    │          │◀─stdout──│  │   CLI   │───▶│ Orchestrator │                │
    └──────────┘          │  └─────────┘    └──────┬───────┘                │
                          │                        │                         │
                          │           ┌────────────┴────────────┐           │
                          │           ▼                         ▼           │
                          │    ┌────────────┐           ┌────────────┐      │
                          │    │  Architect │           │   Coder    │      │
                          │    │   Agent    │           │   Agent    │      │
                          │    └─────┬──────┘           └─────┬──────┘      │
                          │          │                        │             │
                          │          ▼                        ▼             │
                          │    ┌────────────┐           ┌────────────┐      │
                          │    │FinishPlan  │           │FileReader  │      │
                          │    │   Tool     │           │FileWriter  │      │
                          │    └────────────┘           └────────────┘      │
                          └─────────────────────────────────────────────────┘
                                           │
                                           │ HTTP/JSON
                                           ▼
                                    ┌──────────────┐
                                    │  Claude API  │
                                    └──────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Multi-Agent Mode - Components                          │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              Orchestrator                                    │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Run(goal) → WorkflowResult                                           │   │
│  │                                                                       │   │
│  │  Workflow Phases:                                                     │   │
│  │  1. Idle → Planning (create Architect)                                │   │
│  │  2. Planning → Executing (Architect creates plan)                     │   │
│  │  3. Executing → Complete (Coder executes plan)                        │   │
│  │  4. Any → Failed (on error)                                           │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
          │                                           │
          ▼                                           ▼
┌─────────────────────────────┐          ┌─────────────────────────────┐
│      Architect Agent        │          │        Coder Agent          │
│  ┌───────────────────────┐  │          │  ┌───────────────────────┐  │
│  │ System Prompt:        │  │          │  │ System Prompt:        │  │
│  │ "Break down goals     │  │          │  │ "Execute plan steps   │  │
│  │  into actionable      │  │          │  │  using file tools"    │  │
│  │  plans"               │  │          │  │                       │  │
│  └───────────────────────┘  │          │  └───────────────────────┘  │
│  ┌───────────────────────┐  │          │  ┌───────────────────────┐  │
│  │ Tools:                │  │          │  │ Tools:                │  │
│  │ - finish_plan         │  │          │  │ - read_file           │  │
│  └───────────────────────┘  │          │  │ - write_file          │  │
└─────────────────────────────┘          │  └───────────────────────┘  │
                                         └─────────────────────────────┘
```

### Sequence Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Multi-Agent Mode - Sequence                            │
└─────────────────────────────────────────────────────────────────────────────┘

User      CLI      Orchestrator    Architect    Coder     Claude    FileSystem
 │         │            │              │          │          │           │
 │─"goal"─▶│            │              │          │          │           │
 │         │──Run()────▶│              │          │          │           │
 │         │            │──Run()──────▶│          │          │           │
 │         │            │              │─Generate()─────────▶│           │
 │         │            │              │◀─finish_plan────────│           │
 │         │            │              │                     │           │
 │         │            │◀─Plan────────│          │          │           │
 │         │            │              │          │          │           │
 │         │            │──Run(plan)─────────────▶│          │           │
 │         │            │              │          │─Generate()──────────▶│
 │         │            │              │          │◀─write_file──────────│
 │         │            │              │          │                      │
 │         │            │              │          │──WriteFile()────────▶│
 │         │            │              │          │◀─success─────────────│
 │         │            │              │          │                      │
 │         │            │              │          │─Generate()──────────▶│
 │         │            │              │          │◀─"done"──────────────│
 │         │            │◀─Result──────────────────│          │           │
 │         │◀─Result────│              │          │          │           │
 │◀─Result─│            │              │          │          │           │
 │         │            │              │          │          │           │
```

---

## MCP Server Mode

### Container Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MCP Server Mode - Containers                         │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐          ┌─────────────────────────────────────────────────┐
    │          │          │              Agentic System                      │
    │   User   │          │                                                  │
    │          │──stdin──▶│  ┌─────────┐    ┌─────────┐    ┌────────────┐  │
    │          │◀─stdout──│  │   CLI   │───▶│  Agent  │───▶│MCPManager  │  │
    └──────────┘          │  └─────────┘    └────┬────┘    └─────┬──────┘  │
                          │                      │               │          │
                          │                      │               │          │
                          └──────────────────────┼───────────────┼──────────┘
                                                 │               │
                                                 │               │ JSON-RPC
                                                 │               │ (stdio)
                                                 │               ▼
                                                 │    ┌─────────────────────┐
                                                 │    │    MCP Server       │
                                                 │    │  (subprocess)       │
                                                 │    │  ┌───────────────┐  │
                                                 │    │  │  Calculator   │  │
                                                 │    │  │  FileReader   │  │
                                                 │    │  └───────────────┘  │
                                                 │    └─────────────────────┘
                                                 │
                                                 │ HTTP/JSON
                                                 ▼
                                          ┌──────────────┐
                                          │  Claude API  │
                                          └──────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        MCP Server Mode - Components                          │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                               MCPManager                                     │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  LoadFromConfig(mcp.json)                                             │   │
│  │  - Spawns MCP server subprocess                                       │   │
│  │  - Initializes JSON-RPC connection                                    │   │
│  │  - Lists available tools                                              │   │
│  │  - Wraps tools as Tool interface                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            StdioMCPClient                                    │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Connect()     - Start subprocess, send initialize                    │   │
│  │  ListTools()   - Get tool definitions via tools/list                  │   │
│  │  CallTool()    - Execute tool via tools/call                          │   │
│  │  Close()       - Kill subprocess                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
          │
          │ stdin/stdout (JSON-RPC 2.0)
          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCPServer                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Serve(stdin, stdout)                                                 │   │
│  │                                                                       │   │
│  │  Handles JSON-RPC methods:                                            │   │
│  │  - initialize      → Server info + capabilities                       │   │
│  │  - tools/list      → List of tool definitions                         │   │
│  │  - tools/call      → Execute tool, return result                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │  Registered Tools:                                                    │   │
│  │  - Calculator (add, subtract, multiply, divide)                       │   │
│  │  - FileReader (read_file)                                             │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Sequence Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        MCP Server Mode - Sequence                            │
└─────────────────────────────────────────────────────────────────────────────┘

User    CLI     Agent    MCPManager   StdioClient   MCPServer   Calculator
 │       │        │          │            │             │            │
 │       │        │          │            │             │            │
 │       │        │──Load()─▶│            │             │            │
 │       │        │          │──Connect()▶│             │            │
 │       │        │          │            │──spawn─────▶│            │
 │       │        │          │            │◀─running────│            │
 │       │        │          │            │             │            │
 │       │        │          │            │─initialize─▶│            │
 │       │        │          │            │◀─ok─────────│            │
 │       │        │          │            │             │            │
 │       │        │          │            │─tools/list─▶│            │
 │       │        │          │            │◀─[calc,file]│            │
 │       │        │          │◀─tools─────│             │            │
 │       │        │◀─tools───│            │             │            │
 │       │        │          │            │             │            │
 │─"5+3"▶│        │          │            │             │            │
 │       │─Run()─▶│          │            │             │            │
 │       │        │─────────────────────Generate()────────────────▶Claude
 │       │        │◀────────────────────tool_use(calc)─────────────Claude
 │       │        │          │            │             │            │
 │       │        │─Execute()────────────▶│             │            │
 │       │        │          │            │─tools/call─▶│            │
 │       │        │          │            │             │─Execute()─▶│
 │       │        │          │            │             │◀─"8"───────│
 │       │        │          │            │◀─result─────│            │
 │       │        │◀─"8"─────────────────│             │            │
 │       │        │          │            │             │            │
 │       │        │─────────────────────Generate()────────────────▶Claude
 │       │        │◀────────────────────"5+3=8"────────────────────Claude
 │       │◀─Result│          │            │             │            │
 │◀─"8"──│        │          │            │             │            │
```

### JSON-RPC Protocol Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MCP JSON-RPC Protocol                                │
└─────────────────────────────────────────────────────────────────────────────┘

Client (StdioMCPClient)                          Server (MCPServer)
        │                                               │
        │  {"jsonrpc":"2.0","id":1,                     │
        │   "method":"initialize",                      │
        │   "params":{...}}                             │
        │──────────────────────────────────────────────▶│
        │                                               │
        │  {"jsonrpc":"2.0","id":1,                     │
        │   "result":{"protocolVersion":"2024-11-05",   │
        │            "serverInfo":{...}}}               │
        │◀──────────────────────────────────────────────│
        │                                               │
        │  {"jsonrpc":"2.0",                            │
        │   "method":"notifications/initialized"}       │
        │──────────────────────────────────────────────▶│
        │                                               │
        │  {"jsonrpc":"2.0","id":2,                     │
        │   "method":"tools/list"}                      │
        │──────────────────────────────────────────────▶│
        │                                               │
        │  {"jsonrpc":"2.0","id":2,                     │
        │   "result":{"tools":[                         │
        │     {"name":"calculator",...},                │
        │     {"name":"read_file",...}]}}               │
        │◀──────────────────────────────────────────────│
        │                                               │
        │  {"jsonrpc":"2.0","id":3,                     │
        │   "method":"tools/call",                      │
        │   "params":{"name":"calculator",              │
        │            "arguments":{"op":"add",           │
        │                        "a":5,"b":3}}}         │
        │──────────────────────────────────────────────▶│
        │                                               │
        │  {"jsonrpc":"2.0","id":3,                     │
        │   "result":{"content":[{"type":"text",        │
        │                        "text":"8"}],          │
        │            "isError":false}}                  │
        │◀──────────────────────────────────────────────│
        │                                               │
```

---

## Package Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Package Dependencies                               │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌─────────────┐
                              │  cmd/agent  │
                              │  cmd/mcp-   │
                              │   server    │
                              └──────┬──────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
             ┌──────────┐     ┌────────────┐   ┌──────────┐
             │   cli    │     │orchestrator│   │   mcp    │
             └────┬─────┘     └─────┬──────┘   └────┬─────┘
                  │                 │               │
         ┌────────┴────────┐       │               │
         ▼                 ▼       ▼               ▼
    ┌─────────┐      ┌─────────────────┐     ┌─────────┐
    │  agent  │◀─────│                 │     │  tool   │
    └────┬────┘      │                 │     └────┬────┘
         │           │                 │          │
         ▼           ▼                 ▼          ▼
    ┌─────────┐ ┌─────────┐      ┌──────────────────┐
    │ memory  │ │  tool   │      │    provider      │
    └────┬────┘ └────┬────┘      │  (types.go)      │
         │          │           └──────────────────┘
         └──────────┴──────────────────┘
                    │
                    ▼
             ┌──────────────┐
             │   provider   │
             │ (claude.go)  │
             └──────────────┘
```

---

## File Structure

```
agentic-poc/
├── cmd/
│   ├── agent/
│   │   └── main.go              # CLI entry point
│   └── mcp-server/
│       └── main.go              # MCP server entry point
├── internal/
│   ├── agent/
│   │   ├── agent.go             # Core agent loop
│   │   ├── architect.go         # Architect agent factory
│   │   ├── coder.go             # Coder agent factory
│   │   └── plan.go              # Plan data structures
│   ├── cli/
│   │   └── cli.go               # CLI implementation
│   ├── mcp/
│   │   ├── client.go            # MCPClient interface
│   │   ├── config.go            # mcp.json loading
│   │   ├── manager.go           # Multi-server management
│   │   ├── server.go            # MCP server implementation
│   │   ├── stdio.go             # Stdio MCP client
│   │   └── wrapper.go           # MCP tool wrapper
│   ├── memory/
│   │   └── conversation.go      # Conversation memory
│   ├── orchestrator/
│   │   └── orchestrator.go      # Multi-agent coordination
│   ├── provider/
│   │   ├── claude.go            # Claude API client
│   │   ├── provider.go          # LLMProvider interface
│   │   └── types.go             # Shared types
│   └── tool/
│       ├── calculator.go        # Calculator tool
│       ├── file_reader.go       # File reader tool
│       ├── file_writer.go       # File writer tool
│       ├── finish_plan.go       # Plan capture tool
│       └── tool.go              # Tool interface
├── test/
│   └── integration/
│       └── workflow_test.go     # Integration tests
├── docs/
│   └── wiki/
│       ├── ARCHITECTURE.md      # This file
│       └── LEARNINGS.md         # Design decisions
├── mcp.json                     # MCP server config
└── mcp.json.example             # Example config
```
