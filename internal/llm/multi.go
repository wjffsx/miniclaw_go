package llm

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type ModelConfig struct {
	Name        string           `yaml:"name"`
	Provider    string           `yaml:"provider"`
	APIKey      string           `yaml:"api_key,omitempty"`
	Model       string           `yaml:"model"`
	BaseURL     string           `yaml:"base_url,omitempty"`
	MaxTokens   int              `yaml:"max_tokens"`
	Temperature float64          `yaml:"temperature"`
	LocalModel  LocalModelConfig `yaml:"local_model,omitempty"`
}

type MultiModelManager struct {
	mu           sync.RWMutex
	providers    map[string]LLMProvider
	models       map[string]*ModelConfig
	currentModel string
	defaultModel string
}

func NewMultiModelManager(models []*ModelConfig, defaultModel string) (*MultiModelManager, error) {
	mmm := &MultiModelManager{
		providers:    make(map[string]LLMProvider),
		models:       make(map[string]*ModelConfig),
		currentModel: defaultModel,
		defaultModel: defaultModel,
	}

	for _, modelConfig := range models {
		if err := mmm.AddModel(modelConfig); err != nil {
			log.Printf("Warning: failed to add model %s: %v", modelConfig.Name, err)
		}
	}

	if _, ok := mmm.providers[defaultModel]; !ok {
		return nil, fmt.Errorf("default model %s not found", defaultModel)
	}

	return mmm, nil
}

func (mmm *MultiModelManager) AddModel(config *ModelConfig) error {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	if _, ok := mmm.providers[config.Name]; ok {
		return fmt.Errorf("model %s already exists", config.Name)
	}

	llmConfig := &Config{
		Provider:    config.Provider,
		APIKey:      config.APIKey,
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		LocalModel:  config.LocalModel,
	}

	var provider LLMProvider
	var err error

	switch config.Provider {
	case "anthropic":
		if config.APIKey == "" {
			return fmt.Errorf("API key is required for Anthropic provider")
		}
		provider = NewAnthropicProvider(llmConfig)
		log.Printf("Added Anthropic model: %s (%s)", config.Name, config.Model)

	case "openai":
		if config.APIKey == "" {
			return fmt.Errorf("API key is required for OpenAI provider")
		}
		provider = NewOpenAIProvider(llmConfig)
		log.Printf("Added OpenAI model: %s (%s)", config.Name, config.Model)

	case "local":
		if config.LocalModel.Path == "" {
			return fmt.Errorf("model path is required for local provider")
		}
		provider = NewLocalProvider(llmConfig)
		log.Printf("Added local model: %s (%s)", config.Name, config.LocalModel.Path)

	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	mmm.providers[config.Name] = provider
	mmm.models[config.Name] = config

	return nil
}

func (mmm *MultiModelManager) RemoveModel(name string) error {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	if name == mmm.defaultModel {
		return fmt.Errorf("cannot remove default model")
	}

	if _, ok := mmm.providers[name]; !ok {
		return fmt.Errorf("model %s not found", name)
	}

	delete(mmm.providers, name)
	delete(mmm.models, name)

	if mmm.currentModel == name {
		mmm.currentModel = mmm.defaultModel
		log.Printf("Switched to default model: %s", mmm.defaultModel)
	}

	return nil
}

func (mmm *MultiModelManager) SwitchModel(name string) error {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	if _, ok := mmm.providers[name]; !ok {
		return fmt.Errorf("model %s not found", name)
	}

	mmm.currentModel = name
	log.Printf("Switched to model: %s", name)

	return nil
}

func (mmm *MultiModelManager) GetCurrentModel() string {
	mmm.mu.RLock()
	defer mmm.mu.RUnlock()
	return mmm.currentModel
}

func (mmm *MultiModelManager) GetModelConfig(name string) (*ModelConfig, error) {
	mmm.mu.RLock()
	defer mmm.mu.RUnlock()

	config, ok := mmm.models[name]
	if !ok {
		return nil, fmt.Errorf("model %s not found", name)
	}

	return config, nil
}

func (mmm *MultiModelManager) ListModels() []string {
	mmm.mu.RLock()
	defer mmm.mu.RUnlock()

	models := make([]string, 0, len(mmm.models))
	for name := range mmm.models {
		models = append(models, name)
	}

	return models
}

func (mmm *MultiModelManager) Complete(ctx context.Context, messages []Message) (*CompletionResponse, error) {
	mmm.mu.RLock()
	provider, ok := mmm.providers[mmm.currentModel]
	mmm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("current model %s not found", mmm.currentModel)
	}

	config := mmm.models[mmm.currentModel]
	req := &CompletionRequest{
		Messages:    messages,
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	return provider.Complete(ctx, req)
}

func (mmm *MultiModelManager) GetProvider() string {
	mmm.mu.RLock()
	defer mmm.mu.RUnlock()

	config, ok := mmm.models[mmm.currentModel]
	if !ok {
		return "unknown"
	}

	return config.Provider
}

func (mmm *MultiModelManager) GetModel() string {
	mmm.mu.RLock()
	defer mmm.mu.RUnlock()

	config, ok := mmm.models[mmm.currentModel]
	if !ok {
		return "unknown"
	}

	return config.Model
}
