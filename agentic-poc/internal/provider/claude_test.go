package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClaudeProvider(t *testing.T) {
	tests := []struct {
		name      string
		envKey    string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "success with API key set",
			envKey:  "test-api-key",
			wantErr: false,
		},
		{
			name:      "error when API key not set",
			envKey:    "",
			wantErr:   true,
			errSubstr: "ANTHROPIC_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			original := os.Getenv("ANTHROPIC_API_KEY")
			defer os.Setenv("ANTHROPIC_API_KEY", original)

			if tt.envKey != "" {
				os.Setenv("ANTHROPIC_API_KEY", tt.envKey)
			} else {
				os.Unsetenv("ANTHROPIC_API_KEY")
			}

			provider, err := NewClaudeProvider()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("expected provider, got nil")
				return
			}

			if provider.Name() != "claude" {
				t.Errorf("expected name 'claude', got %q", provider.Name())
			}
		})
	}
}

func TestNewClaudeProviderWithKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "success with valid key",
			apiKey:  "test-api-key",
			wantErr: false,
		},
		{
			name:      "error with empty key",
			apiKey:    "",
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewClaudeProviderWithKey(tt.apiKey)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("expected provider, got nil")
			}
		})
	}
}

func TestClaudeProviderOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}
	customModel := "claude-3-opus-20240229"
	customURL := "https://custom.api.com"

	provider, err := NewClaudeProviderWithKey("test-key",
		WithHTTPClient(customClient),
		WithModel(customModel),
		WithBaseURL(customURL),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.client != customClient {
		t.Error("custom HTTP client not set")
	}

	if provider.model != customModel {
		t.Errorf("expected model %q, got %q", customModel, provider.model)
	}

	if provider.baseURL != customURL {
		t.Errorf("expected baseURL %q, got %q", customURL, provider.baseURL)
	}
}

func TestClaudeProviderGenerate_TextResponse(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Error("missing or incorrect x-api-key header")
		}
		if r.Header.Get("anthropic-version") != AnthropicAPIVersion {
			t.Error("missing or incorrect anthropic-version header")
		}

		// Return mock response
		resp := claudeResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []claudeContentBlock{
				{Type: "text", Text: "Hello! How can I help you today?"},
			},
			Usage: claudeUsage{InputTokens: 10, OutputTokens: 8},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := GenerateRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Text != "Hello! How can I help you today?" {
		t.Errorf("unexpected response text: %q", resp.Text)
	}

	if resp.HasToolCalls() {
		t.Error("expected no tool calls")
	}
}

func TestClaudeProviderGenerate_ToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			ID:         "msg_456",
			Type:       "message",
			Role:       "assistant",
			StopReason: "tool_use",
			Content: []claudeContentBlock{
				{Type: "text", Text: "I'll calculate that for you."},
				{
					Type:  "tool_use",
					ID:    "toolu_123",
					Name:  "calculator",
					Input: map[string]interface{}{"operation": "add", "a": float64(5), "b": float64(3)},
				},
			},
			Usage: claudeUsage{InputTokens: 20, OutputTokens: 15},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := GenerateRequest{
		Messages: []Message{
			{Role: "user", Content: "What is 5 + 3?"},
		},
		Tools: []ToolDefinition{
			{
				Name:        "calculator",
				Description: "Performs arithmetic operations",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"operation": map[string]interface{}{"type": "string"},
						"a":         map[string]interface{}{"type": "number"},
						"b":         map[string]interface{}{"type": "number"},
					},
				},
			},
		},
	}

	resp, err := provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HasToolCalls() {
		t.Fatal("expected tool calls")
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_123" {
		t.Errorf("expected tool call ID 'toolu_123', got %q", tc.ID)
	}
	if tc.Name != "calculator" {
		t.Errorf("expected tool name 'calculator', got %q", tc.Name)
	}
	if tc.Arguments["operation"] != "add" {
		t.Errorf("expected operation 'add', got %v", tc.Arguments["operation"])
	}
}

func TestClaudeProviderGenerate_WithSystemPrompt(t *testing.T) {
	var receivedReq claudeRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := claudeResponse{
			ID:         "msg_789",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []claudeContentBlock{
				{Type: "text", Text: "I am a helpful assistant."},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := GenerateRequest{
		SystemPrompt: "You are a helpful assistant.",
		Messages: []Message{
			{Role: "user", Content: "Who are you?"},
		},
	}

	_, err = provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.System != "You are a helpful assistant." {
		t.Errorf("system prompt not set correctly: %q", receivedReq.System)
	}
}

func TestClaudeProviderGenerate_ToolResultMessage(t *testing.T) {
	var receivedReq claudeRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := claudeResponse{
			ID:         "msg_abc",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []claudeContentBlock{
				{Type: "text", Text: "The result is 8."},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := GenerateRequest{
		Messages: []Message{
			{Role: "user", Content: "What is 5 + 3?"},
			{Role: "assistant", Content: "I'll calculate that."},
			{Role: "user", Content: "8", ToolCallID: "toolu_123", ToolName: "calculator"},
		},
	}

	_, err = provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool result was converted correctly
	if len(receivedReq.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(receivedReq.Messages))
	}

	toolResultMsg := receivedReq.Messages[2]
	if toolResultMsg.Role != "user" {
		t.Errorf("tool result message should have role 'user', got %q", toolResultMsg.Role)
	}

	if len(toolResultMsg.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(toolResultMsg.Content))
	}

	if toolResultMsg.Content[0].Type != "tool_result" {
		t.Errorf("expected content type 'tool_result', got %q", toolResultMsg.Content[0].Type)
	}

	if toolResultMsg.Content[0].ToolUseID != "toolu_123" {
		t.Errorf("expected tool_use_id 'toolu_123', got %q", toolResultMsg.Content[0].ToolUseID)
	}
}

func TestClaudeProviderGenerate_ErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   claudeErrorResponse
		errSubstr  string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response: claudeErrorResponse{
				Type: "error",
				Error: struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{Type: "authentication_error", Message: "Invalid API key"},
			},
			errSubstr: "authentication failed",
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			response: claudeErrorResponse{
				Type: "error",
				Error: struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{Type: "rate_limit_error", Message: "Rate limit exceeded"},
			},
			errSubstr: "rate limit exceeded",
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response: claudeErrorResponse{
				Type: "error",
				Error: struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{Type: "invalid_request_error", Message: "Invalid request"},
			},
			errSubstr: "bad request",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response: claudeErrorResponse{
				Type: "error",
				Error: struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{Type: "api_error", Message: "Internal server error"},
			},
			errSubstr: "server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}

			req := GenerateRequest{
				Messages: []Message{{Role: "user", Content: "Hello"}},
			}

			_, err = provider.Generate(context.Background(), req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !containsSubstring(err.Error(), tt.errSubstr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
			}

			// Verify error is wrapped with "claude:" prefix
			if !containsSubstring(err.Error(), "claude:") {
				t.Errorf("error should be wrapped with 'claude:' prefix: %q", err.Error())
			}
		})
	}
}

func TestClaudeProviderGenerate_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := GenerateRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	_, err = provider.Generate(ctx, req)
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestClaudeProviderGenerate_MultipleTextBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			ID:         "msg_multi",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []claudeContentBlock{
				{Type: "text", Text: "First part. "},
				{Type: "text", Text: "Second part."},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewClaudeProviderWithKey("test-api-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := GenerateRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "First part. Second part."
	if resp.Text != expected {
		t.Errorf("expected %q, got %q", expected, resp.Text)
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
