package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileReaderTool_Name(t *testing.T) {
	reader := NewFileReaderTool("")
	if reader.Name() != "read_file" {
		t.Errorf("expected name 'read_file', got '%s'", reader.Name())
	}
}

func TestFileReaderTool_Description(t *testing.T) {
	reader := NewFileReaderTool("")
	if reader.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestFileReaderTool_Parameters(t *testing.T) {
	reader := NewFileReaderTool("")
	params := reader.Parameters()

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
}

func TestFileReaderTool_Execute_ExistingFile(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	testContent := "Hello, World!\nThis is a test file."
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Output != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, result.Output)
	}
}

func TestFileReaderTool_Execute_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "nonexistent.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing file")
	}
	if result.Error == "" {
		t.Error("expected error message for missing file")
	}
}

func TestFileReaderTool_Execute_PermissionError(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "noperm.txt")

	// Create file with no read permissions
	if err := os.WriteFile(testFile, []byte("secret"), 0000); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Restore permissions for cleanup

	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "noperm.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for permission denied")
	}
	if result.Error == "" {
		t.Error("expected error message for permission denied")
	}
}

func TestFileReaderTool_Execute_MissingPathArg(t *testing.T) {
	reader := NewFileReaderTool("")
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for missing path argument")
	}
}

func TestFileReaderTool_Execute_DirectoryTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file outside the base path
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret data"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	// Try to read file using directory traversal
	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "../../../" + filepath.Base(outsideDir) + "/secret.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for directory traversal attempt")
	}
}

func TestFileReaderTool_Execute_NestedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	testContent := "nested file content"
	testFile := filepath.Join(nestedDir, "file.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "subdir/nested/file.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Output != testContent {
		t.Errorf("expected content '%s', got '%s'", testContent, result.Output)
	}
}

func TestFileReaderTool_Execute_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	reader := NewFileReaderTool(tmpDir)
	ctx := context.Background()

	result, err := reader.Execute(ctx, map[string]interface{}{
		"path": "empty.txt",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Output != "" {
		t.Errorf("expected empty content, got '%s'", result.Output)
	}
}

func TestFileReaderTool_ImplementsInterface(t *testing.T) {
	var _ Tool = (*FileReaderTool)(nil)
}
