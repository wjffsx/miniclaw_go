package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/storage"
)

func TestNewSkillFileWatcher(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if watcher == nil {
		t.Error("Expected watcher to be created")
	}

	if watcher.registry != registry {
		t.Error("Expected watcher registry to be set")
	}

	if watcher.parser != parser {
		t.Error("Expected watcher parser to be set")
	}

	if watcher.watcher == nil {
		t.Error("Expected watcher fsnotify watcher to be initialized")
	}

	if watcher.debounce == nil {
		t.Error("Expected debounce map to be initialized")
	}

	watcher.Stop()
}

func TestWatch(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.Watch(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWatchNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.Watch("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

func TestWatchDirectory(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWatchDirectoryNonExistent(t *testing.T) {
	store := storage.NewFileStorage(os.TempDir())
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.WatchDirectory("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestWatchFileChange(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

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

	time.Sleep(1 * time.Second)

	if registry.Count() != 1 {
		t.Errorf("Expected 1 skill after file creation, got %d", registry.Count())
	}
}

func TestWatchFileUpdate(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

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

	time.Sleep(1 * time.Second)

	if registry.Count() != 1 {
		t.Errorf("Expected 1 skill after file creation, got %d", registry.Count())
	}

	updatedContent := `---
name: "test_skill"
description: "Updated test skill"
category: "test"
---

# Test Skill Updated
`

	if err := os.WriteFile(skillPath, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update test skill file: %v", err)
	}

	time.Sleep(1 * time.Second)

	skill, exists := registry.GetByName("test_skill")
	if !exists {
		t.Fatal("Failed to get skill")
	}

	if skill.Description != "Updated test skill" {
		t.Errorf("Expected description 'Updated test skill', got '%s'", skill.Description)
	}
}

func TestWatchFileRemoval(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

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

	time.Sleep(1 * time.Second)

	if registry.Count() != 1 {
		t.Errorf("Expected 1 skill after file creation, got %d", registry.Count())
	}

	if err := os.Remove(skillPath); err != nil {
		t.Fatalf("Failed to remove test skill file: %v", err)
	}

	time.Sleep(1 * time.Second)

	if registry.Count() != 0 {
		t.Errorf("Expected 0 skills after file removal, got %d", registry.Count())
	}
}

func TestStop(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	watcher.Stop()

	if watcher.watcher == nil {
		t.Error("Expected watcher to be closed")
	}
}

func TestReloadDirectory(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

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

	err = watcher.ReloadDirectory(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Expected 1 skill after reload, got %d", registry.Count())
	}
}

func TestIsWatching(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	if watcher.IsWatching(tempDir) {
		t.Error("Expected directory to not be watched initially")
	}

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	if !watcher.IsWatching(tempDir) {
		t.Error("Expected directory to be watched")
	}
}

func TestGetWatchedPaths(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewFileStorage(tempDir)
	registry := NewSkillRegistry(store)
	parser := NewSkillParser(store)

	watcher, err := NewSkillFileWatcher(registry, parser)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	paths := watcher.GetWatchedPaths()
	if len(paths) != 0 {
		t.Errorf("Expected 0 watched paths initially, got %d", len(paths))
	}

	err = watcher.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	paths = watcher.GetWatchedPaths()
	if len(paths) != 1 {
		t.Errorf("Expected 1 watched path, got %d", len(paths))
	}
}
