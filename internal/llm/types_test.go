package llm

import (
	"testing"
)

func TestMessageRole(t *testing.T) {
	tests := []struct {
		name  string
		role  MessageRole
		valid bool
	}{
		{"System role", RoleSystem, true},
		{"User role", RoleUser, true},
		{"Assistant role", RoleAssistant, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				if string(tt.role) == "" {
					t.Errorf("expected valid role, got empty string")
				}
			}
		})
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello, world!",
	}

	if msg.Role != RoleUser {
		t.Errorf("expected RoleUser, got %v", msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %s", msg.Content)
	}
}

func TestCompletionRequest(t *testing.T) {
	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hello!"},
		},
		Model:       "claude-sonnet-4-5",
		MaxTokens:   100,
		Temperature: 0.7,
		Stream:      false,
	}

	if len(req.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(req.Messages))
	}

	if req.Model != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", req.Model)
	}

	if req.MaxTokens != 100 {
		t.Errorf("expected 100, got %d", req.MaxTokens)
	}

	if req.Temperature != 0.7 {
		t.Errorf("expected 0.7, got %f", req.Temperature)
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := &CompletionResponse{
		Content: "Hello! How can I help you today?",
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("expected 'Hello! How can I help you today?', got %s", resp.Content)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected 10, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("expected 20, got %d", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected 30, got %d", resp.Usage.TotalTokens)
	}
}

func TestUsage(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
	}

	if usage.PromptTokens != 100 {
		t.Errorf("expected 100, got %d", usage.PromptTokens)
	}

	if usage.CompletionTokens != 200 {
		t.Errorf("expected 200, got %d", usage.CompletionTokens)
	}

	if usage.TotalTokens != 300 {
		t.Errorf("expected 300, got %d", usage.TotalTokens)
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		Provider:    "anthropic",
		APIKey:      "test-api-key",
		Model:       "claude-sonnet-4-5",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	if config.Provider != "anthropic" {
		t.Errorf("expected 'anthropic', got %s", config.Provider)
	}

	if config.APIKey != "test-api-key" {
		t.Errorf("expected 'test-api-key', got %s", config.APIKey)
	}

	if config.Model != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", config.Model)
	}

	if config.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", config.MaxTokens)
	}

	if config.Temperature != 0.7 {
		t.Errorf("expected 0.7, got %f", config.Temperature)
	}
}