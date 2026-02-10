package llm

import (
	"testing"
)

func TestNewLocalProvider(t *testing.T) {
	config := &Config{
		Provider: "local",
		LocalModel: LocalModelConfig{
			Enabled: true,
			Path:    "/path/to/model.gguf",
			Type:    "llama",
		},
		MaxTokens:   2048,
		Temperature: 0.8,
	}

	provider := NewLocalProvider(config)

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	if provider.GetModel() != "/path/to/model.gguf" {
		t.Errorf("expected '/path/to/model.gguf', got %s", provider.GetModel())
	}
}

func TestLocalProviderGetModel(t *testing.T) {
	config := &Config{
		LocalModel: LocalModelConfig{
			Path: "/path/to/model.gguf",
		},
	}

	provider := NewLocalProvider(config)

	model := provider.GetModel()

	if model != "/path/to/model.gguf" {
		t.Errorf("expected '/path/to/model.gguf', got %s", model)
	}
}

func TestLocalProviderIsAvailable(t *testing.T) {
	provider := &LocalProvider{}

	available := provider.IsAvailable()

	if available {
		t.Log("llama-cli is available (this is expected if llama.cpp is installed)")
	} else {
		t.Log("llama-cli is not available (this is expected if llama.cpp is not installed)")
	}
}
