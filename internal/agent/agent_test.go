package agent

import (
	"context"
	"testing"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/bus"
	"github.com/wjffsx/miniclaw_go/internal/llm"
	"github.com/wjffsx/miniclaw_go/internal/mcp"
	"github.com/wjffsx/miniclaw_go/internal/scheduler"
	"github.com/wjffsx/miniclaw_go/internal/skills"
	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestNewAgent(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if agent == nil {
		t.Error("Expected agent to be created")
	}

	if agent.messageBus != messageBus {
		t.Error("Expected agent messageBus to be set")
	}

	if agent.maxIterations != 10 {
		t.Errorf("Expected maxIterations 10, got %d", agent.maxIterations)
	}
}

func TestNewAgentNilConfig(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	_, err := NewAgent(nil, messageBus, ctx)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestAgentProcessMessage(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	message := &bus.Message{
		ChatID:  "test-chat",
		Content: "test message",
	}

	err = agent.HandleMessage(ctx, message)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestAgentProcessMessageNil(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.HandleMessage(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil message")
	}
}

func TestAgentGetChatHistory(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	history := agent.GetChatHistory("test-chat")
	if history == nil {
		t.Error("Expected chat history to be initialized")
	}
}

func TestAgentClearChatHistory(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	agent.ClearChatHistory("test-chat")

	history := agent.GetChatHistory("test-chat")
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d messages", len(history))
	}
}

func TestAgentSetMaxIterations(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	agent.SetMaxIterations(20)

	if agent.maxIterations != 20 {
		t.Errorf("Expected maxIterations 20, got %d", agent.maxIterations)
	}
}

func TestAgentGetMaxIterations(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	maxIterations := agent.GetMaxIterations()
	if maxIterations != 10 {
		t.Errorf("Expected maxIterations 10, got %d", maxIterations)
	}
}

func TestAgentGetToolExecutor(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	toolExecutor := agent.GetToolExecutor()
	if toolExecutor == nil {
		t.Error("Expected toolExecutor to be set")
	}
}

func TestAgentGetSkillSelector(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	skillSelector := agent.GetSkillSelector()
	if skillSelector == nil {
		t.Error("Expected skillSelector to be set")
	}
}

func TestAgentGetMCPManager(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	mcpManager := agent.GetMCPManager()
	if mcpManager == nil {
		t.Error("Expected mcpManager to be set")
	}
}

func TestAgentGetTaskManager(t *testing.T) {
	messageBus := bus.NewInMemoryMessageBus(context.Background())
	ctx := context.Background()

	config := &Config{
		LLMModels:      []*llm.ModelConfig{},
		DefaultModel:   "default",
		SessionStorage: storage.NewFileSystemSessionStorage(""),
		MemoryStorage:  storage.NewFileSystemMemoryStorage(""),
		Storage:        storage.NewFileStorage(""),
		ToolRegistry:   tools.NewToolRegistry(),
		SkillRegistry:  skills.NewSkillRegistry(nil),
		SkillConfig:    &skills.SkillConfig{},
		MCPManager:     mcp.NewMCPManager(nil),
		TaskManager:    scheduler.NewTaskManager(scheduler.NewScheduler(&scheduler.SchedulerConfig{TickInterval: 1 * time.Second}), nil),
		MaxIterations:  10,
	}

	agent, err := NewAgent(config, messageBus, ctx)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	taskManager := agent.GetTaskManager()
	if taskManager == nil {
		t.Error("Expected taskManager to be set")
	}
}
