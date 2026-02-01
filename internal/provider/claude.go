// Package provider defines the LLM provider abstraction and core data types.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// DefaultClaudeModel is the default Claude model to use.
	DefaultClaudeModel = "claude-sonnet-4-20250514"
	// DefaultClaudeBaseURL is the default Anthropic API base URL.
	DefaultClaudeBaseURL = "https://api.anthropic.com/v1"
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 60 * time.Second
	// AnthropicAPIVersion is the required API version header.
	AnthropicAPIVersion = "2023-06-01"
)

// ClaudeProvider implements LLMProvider for Anthropic's Claude API.
type ClaudeProvider struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// ClaudeOption is a functional option for configuring ClaudeProvider.
type ClaudeOption func(*ClaudeProvider)

// WithModel sets the Claude model to use.
func WithModel(model string) ClaudeOption {
	return func(c *ClaudeProvider) {
		c.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClaudeOption {
	return func(c *ClaudeProvider) {
		c.client = client
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClaudeOption {
	return func(c *ClaudeProvider) {
		c.baseURL = url
	}
}

// NewClaudeProvider creates a new ClaudeProvider.
// It reads the API key from the ANTHROPIC_API_KEY environment variable.
func NewClaudeProvider(opts ...ClaudeOption) (*ClaudeProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY environment variable not set")
	}

	provider := &ClaudeProvider{
		apiKey:  apiKey,
		model:   DefaultClaudeModel,
		client:  &http.Client{Timeout: DefaultTimeout},
		baseURL: DefaultClaudeBaseURL,
	}

	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

// NewClaudeProviderWithKey creates a new ClaudeProvider with an explicit API key.
// This is useful for testing or when the key is provided through other means.
func NewClaudeProviderWithKey(apiKey string, opts ...ClaudeOption) (*ClaudeProvider, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	provider := &ClaudeProvider{
		apiKey:  apiKey,
		model:   DefaultClaudeModel,
		client:  &http.Client{Timeout: DefaultTimeout},
		baseURL: DefaultClaudeBaseURL,
	}

	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

// Name returns the provider name.
func (c *ClaudeProvider) Name() string {
	return "claude"
}

// Generate sends a request to Claude and returns the response.
func (c *ClaudeProvider) Generate(ctx context.Context, req GenerateRequest) (*LLMResponse, error) {
	claudeReq, err := c.buildRequest(req)
	if err != nil {
		return nil, fmt.Errorf("claude: failed to build request: %w", err)
	}

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("claude: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("claude: failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", AnthropicAPIVersion)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, respBody)
	}

	return c.parseResponse(respBody)
}

// claudeRequest represents the request body for Claude API.
type claudeRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []claudeMsg  `json:"messages"`
	Tools     []claudeTool `json:"tools,omitempty"`
}

// claudeMsg represents a message in Claude's format.
type claudeMsg struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

// contentPart represents a content part in Claude's message format.
type contentPart struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
	ID        string                 `json:"id,omitempty"`    // For tool_use blocks
	Name      string                 `json:"name,omitempty"`  // For tool_use blocks
	Input     map[string]interface{} `json:"input,omitempty"` // For tool_use blocks
}

// claudeTool represents a tool definition in Claude's format.
type claudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// claudeResponse represents the response from Claude API.
type claudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []claudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason"`
	StopSequence *string              `json:"stop_sequence"`
	Usage        claudeUsage          `json:"usage"`
}

// claudeContentBlock represents a content block in Claude's response.
type claudeContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// claudeUsage represents token usage in Claude's response.
type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// claudeErrorResponse represents an error response from Claude API.
type claudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// buildRequest converts a GenerateRequest to Claude's API format.
func (c *ClaudeProvider) buildRequest(req GenerateRequest) (*claudeRequest, error) {
	claudeReq := &claudeRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    req.SystemPrompt,
		Messages:  make([]claudeMsg, 0, len(req.Messages)),
	}

	// Convert messages to Claude format
	for _, msg := range req.Messages {
		claudeMsg, err := c.convertMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// Convert tools to Claude format
	for _, tool := range req.Tools {
		claudeReq.Tools = append(claudeReq.Tools, claudeTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.Parameters,
		})
	}

	return claudeReq, nil
}

// convertMessage converts a Message to Claude's message format.
func (c *ClaudeProvider) convertMessage(msg Message) (claudeMsg, error) {
	cm := claudeMsg{
		Role:    msg.Role,
		Content: []contentPart{},
	}

	// Handle tool result messages
	if msg.ToolCallID != "" {
		cm.Role = "user"
		cm.Content = append(cm.Content, contentPart{
			Type:      "tool_result",
			ToolUseID: msg.ToolCallID,
			Content:   msg.Content,
		})
		return cm, nil
	}

	// Handle assistant messages with tool calls
	if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
		// Add text content if present
		if msg.Content != "" {
			cm.Content = append(cm.Content, contentPart{
				Type: "text",
				Text: msg.Content,
			})
		}
		// Add tool_use blocks
		for _, tc := range msg.ToolCalls {
			cm.Content = append(cm.Content, contentPart{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: tc.Arguments,
			})
		}
		return cm, nil
	}

	// Handle regular text messages
	cm.Content = append(cm.Content, contentPart{
		Type: "text",
		Text: msg.Content,
	})

	return cm, nil
}

// parseResponse parses Claude's response into an LLMResponse.
func (c *ClaudeProvider) parseResponse(body []byte) (*LLMResponse, error) {
	var resp claudeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("claude: failed to parse response: %w", err)
	}

	llmResp := &LLMResponse{
		ToolCalls: make([]ToolCall, 0),
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			llmResp.Text += block.Text
		case "tool_use":
			llmResp.ToolCalls = append(llmResp.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	return llmResp, nil
}

// handleErrorResponse creates an appropriate error for non-200 responses.
func (c *ClaudeProvider) handleErrorResponse(statusCode int, body []byte) error {
	var errResp claudeErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("claude: API error (status %d): %s", statusCode, string(body))
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("claude: authentication failed: %s", errResp.Error.Message)
	case http.StatusForbidden:
		return fmt.Errorf("claude: access forbidden: %s", errResp.Error.Message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("claude: rate limit exceeded: %s", errResp.Error.Message)
	case http.StatusBadRequest:
		return fmt.Errorf("claude: bad request: %s", errResp.Error.Message)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("claude: server error (status %d): %s", statusCode, errResp.Error.Message)
	default:
		return fmt.Errorf("claude: API error (status %d): %s", statusCode, errResp.Error.Message)
	}
}
