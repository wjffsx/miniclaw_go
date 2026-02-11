package skills

import (
	"context"
	"fmt"
	"sync"

	"github.com/wjffsx/miniclaw_go/internal/storage"
)

type SkillRegistry struct {
	mu      sync.RWMutex
	skills  map[string]*Skill
	index   *SkillIndex
	storage storage.Storage
	parser  *SkillParser
}

func NewSkillRegistry(storage storage.Storage) *SkillRegistry {
	return &SkillRegistry{
		skills:  make(map[string]*Skill),
		index:   NewSkillIndex(),
		storage: storage,
		parser:  NewSkillParser(storage),
	}
}

func (r *SkillRegistry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if skill.ID == "" {
		return fmt.Errorf("skill ID cannot be empty")
	}

	if skill.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	if _, exists := r.skills[skill.ID]; exists {
		r.index.Remove(skill.ID)
	}

	r.skills[skill.ID] = skill
	r.index.Add(skill)

	return nil
}

func (r *SkillRegistry) Unregister(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skillID]; !exists {
		return fmt.Errorf("skill %s not found", skillID)
	}

	delete(r.skills, skillID)
	r.index.Remove(skillID)

	return nil
}

func (r *SkillRegistry) Get(id string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[id]
	return skill, exists
}

func (r *SkillRegistry) GetByName(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, skill := range r.skills {
		if skill.Name == name {
			return skill, true
		}
	}

	return nil, false
}

func (r *SkillRegistry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		if skill.Enabled {
			skills = append(skills, skill)
		}
	}
	return skills
}

func (r *SkillRegistry) ListAll() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

func (r *SkillRegistry) Search(query string) []*Skill {
	return r.index.Search(query)
}

func (r *SkillRegistry) GetByTag(tag string) []*Skill {
	return r.index.GetByTag(tag)
}

func (r *SkillRegistry) GetByCategory(category string) []*Skill {
	return r.index.GetByCategory(category)
}

func (r *SkillRegistry) LoadFromDirectory(ctx context.Context, dir string) error {
	skills, err := r.parser.ParseDirectory(ctx, dir)
	if err != nil {
		return fmt.Errorf("failed to parse skills directory: %w", err)
	}

	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			return fmt.Errorf("failed to register skill %s: %w", skill.ID, err)
		}
	}

	return nil
}

func (r *SkillRegistry) LoadFromFile(ctx context.Context, path string) error {
	skill, err := r.parser.Parse(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to parse skill file: %w", err)
	}

	if err := r.Register(skill); err != nil {
		return fmt.Errorf("failed to register skill %s: %w", skill.ID, err)
	}

	return nil
}

func (r *SkillRegistry) Enable(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.skills[skillID]
	if !exists {
		return fmt.Errorf("skill %s not found", skillID)
	}

	skill.Enabled = true
	skill.Update()

	r.index.Remove(skillID)
	r.index.Add(skill)

	return nil
}

func (r *SkillRegistry) Disable(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.skills[skillID]
	if !exists {
		return fmt.Errorf("skill %s not found", skillID)
	}

	skill.Enabled = false
	skill.Update()

	r.index.Remove(skillID)

	return nil
}

func (r *SkillRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, skill := range r.skills {
		if skill.Enabled {
			count++
		}
	}
	return count
}

func (r *SkillRegistry) CountAll() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

func (r *SkillRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills = make(map[string]*Skill)
	r.index = NewSkillIndex()
}
