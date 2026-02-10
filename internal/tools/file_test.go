package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileTool(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewReadFileTool(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"
	fullPath := filepath.Join(tempDir, testFile)

	if err := os.WriteFile(fullPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, result)
	}

	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"path": "nonexistent.txt",
	})
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "FILE_NOT_FOUND" {
		t.Errorf("Expected FILE_NOT_FOUND error, got %v", err)
	}
}

func TestReadFileToolInvalidParams(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewReadFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": 123,
	})
	if err == nil {
		t.Error("Expected error for non-string path parameter")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}

	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"path": "",
	})
	if err == nil {
		t.Error("Expected error for empty path parameter")
	}

	toolErr = nil
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}
}

func TestReadFileToolPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewReadFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "../test.txt",
	})
	if err == nil {
		t.Error("Expected error for path outside base directory")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PATH" {
		t.Errorf("Expected INVALID_PATH error, got %v", err)
	}
}

func TestWriteFileTool(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewWriteFileTool(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    testFile,
		"content": testContent,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	fullPath := filepath.Join(tempDir, testFile)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(data))
	}

	subDir := "subdir/nested"
	subFile := filepath.Join(subDir, "test.txt")
	result, err = tool.Execute(context.Background(), map[string]interface{}{
		"path":    subFile,
		"content": "nested content",
	})
	if err != nil {
		t.Fatalf("Execute failed for nested file: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, subFile)); os.IsNotExist(err) {
		t.Error("Nested file was not created")
	}
}

func TestWriteFileToolInvalidParams(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewWriteFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    123,
		"content": "test",
	})
	if err == nil {
		t.Error("Expected error for non-string path parameter")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}

	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"path":    "test.txt",
		"content": 123,
	})
	if err == nil {
		t.Error("Expected error for non-string content parameter")
	}

	toolErr = nil
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}
}

func TestWriteFileToolPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewWriteFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "../test.txt",
		"content": "test",
	})
	if err == nil {
		t.Error("Expected error for path outside base directory")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PATH" {
		t.Errorf("Expected INVALID_PATH error, got %v", err)
	}
}

func TestListDirTool(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewListDirTool(tempDir)

	testFiles := []string{"file1.txt", "file2.txt", "subdir"}
	for _, name := range testFiles {
		fullPath := filepath.Join(tempDir, name)
		if name == "subdir" {
			if err := os.Mkdir(fullPath, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
		} else {
			if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	for _, name := range testFiles {
		if !contains(result, name) {
			t.Errorf("Result does not contain expected file: %s", name)
		}
	}

	subDirPath := "subdir"
	if err := os.WriteFile(filepath.Join(tempDir, subDirPath, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	result, err = tool.Execute(context.Background(), map[string]interface{}{
		"path": subDirPath,
	})
	if err != nil {
		t.Fatalf("Execute failed for subdirectory: %v", err)
	}

	if !contains(result, "nested.txt") {
		t.Error("Result does not contain nested file")
	}
}

func TestListDirToolEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewListDirTool(tempDir)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !contains(result, "empty") {
		t.Error("Expected result to indicate empty directory")
	}
}

func TestListDirToolNonexistentPath(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewListDirTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "nonexistent",
	})
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "DIR_NOT_FOUND" {
		t.Errorf("Expected DIR_NOT_FOUND error, got %v", err)
	}
}

func TestListDirToolNotADirectory(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewListDirTool(tempDir)

	testFile := "test.txt"
	fullPath := filepath.Join(tempDir, testFile)
	if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	if err == nil {
		t.Error("Expected error when path is not a directory")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "NOT_A_DIRECTORY" {
		t.Errorf("Expected NOT_A_DIRECTORY error, got %v", err)
	}
}

func TestDeleteFileTool(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewDeleteFileTool(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"
	fullPath := filepath.Join(tempDir, testFile)

	if err := os.WriteFile(fullPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("File was not deleted")
	}

	testDir := "testdir"
	dirPath := filepath.Join(tempDir, testDir)
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	result, err = tool.Execute(context.Background(), map[string]interface{}{
		"path": testDir,
	})
	if err != nil {
		t.Fatalf("Execute failed for directory: %v", err)
	}

	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

func TestDeleteFileToolNonexistentPath(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewDeleteFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "nonexistent.txt",
	})
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "FILE_NOT_FOUND" {
		t.Errorf("Expected FILE_NOT_FOUND error, got %v", err)
	}
}

func TestDeleteFileToolInvalidParams(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewDeleteFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": 123,
	})
	if err == nil {
		t.Error("Expected error for non-string path parameter")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}

	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"path": "",
	})
	if err == nil {
		t.Error("Expected error for empty path parameter")
	}

	toolErr = nil
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected INVALID_PARAM error, got %v", err)
	}
}

func TestDeleteFileToolPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	tool := NewDeleteFileTool(tempDir)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "../test.txt",
	})
	if err == nil {
		t.Error("Expected error for path outside base directory")
	}

	var toolErr *ToolError
	if !AsToolError(err, &toolErr) || toolErr.Code != "INVALID_PATH" {
		t.Errorf("Expected INVALID_PATH error, got %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
