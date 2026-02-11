package skills

import (
	"time"
)

type Skill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Requires    []string          `json:"requires"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type SkillTrigger struct {
	Keywords   []string `json:"keywords"`
	Intent     string   `json:"intent"`
	Confidence float64  `json:"confidence"`
}

func NewSkill(name, description, category string) *Skill {
	return &Skill{
		ID:          generateSkillID(""),
		Name:        name,
		Description: description,
		Category:    category,
		Tags:        make([]string, 0),
		Requires:    make([]string, 0),
		Metadata:    make(map[string]string),
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (s *Skill) Update() {
	s.UpdatedAt = time.Now()
}

func (s *Skill) AddTag(tag string) {
	for _, t := range s.Tags {
		if t == tag {
			return
		}
	}
	s.Tags = append(s.Tags, tag)
	s.Update()
}

func (s *Skill) RemoveTag(tag string) {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			s.Update()
			return
		}
	}
}

func (s *Skill) AddRequire(tool string) {
	for _, r := range s.Requires {
		if r == tool {
			return
		}
	}
	s.Requires = append(s.Requires, tool)
	s.Update()
}

func (s *Skill) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
	s.Update()
}

func (s *Skill) GetMetadata(key string) (string, bool) {
	if s.Metadata == nil {
		return "", false
	}
	value, exists := s.Metadata[key]
	return value, exists
}

func (s *Skill) Validate() error {
	if s.ID == "" {
		return &SkillError{
			Code:    "EMPTY_ID",
			Message: "skill ID cannot be empty",
		}
	}
	if s.Name == "" {
		return &SkillError{
			Code:    "EMPTY_NAME",
			Message: "skill name cannot be empty",
		}
	}
	return nil
}

type SkillError struct {
	Code    string
	Message string
}

func (e *SkillError) Error() string {
	return e.Message
}

func (e *SkillError) Unwrap() error {
	return nil
}

