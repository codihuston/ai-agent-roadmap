package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"agentic-poc/internal/provider"
)

// FileWriterTool writes content to files.
type FileWriterTool struct {
	basePath string
}

// NewFileWriterTool creates a new FileWriterTool with the given base path.
// All file paths will be resolved relative to basePath for security.
func NewFileWriterTool(basePath string) *FileWriterTool {
	return &FileWriterTool{basePath: basePath}
}

// Name returns the tool's identifier.
func (f *FileWriterTool) Name() string {
	return "write_file"
}

// Description returns what the tool does.
func (f *FileWriterTool) Description() string {
	return "Writes content to a file at the specified path, creating directories as needed"
}

// Parameters returns the JSON Schema for the tool's input.
func (f *FileWriterTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The path to the file to write (relative to base path)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

// Execute writes content to the file.
func (f *FileWriterTool) Execute(ctx context.Context, args map[string]interface{}) (*provider.ToolResult, error) {
	pathArg, ok := args["path"].(string)
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing or invalid 'path' argument",
		}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return &provider.ToolResult{
			Success: false,
			Error:   "missing or invalid 'content' argument",
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

	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create directory: %v", err),
		}, nil
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		if os.IsPermission(err) {
			return &provider.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("permission denied: %s", pathArg),
			}, nil
		}
		return &provider.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return &provider.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), pathArg),
	}, nil
}
