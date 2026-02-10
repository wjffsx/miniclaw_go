package filetools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestNewReadFileTool(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewReadFileTool(fileStorage)
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", tool.Name())
	}
}

func TestReadFileTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"

	err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, result)
	}
}

func TestReadFileTool_Execute_MissingPath(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewReadFileTool(fileStorage)

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing path, got nil")
	}

	var toolErr *tools.ToolError
	if !tools.AsToolError(err, &toolErr) {
		t.Error("Expected ToolError")
	}

	if toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected error code 'INVALID_PARAM', got '%s'", toolErr.Code)
	}
}

func TestReadFileTool_Execute_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewReadFileTool(fileStorage)

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"path": "",
	})
	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}

	var toolErr *tools.ToolError
	if !tools.AsToolError(err, &toolErr) {
		t.Error("Expected ToolError")
	}

	if toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected error code 'INVALID_PARAM', got '%s'", toolErr.Code)
	}
}

func TestNewWriteFileTool(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewWriteFileTool(fileStorage)
	if tool.Name() != "write_file" {
		t.Errorf("Expected name 'write_file', got '%s'", tool.Name())
	}
}

func TestWriteFileTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"

	tool := NewWriteFileTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path":    testFile,
		"content": testContent,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	data, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(data))
	}
}

func TestWriteFileTool_Execute_MissingPath(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewWriteFileTool(fileStorage)

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"content": "test",
	})
	if err == nil {
		t.Error("Expected error for missing path, got nil")
	}

	var toolErr *tools.ToolError
	if !tools.AsToolError(err, &toolErr) {
		t.Error("Expected ToolError")
	}

	if toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected error code 'INVALID_PARAM', got '%s'", toolErr.Code)
	}
}

func TestNewListDirTool(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewListDirTool(fileStorage)
	if tool.Name() != "list_dir" {
		t.Errorf("Expected name 'list_dir', got '%s'", tool.Name())
	}
}

func TestListDirTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, file := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, file), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tool := NewListDirTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	for _, file := range testFiles {
		if !contains(result, file) {
			t.Errorf("Result does not contain file '%s'", file)
		}
	}
}

func TestNewDeleteFileTool(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewDeleteFileTool(fileStorage)
	if tool.Name() != "delete_file" {
		t.Errorf("Expected name 'delete_file', got '%s'", tool.Name())
	}
}

func TestDeleteFileTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFile := "test.txt"
	err := os.WriteFile(filepath.Join(tempDir, testFile), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewDeleteFileTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	_, err = os.Stat(filepath.Join(tempDir, testFile))
	if !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestNewFileExistsTool(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewFileExistsTool(fileStorage)
	if tool.Name() != "file_exists" {
		t.Errorf("Expected name 'file_exists', got '%s'", tool.Name())
	}
}

func TestFileExistsTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFile := "test.txt"
	err := os.WriteFile(filepath.Join(tempDir, testFile), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewFileExistsTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	if !contains(result, "exists") {
		t.Error("Result should indicate file exists")
	}
}

func TestFileExistsTool_Execute_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tool := NewFileExistsTool(fileStorage)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path": "nonexistent.txt",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	if !contains(result, "does not exist") {
		t.Error("Result should indicate file does not exist")
	}
}

func TestNewFileTools(t *testing.T) {
	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	tools := NewFileTools(fileStorage)

	if len(tools) != 5 {
		t.Errorf("Expected 5 tools, got %d", len(tools))
	}

	toolNames := []string{"read_file", "write_file", "list_dir", "delete_file", "file_exists"}
	for i, tool := range tools {
		if tool.Name() != toolNames[i] {
			t.Errorf("Expected tool name '%s', got '%s'", toolNames[i], tool.Name())
		}
	}
}

func TestGetFileInfo(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, err := GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", info.Name)
	}

	if info.IsDir {
		t.Error("File should not be a directory")
	}

	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
