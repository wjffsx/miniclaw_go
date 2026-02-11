package skills

import (
	"testing"
)

func TestNewSkillIndex(t *testing.T) {
	index := NewSkillIndex()

	if index.byName == nil {
		t.Error("Expected byName map to be initialized")
	}

	if index.byTag == nil {
		t.Error("Expected byTag map to be initialized")
	}

	if index.byCategory == nil {
		t.Error("Expected byCategory map to be initialized")
	}

	if index.byKeyword == nil {
		t.Error("Expected byKeyword map to be initialized")
	}
}

func TestAdd(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	skill.AddTag("tag2")

	index.Add(skill)

	if len(index.byName) != 1 {
		t.Errorf("Expected 1 skill in byName, got %d", len(index.byName))
	}

	if len(index.byTag) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(index.byTag))
	}

	if len(index.byCategory) != 1 {
		t.Errorf("Expected 1 category, got %d", len(index.byCategory))
	}

	if len(index.byKeyword) == 0 {
		t.Error("Expected keywords to be extracted")
	}
}

func TestRemove(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")

	index.Add(skill)
	index.Remove(skill.ID)

	if len(index.byName) != 0 {
		t.Errorf("Expected 0 skills in byName after removal, got %d", len(index.byName))
	}

	if len(index.byTag) != 0 {
		t.Errorf("Expected 0 tags after removal, got %d", len(index.byTag))
	}
}

func TestRemoveNonExistent(t *testing.T) {
	index := NewSkillIndex()

	index.Remove("nonexistent")

	if len(index.byName) != 0 {
		t.Error("Expected no change when removing nonexistent skill")
	}
}

func TestGetAll(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	skill.AddTag("tag2")
	index.Add(skill)

	results := index.GetAll()
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].ID != skill.ID {
		t.Errorf("Expected skill ID '%s', got '%s'", skill.ID, results[0].ID)
	}
}

func TestGetByTag(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	skill.AddTag("tag2")
	index.Add(skill)

	results := index.GetByTag("tag1")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	results = index.GetByTag("nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nonexistent tag, got %d", len(results))
	}
}

func TestGetByCategory(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	index.Add(skill)

	results := index.GetByCategory("test-category")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	results = index.GetByCategory("nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nonexistent category, got %d", len(results))
	}
}

func TestSearch(t *testing.T) {
	index := NewSkillIndex()
	skill := NewSkill("test", "test description", "test-category")
	skill.AddTag("tag1")
	index.Add(skill)

	results := index.Search("test")
	if len(results) == 0 {
		t.Error("Expected results for 'test'")
	}

	results = index.Search("tag1")
	if len(results) == 0 {
		t.Error("Expected results for 'tag1'")
	}

	results = index.Search("test-category")
	if len(results) == 0 {
		t.Error("Expected results for 'test-category'")
	}

	results = index.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nonexistent term, got %d", len(results))
	}
}
