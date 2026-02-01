// Package provider defines the LLM provider abstraction and core data types.
package provider

// Message represents a single message in a conversation.
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
}

// ToolCall represents a request from the LLM to execute a tool.
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// LLMResponse represents a response from an LLM provider.
type LLMResponse struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// HasToolCalls returns true if the response contains tool calls.
func (r *LLMResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// ToolDefinition defines a tool that can be used by the LLM.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GenerateRequest represents a request to generate a response from an LLM.
type GenerateRequest struct {
	Messages     []Message        `json:"messages"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
	SystemPrompt string           `json:"system_prompt,omitempty"`
}
