package skills

import (
	"strings"
	"sync"
)

type SkillIndex struct {
	mu         sync.RWMutex
	byName     map[string]*Skill
	byTag      map[string][]*Skill
	byCategory map[string][]*Skill
	byKeyword  map[string][]*Skill
}

func NewSkillIndex() *SkillIndex {
	return &SkillIndex{
		byName:     make(map[string]*Skill),
		byTag:      make(map[string][]*Skill),
		byCategory: make(map[string][]*Skill),
		byKeyword:  make(map[string][]*Skill),
	}
}

func (idx *SkillIndex) Add(skill *Skill) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.byName[skill.ID] = skill

	for _, tag := range skill.Tags {
		idx.byTag[tag] = append(idx.byTag[tag], skill)
	}

	if skill.Category != "" {
		idx.byCategory[skill.Category] = append(idx.byCategory[skill.Category], skill)
	}

	keywords := extractKeywords(skill.Name + " " + skill.Description)
	for _, keyword := range keywords {
		idx.byKeyword[keyword] = append(idx.byKeyword[keyword], skill)
	}
}

func (idx *SkillIndex) Remove(skillID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	skill, exists := idx.byName[skillID]
	if !exists {
		return
	}

	delete(idx.byName, skillID)

	for _, tag := range skill.Tags {
		skills := idx.byTag[tag]
		for i, s := range skills {
			if s.ID == skillID {
				idx.byTag[tag] = append(skills[:i], skills[i+1:]...)
				if len(idx.byTag[tag]) == 0 {
					delete(idx.byTag, tag)
				}
				break
			}
		}
	}

	if skill.Category != "" {
		skills := idx.byCategory[skill.Category]
		for i, s := range skills {
			if s.ID == skillID {
				idx.byCategory[skill.Category] = append(skills[:i], skills[i+1:]...)
				if len(idx.byCategory[skill.Category]) == 0 {
					delete(idx.byCategory, skill.Category)
				}
				break
			}
		}
	}

	keywords := extractKeywords(skill.Name + " " + skill.Description)
	for _, keyword := range keywords {
		skills := idx.byKeyword[keyword]
		for i, s := range skills {
			if s.ID == skillID {
				idx.byKeyword[keyword] = append(skills[:i], skills[i+1:]...)
				if len(idx.byKeyword[keyword]) == 0 {
					delete(idx.byKeyword, keyword)
				}
				break
			}
		}
	}
}

func (idx *SkillIndex) Search(query string) []*Skill {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	query = strings.ToLower(query)
	keywords := extractKeywords(query)

	scoreMap := make(map[string]float64)

	for _, keyword := range keywords {
		if skills, ok := idx.byKeyword[keyword]; ok {
			for _, skill := range skills {
				if !skill.Enabled {
					continue
				}
				scoreMap[skill.ID] += 1.0
			}
		}

		if skills, ok := idx.byTag[query]; ok {
			for _, skill := range skills {
				if !skill.Enabled {
					continue
				}
				scoreMap[skill.ID] += 2.0
			}
		}

		if skills, ok := idx.byCategory[query]; ok {
			for _, skill := range skills {
				if !skill.Enabled {
					continue
				}
				scoreMap[skill.ID] += 1.5
			}
		}
	}

	results := make([]*Skill, 0, len(scoreMap))
	for id, score := range scoreMap {
		if skill, ok := idx.byName[id]; ok && score > 0 {
			results = append(results, skill)
		}
	}

	return results
}

func (idx *SkillIndex) GetByTag(tag string) []*Skill {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	skills, ok := idx.byTag[tag]
	if !ok {
		return []*Skill{}
	}

	enabled := make([]*Skill, 0, len(skills))
	for _, skill := range skills {
		if skill.Enabled {
			enabled = append(enabled, skill)
		}
	}

	return enabled
}

func (idx *SkillIndex) GetByCategory(category string) []*Skill {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	skills, ok := idx.byCategory[category]
	if !ok {
		return []*Skill{}
	}

	enabled := make([]*Skill, 0, len(skills))
	for _, skill := range skills {
		if skill.Enabled {
			enabled = append(enabled, skill)
		}
	}

	return enabled
}

func (idx *SkillIndex) GetAll() []*Skill {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	skills := make([]*Skill, 0, len(idx.byName))
	for _, skill := range idx.byName {
		if skill.Enabled {
			skills = append(skills, skill)
		}
	}

	return skills
}

func extractKeywords(text string) []string {
	text = strings.ToLower(text)

	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"should": true, "could": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
	}

	words := strings.Fields(text)
	keywords := make([]string, 0, len(words))
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:'\"()[]{}")

		if len(word) < 2 {
			continue
		}

		if stopWords[word] {
			continue
		}

		if seen[word] {
			continue
		}

		seen[word] = true
		keywords = append(keywords, word)
	}

	return keywords
}
