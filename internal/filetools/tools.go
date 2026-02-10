package filetools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type FileToolsConfig struct {
	Storage      storage.Storage
	AllowedPaths []string
}

type ReadFileTool struct {
	storage storage.Storage
}

func NewReadFileTool(storage storage.Storage) *ReadFileTool {
	return &ReadFileTool{
		storage: storage,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to read"
			}
		},
		"required": ["path"],
		"additionalProperties": false
	}`)
	return params
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	data, err := t.storage.ReadFile(ctx, path)
	if err != nil {
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to read file",
			Err:     err,
		}
	}

	return string(data), nil
}

type WriteFileTool struct {
	storage storage.Storage
}

func NewWriteFileTool(storage storage.Storage) *WriteFileTool {
	return &WriteFileTool{
		storage: storage,
	}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to write"
			},
			"content": {
				"type": "string",
				"description": "The content to write to the file"
			}
		},
		"required": ["path", "content"],
		"additionalProperties": false
	}`)
	return params
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "content parameter must be a string",
		}
	}

	if path == "" {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	err := t.storage.WriteFile(ctx, path, []byte(content))
	if err != nil {
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to write file",
			Err:     err,
		}
	}

	return fmt.Sprintf("Successfully wrote to file: %s", path), nil
}

type ListDirTool struct {
	storage storage.Storage
}

func NewListDirTool(storage storage.Storage) *ListDirTool {
	return &ListDirTool{
		storage: storage,
	}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a directory"
}

func (t *ListDirTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the directory to list (optional, defaults to root)"
			}
		},
		"additionalProperties": false
	}`)
	return params
}

func (t *ListDirTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path := ""
	if p, ok := params["path"].(string); ok {
		path = p
	}

	files, err := t.storage.ListFiles(ctx, path)
	if err != nil {
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to list directory",
			Err:     err,
		}
	}

	if len(files) == 0 {
		return fmt.Sprintf("Directory '%s' is empty or does not exist", path), nil
	}

	output := fmt.Sprintf("Found %d items in '%s':\n\n", len(files), path)
	for i, file := range files {
		output += fmt.Sprintf("%d. %s\n", i+1, file)
	}

	return output, nil
}

type DeleteFileTool struct {
	storage storage.Storage
}

func NewDeleteFileTool(storage storage.Storage) *DeleteFileTool {
	return &DeleteFileTool{
		storage: storage,
	}
}

func (t *DeleteFileTool) Name() string {
	return "delete_file"
}

func (t *DeleteFileTool) Description() string {
	return "Delete a file"
}

func (t *DeleteFileTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to the file to delete"
			}
		},
		"required": ["path"],
		"additionalProperties": false
	}`)
	return params
}

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	err := t.storage.DeleteFile(ctx, path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("File '%s' does not exist", path), nil
		}
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to delete file",
			Err:     err,
		}
	}

	return fmt.Sprintf("Successfully deleted file: %s", path), nil
}

type FileExistsTool struct {
	storage storage.Storage
}

func NewFileExistsTool(storage storage.Storage) *FileExistsTool {
	return &FileExistsTool{
		storage: storage,
	}
}

func (t *FileExistsTool) Name() string {
	return "file_exists"
}

func (t *FileExistsTool) Description() string {
	return "Check if a file or directory exists"
}

func (t *FileExistsTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "The path to check"
			}
		},
		"required": ["path"],
		"additionalProperties": false
	}`)
	return params
}

func (t *FileExistsTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter must be a string",
		}
	}

	if path == "" {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "path parameter cannot be empty",
		}
	}

	exists, err := t.storage.FileExists(ctx, path)
	if err != nil {
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to check file existence",
			Err:     err,
		}
	}

	if exists {
		return fmt.Sprintf("File '%s' exists", path), nil
	}
	return fmt.Sprintf("File '%s' does not exist", path), nil
}

func NewFileTools(storage storage.Storage) []tools.Tool {
	return []tools.Tool{
		NewReadFileTool(storage),
		NewWriteFileTool(storage),
		NewListDirTool(storage),
		NewDeleteFileTool(storage),
		NewFileExistsTool(storage),
	}
}

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size,omitempty"`
}

func GetFileInfo(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Name:  filepath.Base(path),
		Path:  path,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}, nil
}