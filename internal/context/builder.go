package context

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type Builder struct {
	storage       storage.Storage
	memoryStorage storage.MemoryStorage
}

type Config struct {
	Storage       storage.Storage
	MemoryStorage storage.MemoryStorage
}

func NewBuilder(config *Config) *Builder {
	return &Builder{
		storage:       config.Storage,
		memoryStorage: config.MemoryStorage,
	}
}

type Context struct {
	SystemPrompt string
	Memory      string
	DailyNotes  []string
	Tools       []tools.ToolSchema
}

func (b *Builder) Build(ctx context.Context, toolSchemas []tools.ToolSchema) (*Context, error) {
	result := &Context{
		Tools: toolSchemas,
	}

	if err := b.loadSystemPrompt(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	if err := b.loadMemory(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load memory: %w", err)
	}

	if err := b.loadDailyNotes(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to load daily notes: %w", err)
	}

	return result, nil
}

func (b *Builder) loadSystemPrompt(ctx context.Context, result *Context) error {
	soulContent, err := b.storage.ReadFile(ctx, "config/SOUL.md")
	if err != nil {
		return fmt.Errorf("failed to read SOUL.md: %w", err)
	}

	userContent, err := b.storage.ReadFile(ctx, "config/USER.md")
	if err != nil {
		return fmt.Errorf("failed to read USER.md: %w", err)
	}

	agentsContent, err := b.storage.ReadFile(ctx, "config/AGENTS.md")
	if err == nil && len(agentsContent) > 0 {
		result.SystemPrompt = fmt.Sprintf("%s\n\n%s\n\n%s", string(soulContent), string(userContent), string(agentsContent))
	} else {
		result.SystemPrompt = fmt.Sprintf("%s\n\n%s", string(soulContent), string(userContent))
	}

	return nil
}

func (b *Builder) loadMemory(ctx context.Context, result *Context) error {
	memory, err := b.memoryStorage.GetMemory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}

	result.Memory = memory
	return nil
}

func (b *Builder) loadDailyNotes(ctx context.Context, result *Context) error {
	notes := make([]string, 0, 7)

	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		note, err := b.memoryStorage.GetDailyNote(ctx, date)
		if err != nil {
			continue
		}

		if note != "" {
			notes = append(notes, fmt.Sprintf("## %s\n%s", date, note))
		}
	}

	result.DailyNotes = notes
	return nil
}

func (c *Context) BuildSystemPrompt(toolSchemas []tools.ToolSchema) string {
	var prompt strings.Builder

	prompt.WriteString(c.SystemPrompt)
	prompt.WriteString("\n\n")

	if c.Memory != "" {
		prompt.WriteString("## Memory\n")
		prompt.WriteString(c.Memory)
		prompt.WriteString("\n\n")
	}

	if len(c.DailyNotes) > 0 {
		prompt.WriteString("## Recent Notes\n")
		for _, note := range c.DailyNotes {
			prompt.WriteString(note)
			prompt.WriteString("\n\n")
		}
	}

	if len(toolSchemas) > 0 {
		prompt.WriteString("## Available Tools\n")
		prompt.WriteString("You have access to the following tools:\n\n")

		for _, tool := range toolSchemas {
			prompt.WriteString(fmt.Sprintf("- **%s**: %s\n", tool.Name, tool.Description))
		}

		prompt.WriteString("\n")
		prompt.WriteString(`When you need to use a tool, respond in the following JSON format:
{
  "thought": "Your reasoning about what to do",
  "tool_calls": [
    {
      "name": "tool_name",
      "input": {
        "param1": "value1",
        "param2": "value2"
      }
    }
  ]
}

When you have a final answer and don't need to use any more tools, respond in the following JSON format:
{
  "thought": "Your reasoning",
  "final_answer": "Your final answer to the user"
}
`)
	}

	return prompt.String()
}

func (c *Context) GetTokenEstimate() int {
	totalTokens := len(c.SystemPrompt)
	totalTokens += len(c.Memory)

	for _, note := range c.DailyNotes {
		totalTokens += len(note)
	}

	for _, tool := range c.Tools {
		totalTokens += len(tool.Name) + len(tool.Description) + len(tool.Parameters)
	}

	return totalTokens / 4
}