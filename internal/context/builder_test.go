package context

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestNewBuilder(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Storage:       storage.NewFileStorage(tempDir),
		MemoryStorage: storage.NewFileSystemMemoryStorage(filepath.Join(tempDir, "memory")),
	}
	
	builder := NewBuilder(config)
	if builder == nil {
		t.Fatal("NewBuilder returned nil")
	}
	
	if builder.storage == nil {
		t.Error("storage is nil")
	}
	
	if builder.memoryStorage == nil {
		t.Error("memoryStorage is nil")
	}
}

func TestBuilder_Build(t *testing.T) {
	tempDir := t.TempDir()
	
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	
	soulContent := []byte("# Soul\nYou are a helpful AI assistant.")
	if err := os.WriteFile(filepath.Join(configDir, "SOUL.md"), soulContent, 0644); err != nil {
		t.Fatalf("Failed to write SOUL.md: %v", err)
	}
	
	userContent := []byte("# User\nUser preferences and instructions.")
	if err := os.WriteFile(filepath.Join(configDir, "USER.md"), userContent, 0644); err != nil {
		t.Fatalf("Failed to write USER.md: %v", err)
	}
	
	config := &Config{
		Storage:       storage.NewFileStorage(tempDir),
		MemoryStorage: storage.NewFileSystemMemoryStorage(filepath.Join(tempDir, "memory")),
	}
	
	builder := NewBuilder(config)
	
	toolSchemas := []tools.ToolSchema{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters:  []byte(`{"type": "object"}`),
		},
	}
	
	ctx := context.Background()
	result, err := builder.Build(ctx, toolSchemas)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Build returned nil")
	}
	
	if result.SystemPrompt == "" {
		t.Error("SystemPrompt is empty")
	}
	
	if !contains(result.SystemPrompt, "You are a helpful AI assistant") {
		t.Error("SystemPrompt does not contain expected content")
	}
	
	if !contains(result.SystemPrompt, "User preferences and instructions") {
		t.Error("SystemPrompt does not contain USER.md content")
	}
	
	if len(result.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(result.Tools))
	}
	
	if result.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", result.Tools[0].Name)
	}
}

func TestBuilder_BuildSystemPrompt(t *testing.T) {
	ctx := &Context{
		SystemPrompt: "You are a helpful AI assistant.",
		Memory:       "Some memory content",
		DailyNotes: []string{
			"## 2024-01-01\nNote 1",
			"## 2024-01-02\nNote 2",
		},
	}
	
	toolSchemas := []tools.ToolSchema{
		{
			Name:        "tool1",
			Description: "Tool 1 description",
			Parameters:  []byte(`{"type": "object"}`),
		},
	}
	
	prompt := ctx.BuildSystemPrompt(toolSchemas)
	
	if !contains(prompt, "You are a helpful AI assistant.") {
		t.Error("SystemPrompt not included in prompt")
	}
	
	if !contains(prompt, "Some memory content") {
		t.Error("Memory not included in prompt")
	}
	
	if !contains(prompt, "## 2024-01-01") {
		t.Error("Daily notes not included in prompt")
	}
	
	if !contains(prompt, "## Available Tools") {
		t.Error("Available Tools section not included in prompt")
	}
	
	if !contains(prompt, "tool1") {
		t.Error("Tool name not included in prompt")
	}
	
	if !contains(prompt, "Tool 1 description") {
		t.Error("Tool description not included in prompt")
	}
}

func TestBuilder_BuildSystemPrompt_NoTools(t *testing.T) {
	ctx := &Context{
		SystemPrompt: "You are a helpful AI assistant.",
		Memory:       "",
		DailyNotes:   []string{},
	}
	
	toolSchemas := []tools.ToolSchema{}
	
	prompt := ctx.BuildSystemPrompt(toolSchemas)
	
	if !contains(prompt, "You are a helpful AI assistant.") {
		t.Error("SystemPrompt not included in prompt")
	}
	
	if contains(prompt, "## Available Tools") {
		t.Error("Available Tools section should not be included when no tools")
	}
}

func TestBuilder_GetTokenEstimate(t *testing.T) {
	ctx := &Context{
		SystemPrompt: "You are a helpful AI assistant.",
		Memory:       "Some memory content",
		DailyNotes: []string{
			"## 2024-01-01\nNote 1",
		},
		Tools: []tools.ToolSchema{
			{
				Name:        "tool1",
				Description: "Tool 1 description",
				Parameters:  []byte(`{"type": "object"}`),
			},
		},
	}
	
	estimate := ctx.GetTokenEstimate()
	
	if estimate <= 0 {
		t.Errorf("Token estimate should be positive, got %d", estimate)
	}
}

func TestBuilder_loadDailyNotes(t *testing.T) {
	tempDir := t.TempDir()
	
	memoryStorage := storage.NewFileSystemMemoryStorage(filepath.Join(tempDir, "memory"))
	
	config := &Config{
		Storage:       storage.NewFileStorage(tempDir),
		MemoryStorage: memoryStorage,
	}
	
	builder := NewBuilder(config)
	
	ctx := context.Background()
	
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	
	if err := memoryStorage.SetDailyNote(ctx, today, "Today's note"); err != nil {
		t.Fatalf("Failed to set today's note: %v", err)
	}
	
	if err := memoryStorage.SetDailyNote(ctx, yesterday, "Yesterday's note"); err != nil {
		t.Fatalf("Failed to set yesterday's note: %v", err)
	}
	
	result := &Context{}
	err := builder.loadDailyNotes(ctx, result)
	if err != nil {
		t.Fatalf("loadDailyNotes failed: %v", err)
	}
	
	if len(result.DailyNotes) < 2 {
		t.Errorf("Expected at least 2 daily notes, got %d", len(result.DailyNotes))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}