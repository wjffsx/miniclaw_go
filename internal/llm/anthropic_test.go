package llm

import (
	"testing"
)

func TestNewAnthropicProvider(t *testing.T) {
	config := &Config{
		Provider:    "anthropic",
		APIKey:      "test-api-key",
		Model:       "claude-sonnet-4-5",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	provider := NewAnthropicProvider(config)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	if provider.GetModel() != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", provider.GetModel())
	}
}

func TestAnthropicProviderGetModel(t *testing.T) {
	config := &Config{
		Model: "claude-sonnet-4-5",
	}

	provider := NewAnthropicProvider(config)

	model := provider.GetModel()

	if model != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", model)
	}
}
