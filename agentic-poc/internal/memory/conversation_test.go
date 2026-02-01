package memory

import (
	"sync"
	"testing"
)

func TestNewConversationMemory(t *testing.T) {
	mem := NewConversationMemory()

	if mem == nil {
		t.Fatal("NewConversationMemory returned nil")
	}

	if mem.Len() != 0 {
		t.Errorf("expected empty memory, got %d messages", mem.Len())
	}
}

func TestAddMessage(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		content string
	}{
		{
			name:    "user message",
			role:    "user",
			content: "Hello, how are you?",
		},
		{
			name:    "assistant message",
			role:    "assistant",
			content: "I'm doing well, thank you!",
		},
		{
			name:    "system message",
			role:    "system",
			content: "You are a helpful assistant.",
		},
		{
			name:    "empty content",
			role:    "user",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := NewConversationMemory()
			mem.AddMessage(tt.role, tt.content)

			messages := mem.GetMessages()
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			if messages[0].Role != tt.role {
				t.Errorf("expected role %q, got %q", tt.role, messages[0].Role)
			}

			if messages[0].Content != tt.content {
				t.Errorf("expected content %q, got %q", tt.content, messages[0].Content)
			}
		})
	}
}

func TestAddToolResult(t *testing.T) {
	tests := []struct {
		name       string
		toolCallID string
		toolName   string
		result     string
	}{
		{
			name:       "calculator result",
			toolCallID: "call_123",
			toolName:   "calculator",
			result:     "42",
		},
		{
			name:       "file reader result",
			toolCallID: "call_456",
			toolName:   "file_reader",
			result:     "file contents here",
		},
		{
			name:       "error result",
			toolCallID: "call_789",
			toolName:   "calculator",
			result:     "error: division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := NewConversationMemory()
			mem.AddToolResult(tt.toolCallID, tt.toolName, tt.result)

			messages := mem.GetMessages()
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			msg := messages[0]
			if msg.Role != "tool" {
				t.Errorf("expected role 'tool', got %q", msg.Role)
			}

			if msg.Content != tt.result {
				t.Errorf("expected content %q, got %q", tt.result, msg.Content)
			}

			if msg.ToolCallID != tt.toolCallID {
				t.Errorf("expected tool call ID %q, got %q", tt.toolCallID, msg.ToolCallID)
			}

			if msg.ToolName != tt.toolName {
				t.Errorf("expected tool name %q, got %q", tt.toolName, msg.ToolName)
			}
		})
	}
}

func TestGetMessages_ReturnsOrderedMessages(t *testing.T) {
	mem := NewConversationMemory()

	// Add messages in sequence
	mem.AddMessage("user", "First message")
	mem.AddMessage("assistant", "Second message")
	mem.AddMessage("user", "Third message")

	messages := mem.GetMessages()

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	// Verify order is preserved
	expected := []struct {
		role    string
		content string
	}{
		{"user", "First message"},
		{"assistant", "Second message"},
		{"user", "Third message"},
	}

	for i, exp := range expected {
		if messages[i].Role != exp.role {
			t.Errorf("message %d: expected role %q, got %q", i, exp.role, messages[i].Role)
		}
		if messages[i].Content != exp.content {
			t.Errorf("message %d: expected content %q, got %q", i, exp.content, messages[i].Content)
		}
	}
}

func TestGetMessages_ReturnsCopy(t *testing.T) {
	mem := NewConversationMemory()
	mem.AddMessage("user", "Original message")

	// Get messages and modify the returned slice
	messages := mem.GetMessages()
	messages[0].Content = "Modified content"

	// Verify original is unchanged
	originalMessages := mem.GetMessages()
	if originalMessages[0].Content != "Original message" {
		t.Error("GetMessages did not return a copy; original was modified")
	}
}

func TestClear(t *testing.T) {
	mem := NewConversationMemory()

	// Add some messages
	mem.AddMessage("user", "Message 1")
	mem.AddMessage("assistant", "Message 2")
	mem.AddToolResult("call_1", "calculator", "42")

	if mem.Len() != 3 {
		t.Fatalf("expected 3 messages before clear, got %d", mem.Len())
	}

	// Clear the memory
	mem.Clear()

	if mem.Len() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", mem.Len())
	}

	messages := mem.GetMessages()
	if len(messages) != 0 {
		t.Errorf("expected empty slice after clear, got %d messages", len(messages))
	}
}

func TestLen(t *testing.T) {
	mem := NewConversationMemory()

	if mem.Len() != 0 {
		t.Errorf("expected length 0, got %d", mem.Len())
	}

	mem.AddMessage("user", "Message 1")
	if mem.Len() != 1 {
		t.Errorf("expected length 1, got %d", mem.Len())
	}

	mem.AddMessage("assistant", "Message 2")
	if mem.Len() != 2 {
		t.Errorf("expected length 2, got %d", mem.Len())
	}

	mem.AddToolResult("call_1", "tool", "result")
	if mem.Len() != 3 {
		t.Errorf("expected length 3, got %d", mem.Len())
	}
}

func TestMixedMessageTypes(t *testing.T) {
	mem := NewConversationMemory()

	// Add a mix of regular messages and tool results
	mem.AddMessage("user", "What is 2 + 2?")
	mem.AddMessage("assistant", "Let me calculate that.")
	mem.AddToolResult("call_1", "calculator", "4")
	mem.AddMessage("assistant", "The answer is 4.")

	messages := mem.GetMessages()
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	// Verify the sequence
	if messages[0].Role != "user" || messages[0].Content != "What is 2 + 2?" {
		t.Error("message 0 incorrect")
	}
	if messages[1].Role != "assistant" || messages[1].Content != "Let me calculate that." {
		t.Error("message 1 incorrect")
	}
	if messages[2].Role != "tool" || messages[2].Content != "4" || messages[2].ToolCallID != "call_1" {
		t.Error("message 2 (tool result) incorrect")
	}
	if messages[3].Role != "assistant" || messages[3].Content != "The answer is 4." {
		t.Error("message 3 incorrect")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mem := NewConversationMemory()
	var wg sync.WaitGroup

	// Spawn multiple goroutines to add messages concurrently
	numGoroutines := 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			mem.AddMessage("user", "Message from goroutine")
		}(i)
	}

	wg.Wait()

	// Verify all messages were added
	if mem.Len() != numGoroutines {
		t.Errorf("expected %d messages, got %d", numGoroutines, mem.Len())
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	mem := NewConversationMemory()
	var wg sync.WaitGroup

	// Add some initial messages
	for i := 0; i < 10; i++ {
		mem.AddMessage("user", "Initial message")
	}

	// Spawn readers and writers concurrently
	numReaders := 50
	numWriters := 50

	wg.Add(numReaders + numWriters)

	// Writers
	for i := 0; i < numWriters; i++ {
		go func() {
			defer wg.Done()
			mem.AddMessage("user", "Concurrent message")
		}()
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			_ = mem.GetMessages()
			_ = mem.Len()
		}()
	}

	wg.Wait()

	// Verify final count
	expectedCount := 10 + numWriters
	if mem.Len() != expectedCount {
		t.Errorf("expected %d messages, got %d", expectedCount, mem.Len())
	}
}
