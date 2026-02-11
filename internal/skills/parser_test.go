package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wjffsx/miniclaw_go/internal/storage"
)

func TestNewSkillParser(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	parser := NewSkillParser(store)

	if parser == nil {
		t.Error("Expected parser to be created")
	}

	if parser.storage != store {
		t.Error("Expected parser storage to be set")
	}
}

func TestParseContent(t *testing.T) {
	parser := NewSkillParser(nil)

	content := `---
name: "test_skill"
description: "A test skill"
category: "test"
tags: ["tag1", "tag2"]
requires: ["read_file"]
enabled: true
---

# Test Skill

This is a test skill content.
`

	skill, err := parser.ParseContent(content, "test.md")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if skill.Name != "test_skill" {
		t.Errorf("Expected name 'test_skill', got '%s'", skill.Name)
	}

	if skill.Description != "A test skill" {
		t.Errorf("Expected description 'A test skill', got '%s'", skill.Description)
	}

	if skill.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", skill.Category)
	}

	if len(skill.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(skill.Tags))
	}

	if len(skill.Requires) != 1 {
		t.Errorf("Expected 1 requirement, got %d", len(skill.Requires))
	}

	if !skill.Enabled {
		t.Error("Expected skill to be enabled")
	}

	if skill.Content == "" {
		t.Error("Expected skill content to be set")
	}
}

func TestParseContentInvalidFormat(t *testing.T) {
	parser := NewSkillParser(nil)

	content := `invalid content without front matter`

	_, err := parser.ParseContent(content, "test.md")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestParseContentMissingName(t *testing.T) {
	parser := NewSkillParser(nil)

	content := `---
description: "A test skill"
category: "test"
---

# Test Skill
`

	_, err := parser.ParseContent(content, "test.md")
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestParseContentMissingDescription(t *testing.T) {
	parser := NewSkillParser(nil)

	content := `---
name: "test_skill"
category: "test"
---

# Test Skill
`

	_, err := parser.ParseContent(content, "test.md")
	if err == nil {
		t.Error("Expected error for missing description")
	}
}

func TestParseDirectory(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	parser := NewSkillParser(store)

	skillContent := `---
name: "test_skill"
description: "A test skill"
category: "test"
---

# Test Skill
`

	skillPath := filepath.Join(tempDir, "test_skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	skills, err := parser.ParseDirectory(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(skills))
	}

	if skills[0].Name != "test_skill" {
		t.Errorf("Expected skill name 'test_skill', got '%s'", skills[0].Name)
	}
}

func TestParseDirectoryEmpty(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	parser := NewSkillParser(store)

	skills, err := parser.ParseDirectory(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestParseDirectoryNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	parser := NewSkillParser(store)

	_, err := parser.ParseDirectory(context.Background(), "/nonexistent/directory")
	if err != nil {
		t.Fatalf("Expected no error for nonexistent directory, got %v", err)
	}
}
