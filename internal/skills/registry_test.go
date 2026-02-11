package skills

import (
	"context"
	"os"
	"testing"

	"github.com/wjffsx/miniclaw_go/internal/storage"
)

func TestNewSkillRegistry(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	if registry == nil {
		t.Error("Expected registry to be created")
	}

	if registry.skills == nil {
		t.Error("Expected skills map to be initialized")
	}

	if registry.index == nil {
		t.Error("Expected index to be initialized")
	}
}

func TestRegister(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(registry.skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(registry.skills))
	}

	if _, exists := registry.skills[skill.ID]; !exists {
		t.Error("Expected skill to be registered")
	}
}

func TestRegisterEmptyID(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	skill.ID = ""

	err := registry.Register(skill)
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	skill.Name = ""

	err := registry.Register(skill)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Expected no error on first registration, got %v", err)
	}

	skill.Description = "updated description"
	err = registry.Register(skill)
	if err != nil {
		t.Fatalf("Expected no error on duplicate registration, got %v", err)
	}

	if registry.skills[skill.ID].Description != "updated description" {
		t.Error("Expected skill to be updated")
	}
}

func TestUnregister(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	err := registry.Unregister(skill.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(registry.skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(registry.skills))
	}
}

func TestUnregisterNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent skill")
	}
}

func TestGet(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	retrieved, exists := registry.Get(skill.ID)
	if !exists {
		t.Error("Expected skill to exist")
	}

	if retrieved.ID != skill.ID {
		t.Errorf("Expected skill ID '%s', got '%s'", skill.ID, retrieved.ID)
	}
}

func TestGetNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	_, exists := registry.Get("nonexistent")
	if exists {
		t.Error("Expected skill to not exist")
	}
}

func TestListAll(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill1 := NewSkill("test1", "test description 1", "test-category")
	skill2 := NewSkill("test2", "test description 2", "test-category")

	registry.Register(skill1)
	registry.Register(skill2)

	skills := registry.ListAll()
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}
}

func TestListAllEmpty(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skills := registry.ListAll()
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestRegistryCount(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	if registry.Count() != 0 {
		t.Errorf("Expected count 0, got %d", registry.Count())
	}

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}
}

func TestEnable(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	skill.Enabled = false
	registry.Register(skill)

	err := registry.Enable(skill.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !registry.skills[skill.ID].Enabled {
		t.Error("Expected skill to be enabled")
	}
}

func TestEnableNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	err := registry.Enable("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent skill")
	}
}

func TestDisable(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	err := registry.Disable(skill.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if registry.skills[skill.ID].Enabled {
		t.Error("Expected skill to be disabled")
	}
}

func TestDisableNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	err := registry.Disable("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent skill")
	}
}

func TestRegistryClear(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	skill := NewSkill("test", "test description", "test-category")
	registry.Register(skill)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", registry.Count())
	}
}

func TestLoadFromDirectory(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)

	skillContent := `---
name: "test_skill"
description: "A test skill"
category: "test"
---

# Test Skill
`

	skillPath := tempDir + "/test_skill.md"
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to create test skill file: %v", err)
	}

	err := registry.LoadFromDirectory(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected 1 skill, got %d", registry.Count())
	}
}

func TestLoadFromDirectoryNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)

	err := registry.LoadFromDirectory(context.Background(), "/nonexistent/directory")
	if err != nil {
		t.Fatalf("Expected no error for nonexistent directory, got %v", err)
	}
}
