package skills

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/storage"
	"gopkg.in/yaml.v3"
)

type SkillParser struct {
	storage storage.Storage
}

func NewSkillParser(storage storage.Storage) *SkillParser {
	return &SkillParser{
		storage: storage,
	}
}

func (p *SkillParser) Parse(ctx context.Context, path string) (*Skill, error) {
	var content []byte
	var err error

	if filepath.IsAbs(path) {
		content, err = os.ReadFile(path)
	} else {
		content, err = p.storage.ReadFile(ctx, path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	return p.ParseContent(string(content), path)
}

func (p *SkillParser) ParseContent(content, path string) (*Skill, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid skill format: expected front matter between --- markers")
	}

	frontMatter := parts[1]
	skillContent := strings.TrimSpace(parts[2])

	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontMatter), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse front matter: %w", err)
	}

	skill := &Skill{
		ID:          generateSkillID(path),
		Name:        getString(metadata, "name"),
		Description: getString(metadata, "description"),
		Category:    getString(metadata, "category"),
		Tags:        getStringSlice(metadata, "tags"),
		Requires:    getStringSlice(metadata, "requires"),
		Content:     skillContent,
		Metadata:    extractMetadata(metadata),
		Enabled:     getBool(metadata, "enabled", true),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if skill.Name == "" {
		return nil, fmt.Errorf("skill name is required")
	}

	if skill.Description == "" {
		return nil, fmt.Errorf("skill description is required")
	}

	return skill, nil
}

func (p *SkillParser) ParseDirectory(ctx context.Context, dir string) ([]*Skill, error) {
	var files []string
	var err error

	if filepath.IsAbs(dir) {
		files, err = p.listAbsoluteDirectory(ctx, dir)
	} else {
		files, err = p.storage.ListFiles(ctx, dir)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list skill directory: %w", err)
	}

	skills := make([]*Skill, 0, len(files))

	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file), ".md") {
			continue
		}

		skill, err := p.Parse(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse skill file %s: %w", file, err)
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

func (p *SkillParser) listAbsoluteDirectory(ctx context.Context, dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return make([]string, 0)
}

func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func extractMetadata(m map[string]interface{}) map[string]string {
	result := make(map[string]string)

	excludeKeys := map[string]bool{
		"name":        true,
		"description": true,
		"category":    true,
		"tags":        true,
		"requires":    true,
		"enabled":     true,
	}

	for key, val := range m {
		if excludeKeys[key] {
			continue
		}

		switch v := val.(type) {
		case string:
			result[key] = v
		case bool:
			result[key] = fmt.Sprintf("%v", v)
		case float64:
			result[key] = fmt.Sprintf("%v", v)
		case int:
			result[key] = fmt.Sprintf("%d", v)
		}
	}

	return result
}

func generateSkillID(path string) string {
	if path != "" {
		hash := sha256.Sum256([]byte(path))
		hashStr := hex.EncodeToString(hash[:])[:8]
		filename := filepath.Base(path)
		return fmt.Sprintf("%s-%s", strings.TrimSuffix(filename, filepath.Ext(filename)), hashStr)
	}

	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
