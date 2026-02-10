package llm

import (
	"testing"
)

func TestNewMultiModelManager(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:        "model1",
			Provider:    "anthropic",
			APIKey:      "key1",
			Model:       "claude-sonnet-4-5",
			MaxTokens:   100,
			Temperature: 0.7,
		},
	}

	manager, err := NewMultiModelManager(models, "model1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.GetCurrentModel() != "model1" {
		t.Errorf("expected 'model1', got %s", manager.GetCurrentModel())
	}
}

func TestMultiModelManagerAddModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:        "default",
			Provider:    "anthropic",
			APIKey:      "key1",
			Model:       "claude-sonnet-4-5",
			MaxTokens:   100,
			Temperature: 0.7,
		},
	}

	manager, _ := NewMultiModelManager(models, "default")

	config := &ModelConfig{
		Name:        "model1",
		Provider:    "anthropic",
		APIKey:      "key1",
		Model:       "claude-sonnet-4-5",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	err := manager.AddModel(config)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	modelsList := manager.ListModels()

	if len(modelsList) != 2 {
		t.Errorf("expected 2 models, got %d", len(modelsList))
	}
}

func TestMultiModelManagerAddDuplicateModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "default",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "default")

	config := &ModelConfig{
		Name:     "model1",
		Provider: "anthropic",
		APIKey:   "key1",
		Model:    "claude-sonnet-4-5",
	}

	manager.AddModel(config)

	err := manager.AddModel(config)

	if err == nil {
		t.Error("expected error for duplicate model")
	}
}

func TestMultiModelManagerRemoveModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
		{
			Name:     "model2",
			Provider: "openai",
			APIKey:   "key2",
			Model:    "gpt-4o",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	err := manager.RemoveModel("model2")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	modelsList := manager.ListModels()

	if len(modelsList) != 1 {
		t.Errorf("expected 1 model, got %d", len(modelsList))
	}
}

func TestMultiModelManagerRemoveDefaultModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	err := manager.RemoveModel("model1")

	if err == nil {
		t.Error("expected error when removing default model")
	}
}

func TestMultiModelManagerSwitchModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
		{
			Name:     "model2",
			Provider: "openai",
			APIKey:   "key2",
			Model:    "gpt-4o",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	err := manager.SwitchModel("model2")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if manager.GetCurrentModel() != "model2" {
		t.Errorf("expected 'model2', got %s", manager.GetCurrentModel())
	}
}

func TestMultiModelManagerSwitchToNonExistentModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	err := manager.SwitchModel("nonexistent")

	if err == nil {
		t.Error("expected error when switching to non-existent model")
	}
}

func TestMultiModelManagerListModels(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
		{
			Name:     "model2",
			Provider: "openai",
			APIKey:   "key2",
			Model:    "gpt-4o",
		},
		{
			Name:     "model3",
			Provider: "local",
			LocalModel: LocalModelConfig{
				Path: "/path/to/model.gguf",
				Type: "llama",
			},
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	modelsList := manager.ListModels()

	if len(modelsList) != 3 {
		t.Errorf("expected 3 models, got %d", len(modelsList))
	}
}

func TestMultiModelManagerGetModelConfig(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:        "model1",
			Provider:    "anthropic",
			APIKey:      "key1",
			Model:       "claude-sonnet-4-5",
			MaxTokens:   100,
			Temperature: 0.7,
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	config, err := manager.GetModelConfig("model1")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if config.Name != "model1" {
		t.Errorf("expected 'model1', got %s", config.Name)
	}

	if config.Provider != "anthropic" {
		t.Errorf("expected 'anthropic', got %s", config.Provider)
	}

	if config.Model != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", config.Model)
	}
}

func TestMultiModelManagerGetModelConfigNotFound(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	_, err := manager.GetModelConfig("nonexistent")

	if err == nil {
		t.Error("expected error when getting non-existent model config")
	}
}

func TestMultiModelManagerGetProvider(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	provider := manager.GetProvider()

	if provider != "anthropic" {
		t.Errorf("expected 'anthropic', got %s", provider)
	}
}

func TestMultiModelManagerGetModel(t *testing.T) {
	models := []*ModelConfig{
		{
			Name:     "model1",
			Provider: "anthropic",
			APIKey:   "key1",
			Model:    "claude-sonnet-4-5",
		},
	}

	manager, _ := NewMultiModelManager(models, "model1")

	model := manager.GetModel()

	if model != "claude-sonnet-4-5" {
		t.Errorf("expected 'claude-sonnet-4-5', got %s", model)
	}
}
