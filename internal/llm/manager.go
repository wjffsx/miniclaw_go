package llm

import (
	"context"
	"fmt"
	"log"
)

type Manager struct {
	provider LLMProvider
	config   *Config
}

func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	var provider LLMProvider

	switch config.Provider {
	case "anthropic":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Anthropic provider")
		}
		if config.Model == "" {
			config.Model = "claude-sonnet-4-5"
		}
		provider = NewAnthropicProvider(config)
		log.Printf("Initialized Anthropic provider with model: %s", config.Model)

	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for OpenAI provider")
		}
		if config.Model == "" {
			config.Model = "gpt-4o"
		}
		provider = NewOpenAIProvider(config)
		log.Printf("Initialized OpenAI provider with model: %s", config.Model)

	case "local":
		if config.LocalModel.Path == "" {
			return nil, fmt.Errorf("model path is required for local provider")
		}
		if config.LocalModel.Type == "" {
			config.LocalModel.Type = "llama"
		}
		provider = NewLocalProvider(config)
		log.Printf("Initialized local provider with model: %s (%s)", config.LocalModel.Path, config.LocalModel.Type)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}

	return &Manager{
		provider: provider,
		config:   config,
	}, nil
}

func (m *Manager) Complete(ctx context.Context, messages []Message) (*CompletionResponse, error) {
	req := &CompletionRequest{
		Messages:    messages,
		Model:       m.config.Model,
		MaxTokens:   m.config.MaxTokens,
		Temperature: m.config.Temperature,
	}

	return m.provider.Complete(ctx, req)
}

func (m *Manager) GetModel() string {
	return m.provider.GetModel()
}

func (m *Manager) GetProvider() string {
	return m.config.Provider
}
