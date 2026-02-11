package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/wjffsx/miniclaw_go/internal/llm"
)

type SkillSelector struct {
	registry *SkillRegistry
	llm      llm.LLMProvider
	config   *SelectionConfig
	mu       sync.RWMutex
}

type SkillConfig struct {
	Directory  string          `yaml:"directory"`
	AutoReload bool            `yaml:"auto_reload"`
	MaxActive  int             `yaml:"max_active"`
	Selection  SelectionConfig `yaml:"selection"`
}

type SelectionConfig struct {
	Method    string  `yaml:"method"`
	Threshold float64 `yaml:"threshold"`
	MaxActive int     `yaml:"max_active"`
}

type SkillSelection struct {
	Skill     *Skill
	Score     float64
	Reasoning string
}

func NewSkillSelector(registry *SkillRegistry, llm llm.LLMProvider, config *SelectionConfig) *SkillSelector {
	if config == nil {
		config = &SelectionConfig{
			Method:    "hybrid",
			Threshold: 0.5,
			MaxActive: 5,
		}
	}

	return &SkillSelector{
		registry: registry,
		llm:      llm,
		config:   config,
	}
}

func (s *SkillSelector) Select(ctx context.Context, userMessage string) ([]*Skill, error) {
	switch s.config.Method {
	case "keyword":
		return s.selectByKeyword(userMessage)
	case "llm":
		return s.selectByLLM(ctx, userMessage)
	case "hybrid":
		return s.selectHybrid(ctx, userMessage)
	default:
		return s.selectHybrid(ctx, userMessage)
	}
}

func (s *SkillSelector) selectByKeyword(userMessage string) ([]*Skill, error) {
	keywords := extractKeywords(userMessage)

	candidates := make([]*SkillSelection, 0)

	for _, skill := range s.registry.List() {
		score := s.calculateKeywordScore(skill, keywords, userMessage)
		if score >= s.config.Threshold {
			candidates = append(candidates, &SkillSelection{
				Skill:     skill,
				Score:     score,
				Reasoning: fmt.Sprintf("Keyword match score: %.2f", score),
			})
		}
	}

	return s.rankAndFilter(candidates), nil
}

func (s *SkillSelector) selectByLLM(ctx context.Context, userMessage string) ([]*Skill, error) {
	if s.llm == nil {
		return s.selectByKeyword(userMessage)
	}

	skills := s.registry.List()
	if len(skills) == 0 {
		return []*Skill{}, nil
	}

	skillList := s.buildSkillList(skills)

	prompt := fmt.Sprintf(`You are a skill selector. Given the user's message, select the most relevant skills from the list below.

Available Skills:
%s

User Message: %s

Respond with a JSON object in the following format:
{
  "selected_skills": [
    {
      "skill_id": "skill_id_here",
      "reasoning": "brief explanation of why this skill is relevant"
    }
  ]
}

Select at most %d skills. Only select skills that are directly relevant to the user's request.`, skillList, userMessage, s.config.MaxActive)

	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are a helpful assistant that selects relevant skills based on user messages.",
		},
		{
			Role:    llm.RoleUser,
			Content: prompt,
		},
	}

	req := &llm.CompletionRequest{
		Messages:  messages,
		MaxTokens: 1000,
	}

	resp, err := s.llm.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM selection failed: %w", err)
	}

	return s.parseLLMResponse(resp.Content)
}

func (s *SkillSelector) selectHybrid(ctx context.Context, userMessage string) ([]*Skill, error) {
	keywordResults, err := s.selectByKeyword(userMessage)
	if err != nil {
		return nil, err
	}

	if len(keywordResults) > 0 && len(keywordResults) <= s.config.MaxActive {
		return keywordResults, nil
	}

	if s.llm != nil {
		llmResults, err := s.selectByLLM(ctx, userMessage)
		if err == nil && len(llmResults) > 0 {
			return llmResults, nil
		}
	}

	return keywordResults, nil
}

func (s *SkillSelector) calculateKeywordScore(skill *Skill, keywords []string, message string) float64 {
	var score float64
	lowerMessage := strings.ToLower(message)

	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(skill.Name), keyword) {
			score += 0.3
		}
		if strings.Contains(strings.ToLower(skill.Description), keyword) {
			score += 0.2
		}
		for _, tag := range skill.Tags {
			if strings.Contains(strings.ToLower(tag), keyword) {
				score += 0.15
			}
		}
		if strings.Contains(strings.ToLower(skill.Category), keyword) {
			score += 0.1
		}
		if strings.Contains(strings.ToLower(skill.Content), keyword) {
			score += 0.05
		}
	}

	for _, tag := range skill.Tags {
		if strings.Contains(lowerMessage, strings.ToLower(tag)) {
			score += 0.25
		}
	}

	return math.Min(score, 1.0)
}

func (s *SkillSelector) buildSkillList(skills []*Skill) string {
	var builder strings.Builder

	for i, skill := range skills {
		builder.WriteString(fmt.Sprintf("%d. ID: %s, Name: %s, Description: %s, Tags: %v\n",
			i+1, skill.ID, skill.Name, skill.Description, skill.Tags))
	}

	return builder.String()
}

func (s *SkillSelector) parseLLMResponse(content string) ([]*Skill, error) {
	type LLMResponse struct {
		SelectedSkills []struct {
			SkillID   string `json:"skill_id"`
			Reasoning string `json:"reasoning"`
		} `json:"selected_skills"`
	}

	var resp LLMResponse
	if err := parseJSON(content, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	skills := make([]*Skill, 0, len(resp.SelectedSkills))

	for _, selection := range resp.SelectedSkills {
		if skill, exists := s.registry.Get(selection.SkillID); exists {
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

func (s *SkillSelector) rankAndFilter(candidates []*SkillSelection) []*Skill {
	if len(candidates) == 0 {
		return []*Skill{}
	}

	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].Score < candidates[j].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	maxSkills := s.config.MaxActive
	if maxSkills <= 0 {
		maxSkills = 5
	}

	if len(candidates) > maxSkills {
		candidates = candidates[:maxSkills]
	}

	skills := make([]*Skill, 0, len(candidates))
	for _, selection := range candidates {
		skills = append(skills, selection.Skill)
	}

	return skills
}

func (s *SkillSelector) SetConfig(config *SelectionConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *SkillSelector) GetConfig() *SelectionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func parseJSON(content string, v interface{}) error {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 {
		return fmt.Errorf("no JSON object found")
	}

	jsonStr := content[start : end+1]

	return json.Unmarshal([]byte(jsonStr), v)
}
