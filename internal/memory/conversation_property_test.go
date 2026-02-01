package memory

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Validates: Requirements 2.1, 2.2**
//
// Property 1: Conversation Memory Preserves Order and Content
// For any sequence of messages added to ConversationMemory, the messages returned
// by GetMessages() SHALL appear in the same order they were added, with each message
// containing the exact role and content that was provided.

// messageInput represents a message to be added to ConversationMemory
type messageInput struct {
	Role    string
	Content string
}

// genRole generates valid message roles
func genRole() gopter.Gen {
	return gen.OneConstOf("user", "assistant", "system", "tool")
}

// genContent generates message content strings
func genContent() gopter.Gen {
	return gen.AnyString()
}

// genMessageInput generates a single message input
func genMessageInput() gopter.Gen {
	return gopter.CombineGens(
		genRole(),
		genContent(),
	).Map(func(values []interface{}) messageInput {
		return messageInput{
			Role:    values[0].(string),
			Content: values[1].(string),
		}
	})
}

// genMessageSequence generates a sequence of messages (1 to 50 messages)
func genMessageSequence() gopter.Gen {
	return gen.SliceOfN(50, genMessageInput()).SuchThat(func(msgs []messageInput) bool {
		return len(msgs) > 0
	})
}

func TestProperty_MessageOrderingPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 50

	properties := gopter.NewProperties(parameters)

	// **Feature: agentic-system-poc, Property 1: Conversation Memory Preserves Order and Content**
	properties.Property("messages are returned in the same order they were added", prop.ForAll(
		func(inputs []messageInput) bool {
			mem := NewConversationMemory()

			// Add all messages
			for _, input := range inputs {
				mem.AddMessage(input.Role, input.Content)
			}

			// Get messages back
			messages := mem.GetMessages()

			// Verify count matches
			if len(messages) != len(inputs) {
				return false
			}

			// Verify order and content preservation
			for i, input := range inputs {
				if messages[i].Role != input.Role {
					return false
				}
				if messages[i].Content != input.Content {
					return false
				}
			}

			return true
		},
		genMessageSequence(),
	))

	properties.TestingRun(t)
}

func TestProperty_MessageContentExactPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// **Feature: agentic-system-poc, Property 1: Content Exact Preservation**
	properties.Property("message content is preserved exactly as provided", prop.ForAll(
		func(role string, content string) bool {
			mem := NewConversationMemory()
			mem.AddMessage(role, content)

			messages := mem.GetMessages()
			if len(messages) != 1 {
				return false
			}

			// Content must be exactly equal (byte-for-byte)
			return messages[0].Role == role && messages[0].Content == content
		},
		genRole(),
		genContent(),
	))

	properties.TestingRun(t)
}

func TestProperty_ToolResultOrderingPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// toolResultInput represents a tool result to be added
	type toolResultInput struct {
		ToolCallID string
		ToolName   string
		Result     string
	}

	genToolResultInput := gopter.CombineGens(
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AnyString(),
	).Map(func(values []interface{}) toolResultInput {
		return toolResultInput{
			ToolCallID: values[0].(string),
			ToolName:   values[1].(string),
			Result:     values[2].(string),
		}
	})

	genToolResultSequence := gen.SliceOfN(30, genToolResultInput).SuchThat(func(results []toolResultInput) bool {
		return len(results) > 0
	})

	// **Feature: agentic-system-poc, Property 1: Tool Result Ordering Preservation**
	properties.Property("tool results are returned in the same order they were added", prop.ForAll(
		func(inputs []toolResultInput) bool {
			mem := NewConversationMemory()

			// Add all tool results
			for _, input := range inputs {
				mem.AddToolResult(input.ToolCallID, input.ToolName, input.Result)
			}

			// Get messages back
			messages := mem.GetMessages()

			// Verify count matches
			if len(messages) != len(inputs) {
				return false
			}

			// Verify order and content preservation
			for i, input := range inputs {
				msg := messages[i]
				if msg.Role != "tool" {
					return false
				}
				if msg.ToolCallID != input.ToolCallID {
					return false
				}
				if msg.ToolName != input.ToolName {
					return false
				}
				if msg.Content != input.Result {
					return false
				}
			}

			return true
		},
		genToolResultSequence,
	))

	properties.TestingRun(t)
}

