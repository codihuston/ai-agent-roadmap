package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agentic-poc/internal/memory"
	"agentic-poc/internal/provider"
)

func TestNewCoderAgent(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	basePath := "/tmp/test"

	agent := NewCoderAgent(mockProvider, basePath)

	// Verify agent is created
	if agent == nil {
		t.Fatal("expected agent to be created, got nil")
	}

	// Verify system prompt is set
	if agent.systemPrompt != CoderSystemPrompt {
		t.Error("system prompt not set correctly")
	}

	// Verify tools are registered
	tools := agent.GetTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Verify read_file and write_file tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["read_file"] {
		t.Error("read_file tool not found in agent tools")
	}
	if !toolNames["write_file"] {
		t.Error("write_file tool not found in agent tools")
	}
}

func TestCoderAgent_SystemPromptIncluded(t *testing.T) {
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "I have completed the task."},
		},
	}

	agent := NewCoderAgent(mockProvider, "/tmp/test")
	mem := memory.NewConversationMemory()

	_, err := agent.Run(context.Background(), "Execute the plan", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify system prompt was sent to LLM
	if len(mockProvider.requests) == 0 {
		t.Fatal("expected at least one LLM request")
	}

	if mockProvider.requests[0].SystemPrompt != CoderSystemPrompt {
		t.Error("system prompt not included in LLM request")
	}
}

func TestCoderAgent_ToolDefinitionsIncluded(t *testing.T) {
	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{Text: "Task completed."},
		},
	}

	agent := NewCoderAgent(mockProvider, "/tmp/test")
	mem := memory.NewConversationMemory()

	_, err := agent.Run(context.Background(), "Execute the plan", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool definitions were sent to LLM
	if len(mockProvider.requests) == 0 {
		t.Fatal("expected at least one LLM request")
	}

	tools := mockProvider.requests[0].Tools
	if len(tools) != 2 {
		t.Errorf("expected 2 tool definitions, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["read_file"] {
		t.Error("read_file tool definition not found")
	}
	if !toolNames["write_file"] {
		t.Error("write_file tool definition not found")
	}
}

func TestCoderAgent_ExecutesWriteFile(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "hello.txt",
							"content": "Hello, World!",
						},
					},
				},
			},
			{Text: "I have created the file hello.txt with the greeting."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create a hello.txt file", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the agent completed successfully
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify write_file tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCallsMade))
	}

	if result.ToolCallsMade[0].Name != "write_file" {
		t.Errorf("expected write_file tool call, got %s", result.ToolCallsMade[0].Name)
	}

	// Verify file was actually created
	content, err := os.ReadFile(filepath.Join(tmpDir, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("file content = %q, want %q", string(content), "Hello, World!")
	}
}

func TestCoderAgent_ExecutesReadFile(t *testing.T) {
	// Create a temporary directory with a test file
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file to read
	testContent := "This is test content"
	err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "read_file",
						Arguments: map[string]interface{}{
							"path": "test.txt",
						},
					},
				},
			},
			{Text: "The file contains: This is test content"},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Read the test.txt file", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the agent completed successfully
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify read_file tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCallsMade))
	}

	if result.ToolCallsMade[0].Name != "read_file" {
		t.Errorf("expected read_file tool call, got %s", result.ToolCallsMade[0].Name)
	}
}

func TestCoderAgent_ExecutesMultipleSteps(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "main.go",
							"content": "package main\n\nfunc main() {}",
						},
					},
				},
			},
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_2",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "utils.go",
							"content": "package main\n\nfunc helper() {}",
						},
					},
				},
			},
			{Text: "I have created both main.go and utils.go files."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create main.go and utils.go", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the agent completed successfully
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify both tool calls were made
	if len(result.ToolCallsMade) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(result.ToolCallsMade))
	}

	// Verify both files were created
	if _, err := os.Stat(filepath.Join(tmpDir, "main.go")); os.IsNotExist(err) {
		t.Error("main.go was not created")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "utils.go")); os.IsNotExist(err) {
		t.Error("utils.go was not created")
	}
}

func TestCoderAgent_ReadThenWrite(t *testing.T) {
	// Test a common pattern: read a file, then write an updated version
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial file
	initialContent := "package main\n\nfunc main() {\n\t// TODO: implement\n}"
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "read_file",
						Arguments: map[string]interface{}{
							"path": "main.go",
						},
					},
				},
			},
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_2",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "main.go",
							"content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello!\")\n}",
						},
					},
				},
			},
			{Text: "I have read the file and updated it with the implementation."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Read main.go and implement the TODO", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both tool calls were made
	if len(result.ToolCallsMade) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(result.ToolCallsMade))
	}

	// Verify the file was updated
	content, err := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	expectedContent := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello!\")\n}"
	if string(content) != expectedContent {
		t.Errorf("file content = %q, want %q", string(content), expectedContent)
	}
}

func TestCoderAgent_HandlesFileNotFound(t *testing.T) {
	// Test that the coder agent handles file not found errors gracefully
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "read_file",
						Arguments: map[string]interface{}{
							"path": "nonexistent.txt",
						},
					},
				},
			},
			{Text: "The file does not exist. I will create it instead."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Read nonexistent.txt", mem)

	// Agent should not error - it should handle the tool error gracefully
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify the tool was called
	if len(result.ToolCallsMade) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result.ToolCallsMade))
	}
}

func TestCoderAgent_CreatesNestedDirectories(t *testing.T) {
	// Test that the coder agent can create files in nested directories
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "src/internal/utils/helper.go",
							"content": "package utils\n\nfunc Helper() {}",
						},
					},
				},
			},
			{Text: "I have created the helper.go file in the nested directory structure."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Create src/internal/utils/helper.go", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Verify the file was created in the nested directory
	filePath := filepath.Join(tmpDir, "src", "internal", "utils", "helper.go")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	expectedContent := "package utils\n\nfunc Helper() {}"
	if string(content) != expectedContent {
		t.Errorf("file content = %q, want %q", string(content), expectedContent)
	}
}

func TestCoderAgent_ReturnsActionSummary(t *testing.T) {
	// Validates: Requirements 6.5 - Coder returns action summary
	tmpDir, err := os.MkdirTemp("", "coder_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockProvider := &mockLLMProvider{
		responses: []provider.LLMResponse{
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_1",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "file1.txt",
							"content": "content1",
						},
					},
				},
			},
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_2",
						Name: "read_file",
						Arguments: map[string]interface{}{
							"path": "file1.txt",
						},
					},
				},
			},
			{
				ToolCalls: []provider.ToolCall{
					{
						ID:   "call_3",
						Name: "write_file",
						Arguments: map[string]interface{}{
							"path":    "file2.txt",
							"content": "content2",
						},
					},
				},
			},
			{Text: "Completed all steps."},
		},
	}

	agent := NewCoderAgent(mockProvider, tmpDir)
	mem := memory.NewConversationMemory()

	result, err := agent.Run(context.Background(), "Execute the plan", mem)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all tool calls are tracked in the result
	if len(result.ToolCallsMade) != 3 {
		t.Errorf("expected 3 tool calls in summary, got %d", len(result.ToolCallsMade))
	}

	// Verify the tool call names
	expectedCalls := []string{"write_file", "read_file", "write_file"}
	for i, expected := range expectedCalls {
		if result.ToolCallsMade[i].Name != expected {
			t.Errorf("tool call %d = %s, want %s", i, result.ToolCallsMade[i].Name, expected)
		}
	}
}
