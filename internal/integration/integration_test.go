package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/agent"
	"github.com/wjffsx/miniclaw_go/internal/bus"
	"github.com/wjffsx/miniclaw_go/internal/config"
	agentcontext "github.com/wjffsx/miniclaw_go/internal/context"
	"github.com/wjffsx/miniclaw_go/internal/llm"
	"github.com/wjffsx/miniclaw_go/internal/memory"
	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestAgentIntegration(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)
	sessionStorage := storage.NewFileSystemSessionStorage(tempDir)
	memoryStorage := storage.NewFileSystemMemoryStorage(tempDir)

	messageBus := bus.NewInMemoryMessageBus(ctx)

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-5",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
	}

	toolRegistry := tools.NewToolRegistry()

	getTimeTool := tools.NewGetTimeTool()
	if err := toolRegistry.Register(getTimeTool); err != nil {
		t.Fatalf("Failed to register get_time tool: %v", err)
	}

	echoTool := tools.NewEchoTool()
	if err := toolRegistry.Register(echoTool); err != nil {
		t.Fatalf("Failed to register echo tool: %v", err)
	}

	memoryManager := memory.NewManager(memoryStorage)
	memoryTools := memory.NewMemoryTools(memoryManager)
	for _, memTool := range memoryTools {
		if err := toolRegistry.Register(memTool); err != nil {
			t.Fatalf("Failed to register %s tool: %v", memTool.Name(), err)
		}
	}

	llmModels := []*llm.ModelConfig{
		{
			Name:        "default",
			Provider:    "anthropic",
			APIKey:      "test-api-key",
			Model:       cfg.LLM.Model,
			MaxTokens:   cfg.LLM.MaxTokens,
			Temperature: cfg.LLM.Temperature,
		},
	}

	agentConfig := &agent.Config{
		LLMModels:      llmModels,
		DefaultModel:   "default",
		SessionStorage: sessionStorage,
		MemoryStorage:  memoryStorage,
		Storage:        fileStorage,
		ToolRegistry:   toolRegistry,
		MaxIterations:  10,
	}

	agentService, err := agent.NewAgent(agentConfig, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agentService == nil {
		t.Fatal("Failed to create agent")
	}

	if len(toolRegistry.List()) == 0 {
		t.Error("No tools registered")
	}

	t.Log("Agent integration test passed")
}

func TestMemoryIntegration(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	memoryStorage := storage.NewFileSystemMemoryStorage(tempDir)

	memoryManager := memory.NewManager(memoryStorage)

	testMemory := "This is a test memory entry"
	testEntry := &memory.MemoryEntry{
		Content:   testMemory,
		Timestamp: time.Now(),
		Type:      "test",
	}
	if err := memoryManager.AddMemoryEntry(ctx, testEntry); err != nil {
		t.Fatalf("Failed to add memory: %v", err)
	}

	retrievedMemory, err := memoryManager.GetMemory(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve memory: %v", err)
	}

	if retrievedMemory == "" {
		t.Error("Retrieved memory is empty")
	}

	if !strings.Contains(retrievedMemory, testMemory) {
		t.Errorf("Retrieved memory does not contain expected content")
	}

	t.Log("Memory integration test passed")
}

func TestToolExecutionIntegration(t *testing.T) {
	ctx := context.Background()

	toolRegistry := tools.NewToolRegistry()

	getTimeTool := tools.NewGetTimeTool()
	if err := toolRegistry.Register(getTimeTool); err != nil {
		t.Fatalf("Failed to register get_time tool: %v", err)
	}

	echoTool := tools.NewEchoTool()
	if err := toolRegistry.Register(echoTool); err != nil {
		t.Fatalf("Failed to register echo tool: %v", err)
	}

	toolExecutor := tools.NewToolExecutor(toolRegistry)

	result, err := toolExecutor.Execute(ctx, "get_time", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to execute get_time tool: %v", err)
	}

	if result == nil {
		t.Error("get_time tool result is nil")
	}

	if result.Error != "" {
		t.Errorf("get_time tool returned error: %s", result.Error)
	}

	echoResult, err := toolExecutor.Execute(ctx, "echo", map[string]interface{}{
		"message": "test message",
	})
	if err != nil {
		t.Fatalf("Failed to execute echo tool: %v", err)
	}

	if echoResult == nil {
		t.Error("echo tool result is nil")
	}

	if echoResult.Result != "Echo: test message" {
		t.Errorf("Expected echo result 'Echo: test message', got '%s'", echoResult.Result)
	}

	t.Log("Tool execution integration test passed")
}

