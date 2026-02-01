package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriterTool_Name(t *testing.T) {
	writer := NewFileWriterTool("")
	if writer.Name() != "write_file" {
		t.Errorf("expected name 'write_file', got '%s'", writer.Name())
	}
}

func TestFileWriterTool_Description(t *testing.T) {
	writer := NewFileWriterTool("")
	if writer.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestFileWriterTool_Parameters(t *testing.T) {
	writer := NewFileWriterTool("")
	params := writer.Parameters()

	if params["type"] != "object" {
		t.Error("expected parameters type to be 'object'")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property")
	}
	if _, ok := props["content"]; !ok {
		t.Error("expected 'content' property")
	}
}

func TestFileWriterTool_Execute_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	testContent := "Hello, World!"
	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "test.txt",
		"content": testContent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestFileWriterTool_Execute_OverwriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")

	// Create existing file
	if err := os.WriteFile(testFile, []byte("old content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	newContent := "new content"
	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "existing.txt",
		"content": newContent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != newContent {
		t.Errorf("expected content '%s', got '%s'", newContent, string(content))
	}
}

func TestFileWriterTool_Execute_CreateNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	testContent := "nested content"
	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "subdir/nested/deep/file.txt",
		"content": testContent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "subdir/nested/deep/file.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestFileWriterTool_Execute_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "empty.txt",
		"content": "",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "empty.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "" {
		t.Errorf("expected empty content, got '%s'", string(content))
	}
}

func TestFileWriterTool_Execute_MissingPathArg(t *testing.T) {
	writer := NewFileWriterTool("")
	ctx := context.Background()

	result, err := writer.Execute(ctx, map[string]interface{}{
		"content": "some content",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing path argument")
	}
}

func TestFileWriterTool_Execute_MissingContentArg(t *testing.T) {
	writer := NewFileWriterTool("")
	ctx := context.Background()

	result, err := writer.Execute(ctx, map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing content argument")
	}
}

func TestFileWriterTool_Execute_DirectoryTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	// Try to write file using directory traversal
	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "../../../etc/passwd",
		"content": "malicious content",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for directory traversal attempt")
	}
}

func TestFileWriterTool_Execute_LargeContent(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}

	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "large.txt",
		"content": string(largeContent),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "large.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if len(content) != len(largeContent) {
		t.Errorf("expected %d bytes, got %d bytes", len(largeContent), len(content))
	}
}

func TestFileWriterTool_Execute_SpecialCharactersInContent(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriterTool(tmpDir)
	ctx := context.Background()

	testContent := "Line 1\nLine 2\tTabbed\r\nWindows line\x00Null byte"
	result, err := writer.Execute(ctx, map[string]interface{}{
		"path":    "special.txt",
		"content": testContent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "special.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, string(content))
	}
}

func TestFileWriterTool_ImplementsInterface(t *testing.T) {
	var _ Tool = (*FileWriterTool)(nil)
}