func TestProperty_MixedMessageOrderingPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// mixedInput represents either a regular message or a tool result
	type mixedInput struct {
		IsToolResult bool
		Role         string
		Content      string
		ToolCallID   string
		ToolName     string
	}

	genMixedInput := gopter.CombineGens(
		gen.Bool(),
		genRole(),
		genContent(),
		gen.AlphaString(),
		gen.AlphaString(),
	).Map(func(values []interface{}) mixedInput {
		return mixedInput{
			IsToolResult: values[0].(bool),
			Role:         values[1].(string),
			Content:      values[2].(string),
			ToolCallID:   values[3].(string),
			ToolName:     values[4].(string),
		}
	})

	genMixedSequence := gen.SliceOfN(40, genMixedInput).SuchThat(func(inputs []mixedInput) bool {
		return len(inputs) > 0
	})

	// **Feature: agentic-system-poc, Property 1: Mixed Message Ordering Preservation**
	properties.Property("mixed messages and tool results preserve order", prop.ForAll(
		func(inputs []mixedInput) bool {
			mem := NewConversationMemory()

			// Add all inputs
			for _, input := range inputs {
				if input.IsToolResult {
					mem.AddToolResult(input.ToolCallID, input.ToolName, input.Content)
				} else {
					mem.AddMessage(input.Role, input.Content)
				}
			}

			// Get messages back
			messages := mem.GetMessages()

			// Verify count matches
			if len(messages) != len(inputs) {
				return false
			}

			// Verify order and content preservation
			for i, input := range inputs {
				msg := messages[i]
				if input.IsToolResult {
					if msg.Role != "tool" {
						return false
					}
					if msg.ToolCallID != input.ToolCallID {
						return false
					}
					if msg.ToolName != input.ToolName {
						return false
					}
					if msg.Content != input.Content {
						return false
					}
				} else {
					if msg.Role != input.Role {
						return false
					}
					if msg.Content != input.Content {
						return false
					}
				}
			}

			return true
		},
		genMixedSequence,
	))

	properties.TestingRun(t)
}

func TestProperty_ClearResetsMemory(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// **Feature: agentic-system-poc, Property 1: Clear Resets Memory**
	properties.Property("Clear() removes all messages", prop.ForAll(
		func(inputs []messageInput) bool {
			mem := NewConversationMemory()

			// Add all messages
			for _, input := range inputs {
				mem.AddMessage(input.Role, input.Content)
			}

			// Clear the memory
			mem.Clear()

			// Verify memory is empty
			return mem.Len() == 0 && len(mem.GetMessages()) == 0
		},
		genMessageSequence(),
	))

	properties.TestingRun(t)
}

func TestProperty_GetMessagesReturnsCopy(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// **Feature: agentic-system-poc, Property 1: GetMessages Returns Copy**
	properties.Property("GetMessages returns a copy that doesn't affect original", prop.ForAll(
		func(inputs []messageInput) bool {
			mem := NewConversationMemory()

			// Add all messages
			for _, input := range inputs {
				mem.AddMessage(input.Role, input.Content)
			}

			// Get messages and modify the returned slice
			messages := mem.GetMessages()
			if len(messages) > 0 {
				messages[0].Content = "MODIFIED"
				messages[0].Role = "MODIFIED"
			}

			// Get messages again and verify original is unchanged
			originalMessages := mem.GetMessages()
			for i, input := range inputs {
				if originalMessages[i].Role != input.Role {
					return false
				}
				if originalMessages[i].Content != input.Content {
					return false
				}
			}

			return true
		},
		genMessageSequence(),
	))

	properties.TestingRun(t)
}
