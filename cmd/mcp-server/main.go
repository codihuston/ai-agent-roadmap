// Package main provides the entry point for the MCP server that exposes
// built-in tools via the Model Context Protocol.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"agentic-poc/internal/mcp"
	"agentic-poc/internal/tool"
)

func main() {
	// Create tools to expose
	tools := []tool.Tool{
		tool.NewCalculatorTool(),
		tool.NewFileReaderTool("."), // Use current directory as base
	}

	// Create MCP server
	server := mcp.NewMCPServer("agentic-poc-tools", "1.0.0", tools)

	// Setup context with cancellation on signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run server on stdin/stdout
	log.SetOutput(os.Stderr) // Redirect logs to stderr to not interfere with JSON-RPC
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		if err != context.Canceled {
			log.Fatalf("Server error: %v", err)
		}
	}
}
