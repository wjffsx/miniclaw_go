package skills

import (
	"testing"
	"time"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")

	if skill.ID == "" {
		t.Error("Expected skill ID to be generated")
	}

	if skill.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", skill.Name)
	}

	if skill.Description != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", skill.Description)
	}

	if skill.Category != "test-category" {
		t.Errorf("Expected category 'test-category', got '%s'", skill.Category)
	}

	if !skill.Enabled {
		t.Error("Expected skill to be enabled by default")
	}

	if len(skill.Tags) != 0 {
		t.Errorf("Expected empty tags, got %d", len(skill.Tags))
	}
}

func TestSkillUpdate(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")
	originalTime := skill.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	skill.Update()

	if !skill.UpdatedAt.After(originalTime) {
		t.Error("Expected UpdatedAt to be updated")
	}
}

func TestSkillAddTag(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")

	skill.AddTag("tag1")
	if len(skill.Tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(skill.Tags))
	}

	if skill.Tags[0] != "tag1" {
		t.Errorf("Expected tag 'tag1', got '%s'", skill.Tags[0])
	}

	skill.AddTag("tag1")
	if len(skill.Tags) != 1 {
		t.Error("Expected duplicate tag to not be added")
	}

	skill.AddTag("tag2")
	if len(skill.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(skill.Tags))
	}
}

func TestSkillRemoveTag(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	skill.AddTag("tag2")

	skill.RemoveTag("tag1")
	if len(skill.Tags) != 1 {
		t.Errorf("Expected 1 tag after removal, got %d", len(skill.Tags))
	}

	if skill.Tags[0] != "tag2" {
		t.Errorf("Expected tag 'tag2', got '%s'", skill.Tags[0])
	}

	skill.RemoveTag("nonexistent")
	if len(skill.Tags) != 1 {
		t.Error("Expected tag count to remain the same when removing nonexistent tag")
	}
}

func TestSkillHasTag(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")

	hasTag := false
	for _, tag := range skill.Tags {
		if tag == "tag1" {
			hasTag = true
			break
		}
	}

	if !hasTag {
		t.Error("Expected skill to have tag 'tag1'")
	}

	hasTag = false
	for _, tag := range skill.Tags {
		if tag == "nonexistent" {
			hasTag = true
			break
		}
	}

	if hasTag {
		t.Error("Expected skill to not have nonexistent tag")
	}
}

func TestSkillSetMetadata(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")

	skill.SetMetadata("key1", "value1")
	if skill.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata 'key1' to be 'value1', got '%s'", skill.Metadata["key1"])
	}

	skill.SetMetadata("key1", "value2")
	if skill.Metadata["key1"] != "value2" {
		t.Errorf("Expected metadata 'key1' to be updated to 'value2', got '%s'", skill.Metadata["key1"])
	}
}

func TestSkillGetMetadata(t *testing.T) {
	skill := NewSkill("test", "test description", "test-category")

	value, exists := skill.GetMetadata("nonexistent")
	if exists {
		t.Errorf("Expected nonexistent metadata to not exist, got value '%s'", value)
	}

	skill.SetMetadata("key1", "value1")
	value, exists = skill.GetMetadata("key1")
	if !exists {
		t.Error("Expected metadata 'key1' to exist")
	}

	if value != "value1" {
		t.Errorf("Expected metadata 'key1' to be 'value1', got '%s'", value)
	}
}
