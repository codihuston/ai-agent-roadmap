// Package memory provides conversation history management for agents.
package memory

import (
	"sync"

	"agentic-poc/internal/provider"
)

// ConversationMemory maintains an ordered list of messages for conversation context.
// It is thread-safe and can be used concurrently by multiple goroutines.
type ConversationMemory struct {
	messages []provider.Message
	mu       sync.RWMutex
}

// NewConversationMemory creates a new empty ConversationMemory.
func NewConversationMemory() *ConversationMemory {
	return &ConversationMemory{
		messages: make([]provider.Message, 0),
	}
}

// AddMessage appends a new message with the given role and content to the conversation history.
func (m *ConversationMemory) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, provider.Message{
		Role:    role,
		Content: content,
	})
}

// AddAssistantMessageWithToolCalls appends an assistant message that includes tool calls.
// This is used when the LLM responds with tool_use blocks.
func (m *ConversationMemory) AddAssistantMessageWithToolCalls(content string, toolCalls []provider.ToolCall) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, provider.Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult appends a tool result message to the conversation history.
// The message includes the tool call ID and tool name for proper context.
func (m *ConversationMemory) AddToolResult(toolCallID, toolName, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, provider.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
		ToolName:   toolName,
	})
}

// GetMessages returns a copy of all messages in the conversation history.
// The returned slice is a copy to prevent external modification.
func (m *ConversationMemory) GetMessages() []provider.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]provider.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// Clear removes all messages from the conversation history.
func (m *ConversationMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]provider.Message, 0)
}

// Len returns the number of messages in the conversation history.
func (m *ConversationMemory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}
