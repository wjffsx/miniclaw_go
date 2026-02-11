package skills

import (
	"context"
	"testing"

	"github.com/wjffsx/miniclaw_go/internal/llm"
)

func TestNewSkillSelector(t *testing.T) {
	registry := NewSkillRegistry(nil)
	config := &SelectionConfig{
		Method:    "hybrid",
		Threshold: 0.5,
		MaxActive: 5,
	}

	selector := NewSkillSelector(registry, nil, config)

	if selector == nil {
		t.Error("Expected selector to be created")
	}

	if selector.registry != registry {
		t.Error("Expected selector registry to be set")
	}

	if selector.config.Method != "hybrid" {
		t.Errorf("Expected method 'hybrid', got '%s'", selector.config.Method)
	}
}

func TestNewSkillSelectorNilConfig(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, nil)

	if selector == nil {
		t.Error("Expected selector to be created with default config")
	}

	if selector.config.Method != "hybrid" {
		t.Errorf("Expected default method 'hybrid', got '%s'", selector.config.Method)
	}

	if selector.config.Threshold != 0.5 {
		t.Errorf("Expected default threshold 0.5, got %f", selector.config.Threshold)
	}

	if selector.config.MaxActive != 5 {
		t.Errorf("Expected default MaxActive 5, got %d", selector.config.MaxActive)
	}
}

func TestSelectByKeyword(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.5,
	})

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) == 0 {
		t.Error("Expected at least one selection")
	}

	if selections[0].ID != skill.ID {
		t.Errorf("Expected skill ID '%s', got '%s'", skill.ID, selections[0].ID)
	}
}

func TestSelectByTag(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.3,
	})

	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	registry.Register(skill)

	selections, err := selector.Select(nil, "I need tag1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) == 0 {
		t.Error("Expected at least one selection")
	}
}

func TestSelectByCategory(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.3,
	})

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) == 0 {
		t.Error("Expected at least one selection")
	}
}

func TestSelectWithLLM(t *testing.T) {
	registry := NewSkillRegistry(nil)

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	mockLLM := &mockLLMProvider{
		responses: []string{
			`{"selected_skills": [{"skill_id": "` + skill.ID + `", "reasoning": "Good match"}]}`,
		},
	}

	selector := NewSkillSelector(registry, mockLLM, &SelectionConfig{
		Method:    "llm",
		Threshold: 0.5,
	})

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) == 0 {
		t.Error("Expected at least one selection")
	}
}

func TestSelectHybrid(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "hybrid",
		Threshold: 0.5,
	})

	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	registry.Register(skill)

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) == 0 {
		t.Error("Expected at least one selection")
	}
}

func TestSelectNoResults(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.5,
	})

	selections, err := selector.Select(nil, "nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) != 0 {
		t.Errorf("Expected 0 selections, got %d", len(selections))
	}
}

func TestSelectThreshold(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.9,
	})

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	selections, err := selector.Select(nil, "nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) != 0 {
		t.Errorf("Expected 0 selections below threshold, got %d", len(selections))
	}
}

func TestSelectMaxActive(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.0,
		MaxActive: 2,
	})

	for i := 0; i < 5; i++ {
		skill := NewSkill("test"+string(rune('0'+i)), "test description", "test-category")
		registry.Register(skill)
	}

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) > 2 {
		t.Errorf("Expected max 2 selections, got %d", len(selections))
	}
}

func TestSelectDisabledSkills(t *testing.T) {
	registry := NewSkillRegistry(nil)
	selector := NewSkillSelector(registry, nil, &SelectionConfig{
		Method:    "keyword",
		Threshold: 0.5,
	})

	skill := NewSkill("test", "test description", "test-category")
	skill.Enabled = false
	registry.Register(skill)

	selections, err := selector.Select(nil, "test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(selections) != 0 {
		t.Errorf("Expected 0 selections for disabled skill, got %d", len(selections))
	}
}

type mockLLMProvider struct {
	responses []string
	current   int
}

func (m *mockLLMProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.current >= len(m.responses) {
		return &llm.CompletionResponse{}, nil
	}
	response := m.responses[m.current]
	m.current++
	return &llm.CompletionResponse{Content: response}, nil
}

func (m *mockLLMProvider) StreamComplete(ctx context.Context, req *llm.CompletionRequest, callback func(chunk string) error) error {
	return nil
}

func (m *mockLLMProvider) GetModel() string {
	return "mock"
}
