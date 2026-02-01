package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"agentic-poc/internal/provider"
)

// FileReaderTool reads content from files.
type FileReaderTool struct {
	basePath string
}

// NewFileReaderTool creates a new FileReaderTool with the given base path.
// All file paths will be resolved relative to basePath for security.
func NewFileReaderTool(basePath string) *FileReaderTool {
	return &FileReaderTool{basePath: basePath}
}

// Name returns the tool's identifier.
func (f *FileReaderTool) Name() string {
	return "read_file"
}

// Description returns what the tool does.
func (f *FileReaderTool) Description() string {
	return "Reads the content of a file at the specified path"
}

// Parameters returns the JSON Schema for the tool's input.
func (f *FileReaderTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to read (relative to base path)",
			},
		},
		"required": []string{"path"},
	}
}

// Execute reads the file and returns its content.
func (f *FileReaderTool) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	pathArg, ok := args["path"].(string)
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing or invalid 'path' argument",
		}, nil
	}

	// Resolve the full path
	fullPath := filepath.Join(f.basePath, pathArg)

	// Clean the path to prevent directory traversal attacks
	fullPath = filepath.Clean(fullPath)

	// Verify the path is still within basePath
	if f.basePath != "" {
		absBase, err := filepath.Abs(f.basePath)
		if err != nil {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to resolve base path: %v", err),
			}, nil
		}
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to resolve file path: %v", err),
			}, nil
		}

		// Check that the resolved path starts with the base path
		rel, err := filepath.Rel(absBase, absPath)
		if err != nil || len(rel) > 0 && rel[0] == '.' {
			return &provider.ToolResult{
				Success: false,
				Error:   "path escapes base directory",
			}, nil
		}
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("file not found: %s", pathArg),
			}, nil
		}
		if os.IsPermission(err) {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("permission denied: %s", pathArg),
			}, nil
		}
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	return &provider.ToolResult{
		Success: true,
		Output:  string(content),
	}, nil
}
