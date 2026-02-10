package llm

import (
	"testing"
)

func TestNewOpenAIProvider(t *testing.T) {
	config := &Config{
		Provider:    "openai",
		APIKey:      "test-api-key",
		Model:       "gpt-4o",
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	provider := NewOpenAIProvider(config)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	if provider.GetModel() != "gpt-4o" {
		t.Errorf("expected 'gpt-4o', got %s", provider.GetModel())
	}
}

func TestOpenAIProviderGetModel(t *testing.T) {
	config := &Config{
		Model: "gpt-4o",
	}

	provider := NewOpenAIProvider(config)

	model := provider.GetModel()

	if model != "gpt-4o" {
		t.Errorf("expected 'gpt-4o', got %s", model)
	}
}
