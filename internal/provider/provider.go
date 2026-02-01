// Package provider defines the LLM provider abstraction and core data types.
package provider

import "context"

// LLMProvider defines the interface for LLM communication.
// Any LLM service (Claude, Gemini, OpenAI, etc.) can be used by implementing this interface.
type LLMProvider interface {
	// Generate sends a request to the LLM and returns the response.
	// The request includes messages, optional tools, and an optional system prompt.
	// Returns an LLMResponse containing either text or tool calls.
	Generate(ctx context.Context, req GenerateRequest) (*LLMResponse, error)

	// Name returns the name of the provider (e.g., "claude", "gemini", "openai").
	Name() string
}