func TestStorageIntegration(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)

	testFile := "test.txt"
	testContent := "Hello, World!"

	err := fileStorage.WriteFile(ctx, testFile, []byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	data, err := fileStorage.ReadFile(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(data))
	}

	files, err := fileStorage.ListFiles(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) == 0 {
		t.Error("No files found")
	}

	err = fileStorage.DeleteFile(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	exists, err := fileStorage.FileExists(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}

	if exists {
		t.Error("File should not exist after deletion")
	}

	t.Log("Storage integration test passed")
}

func TestContextBuilderIntegration(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	fileStorage := storage.NewFileStorage(tempDir)
	memoryStorage := storage.NewFileSystemMemoryStorage(tempDir)

	soulContent := "You are a helpful AI assistant."
	if err := fileStorage.WriteFile(ctx, "config/SOUL.md", []byte(soulContent)); err != nil {
		t.Fatalf("Failed to write SOUL.md: %v", err)
	}

	userContent := "User preferences and instructions."
	if err := fileStorage.WriteFile(ctx, "config/USER.md", []byte(userContent)); err != nil {
		t.Fatalf("Failed to write USER.md: %v", err)
	}

	memoryManager := memory.NewManager(memoryStorage)

	testMemory := "User prefers Python programming"
	testEntry := &memory.MemoryEntry{
		Content:   testMemory,
		Timestamp: time.Now(),
		Type:      "preference",
	}
	if err := memoryManager.AddMemoryEntry(ctx, testEntry); err != nil {
		t.Fatalf("Failed to add memory: %v", err)
	}

	toolRegistry := tools.NewToolRegistry()

	getTimeTool := tools.NewGetTimeTool()
	if err := toolRegistry.Register(getTimeTool); err != nil {
		t.Fatalf("Failed to register get_time tool: %v", err)
	}

	contextBuilder := agentcontext.NewBuilder(&agentcontext.Config{
		Storage:       fileStorage,
		MemoryStorage: memoryStorage,
	})

	toolSchemas := toolRegistry.GetSchemas()
	builtContext, err := contextBuilder.Build(ctx, toolSchemas)
	if err != nil {
		t.Fatalf("Failed to build context: %v", err)
	}

	if builtContext == nil {
		t.Error("Built context is nil")
	}

	if builtContext.SystemPrompt == "" {
		t.Error("System prompt is empty")
	}

	if len(builtContext.Tools) == 0 {
		t.Error("No tools in context")
	}

	t.Log("Context builder integration test passed")
}

func TestMultiModelManagerIntegration(t *testing.T) {
	llmModels := []*llm.ModelConfig{
		{
			Name:        "anthropic",
			Provider:    "anthropic",
			APIKey:      "test-key-1",
			Model:       "claude-3-5-sonnet",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		{
			Name:        "openai",
			Provider:    "openai",
			APIKey:      "test-key-2",
			Model:       "gpt-4",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
	}

	multiModelManager, err := llm.NewMultiModelManager(llmModels, "anthropic")
	if err != nil {
		t.Fatalf("Failed to create multi-model manager: %v", err)
	}

	models := multiModelManager.ListModels()
	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	currentModel := multiModelManager.GetModel()
	if currentModel != "claude-3-5-sonnet" {
		t.Errorf("Expected current model 'claude-3-5-sonnet', got '%s'", currentModel)
	}

	if err := multiModelManager.SwitchModel("openai"); err != nil {
		t.Fatalf("Failed to switch model: %v", err)
	}

	currentModel = multiModelManager.GetModel()
	if currentModel != "gpt-4" {
		t.Errorf("Expected current model 'gpt-4', got '%s'", currentModel)
	}

	t.Log("Multi-model manager integration test passed")
}

func TestMessageBusIntegration(t *testing.T) {
	ctx := context.Background()

	messageBus := bus.NewInMemoryMessageBus(ctx)
	messageBus.Start()

	messageReceived := make(chan bool, 1)

	handlerID, err := messageBus.Subscribe("test-topic", func(ctx context.Context, msg *bus.Message) error {
		messageReceived <- true
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	testMessage := &bus.Message{
		ID:        "test-id",
		Channel:   "test-topic",
		ChatID:    "test-chat",
		Content:   "test payload",
		Timestamp: time.Now(),
	}

	err = messageBus.Publish(ctx, "test-topic", testMessage)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	select {
	case <-messageReceived:
	case <-time.After(1 * time.Second):
		t.Error("Message not received within timeout")
	}

	err = messageBus.Unsubscribe("test-topic", handlerID)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	err = messageBus.Close()
	if err != nil {
		t.Fatalf("Failed to close message bus: %v", err)
	}

	t.Log("Message bus integration test passed")
}
