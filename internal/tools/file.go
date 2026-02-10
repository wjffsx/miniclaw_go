package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct {
	basePath string
}

func NewReadFileTool(basePath string) *ReadFileTool {
	return &ReadFileTool{
		basePath: basePath,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Returns the file content as a string."
}

func (t *ReadFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to read, relative to the base directory"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	fullPath := filepath.Join(t.basePath, path)

	if err := validatePath(t.basePath, fullPath); err != nil {
		return "", &ToolError{
			Code:    "INVALID_PATH",
			Message: fmt.Sprintf("invalid file path: %v", err),
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &ToolError{
				Code:    "FILE_NOT_FOUND",
				Message: fmt.Sprintf("file not found: %s", path),
			}
		}
		return "", &ToolError{
			Code:    "READ_FAILED",
			Message: fmt.Sprintf("failed to read file: %v", err),
		}
	}

	return string(data), nil
}

type WriteFileTool struct {
	basePath string
}

func NewWriteFileTool(basePath string) *WriteFileTool {
	return &WriteFileTool{
		basePath: basePath,
	}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file. Creates the file and any necessary directories if they don't exist."
}

func (t *WriteFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to write, relative to the base directory"
			},
			"content": {
				"type": "string",
				"description": "The content to write to the file"
			}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "content parameter must be a string",
		}
	}

	fullPath := filepath.Join(t.basePath, path)

	if err := validatePath(t.basePath, fullPath); err != nil {
		return "", &ToolError{
			Code:    "INVALID_PATH",
			Message: fmt.Sprintf("invalid file path: %v", err),
		}
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", &ToolError{
			Code:    "WRITE_FAILED",
			Message: fmt.Sprintf("failed to create directory: %v", err),
		}
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", &ToolError{
			Code:    "WRITE_FAILED",
			Message: fmt.Sprintf("failed to write file: %v", err),
		}
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

type ListDirTool struct {
	basePath string
}

func NewListDirTool(basePath string) *ListDirTool {
	return &ListDirTool{
		basePath: basePath,
	}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a given path. Returns a list of file names and their types."
}

func (t *ListDirTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to list, relative to the base directory. Defaults to the base directory if not provided."
			}
		}
	}`)
}

func (t *ListDirTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path := ""
	if p, ok := params["path"].(string); ok {
		path = p
	}

	fullPath := t.basePath
	if path != "" {
		fullPath = filepath.Join(t.basePath, path)
	}

	if err := validatePath(t.basePath, fullPath); err != nil {
		return "", &ToolError{
			Code:    "INVALID_PATH",
			Message: fmt.Sprintf("invalid directory path: %v", err),
		}
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &ToolError{
				Code:    "DIR_NOT_FOUND",
				Message: fmt.Sprintf("directory not found: %s", path),
			}
		}
		return "", &ToolError{
			Code:    "LIST_FAILED",
			Message: fmt.Sprintf("failed to access path: %v", err),
		}
	}

	if !info.IsDir() {
		return "", &ToolError{
			Code:    "NOT_A_DIRECTORY",
			Message: fmt.Sprintf("path is not a directory: %s", path),
		}
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return "", &ToolError{
			Code:    "LIST_FAILED",
			Message: fmt.Sprintf("failed to list directory: %v", err),
		}
	}

	if len(entries) == 0 {
		return fmt.Sprintf("Directory is empty: %s", path), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Contents of %s:\n\n", path))

	for _, entry := range entries {
		entryType := "file"
		if entry.IsDir() {
			entryType = "dir"
		}
		output.WriteString(fmt.Sprintf("  [%s] %s\n", entryType, entry.Name()))
	}

	output.WriteString(fmt.Sprintf("\nTotal: %d items", len(entries)))

	return output.String(), nil
}

type DeleteFileTool struct {
	basePath string
}

func NewDeleteFileTool(basePath string) *DeleteFileTool {
	return &DeleteFileTool{
		basePath: basePath,
	}
}

func (t *DeleteFileTool) Name() string {
	return "delete_file"
}

func (t *DeleteFileTool) Description() string {
	return "Delete a file or directory. Use with caution as this operation cannot be undone."
}

func (t *DeleteFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file or directory to delete, relative to the base directory"
			}
		},
		"required": ["path"]
	}`)
}

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	fullPath := filepath.Join(t.basePath, path)

	if err := validatePath(t.basePath, fullPath); err != nil {
		return "", &ToolError{
			Code:    "INVALID_PATH",
			Message: fmt.Sprintf("invalid file path: %v", err),
		}
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &ToolError{
				Code:    "FILE_NOT_FOUND",
				Message: fmt.Sprintf("file not found: %s", path),
			}
		}
		return "", &ToolError{
			Code:    "DELETE_FAILED",
			Message: fmt.Sprintf("failed to access path: %v", err),
		}
	}

	var deleteErr error
	if info.IsDir() {
		deleteErr = os.RemoveAll(fullPath)
	} else {
		deleteErr = os.Remove(fullPath)
	}

	if deleteErr != nil {
		return "", &ToolError{
			Code:    "DELETE_FAILED",
			Message: fmt.Sprintf("failed to delete: %v", deleteErr),
		}
	}

	return fmt.Sprintf("Successfully deleted: %s", path), nil
}

func validatePath(basePath, fullPath string) error {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return err
	}

	absFull, err := filepath.Abs(fullPath)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(absBase, absFull)
	if err != nil {
		return err
	}

	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path is outside base directory")
	}

	return nil
}