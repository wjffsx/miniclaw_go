package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileConfigManager(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager == nil {
		t.Error("Expected manager to be created")
	}

	if manager.path != configPath {
		t.Errorf("Expected path '%s', got '%s'", configPath, manager.path)
	}
}

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("Expected config to be loaded")
	}
}

func TestLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nonexistent.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Load()
	if err != nil {
		t.Fatalf("Expected no error for nonexistent file, got %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("Expected default config to be loaded")
	}
}

func TestLoadWithValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
telegram:
  enabled: false
  token: ""
  webhook: ""

websocket:
  enabled: true
  port: 18789
  host: "0.0.0.0"

llm:
  provider: "anthropic"
  api_key: ""
  model: "claude-sonnet-4-5"
  max_tokens: 4096
  temperature: 0.7
  local_model:
    enabled: false
    path: "./models/llama-2-7b.gguf"
    type: "llama"

storage:
  base_path: "./data"

tools:
  web_search:
    enabled: false
    api_key: ""
    provider: "brave"

skills:
  enabled: true
  directory: "./data/skills"
  auto_reload: true
  max_active: 5
  selection:
    method: "hybrid"
    threshold: 0.5

mcp:
  enabled: false
  clients: []

scheduler:
  enabled: false
  tasks_file: "./data/tasks.json"
  auto_start: true
  tick_interval: 1

search:
  brave_api_key: ""

proxy:
  enabled: false
  host: "127.0.0.1"
  port: 7890
  username: ""
  password: ""
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("Expected config to be loaded")
	}

	if config.WebSocket.Port != 18789 {
		t.Errorf("Expected port 18789, got %d", config.WebSocket.Port)
	}

	if config.Skills.Enabled != true {
		t.Error("Expected skills to be enabled")
	}
}

func TestGetConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("Expected config to be returned")
	}

	if config.LLM.Provider == "" {
		t.Error("Expected LLM provider to be set")
	}
}

func TestReload(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	err = manager.Reload()
	if err != nil {
		t.Fatalf("Expected no error on reload, got %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("Expected config to be reloaded")
	}
}

func TestRegisterWatcher(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	watcher := &mockConfigWatcher{}
	manager.AddWatcher(watcher)

	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !watcher.called {
		t.Error("Expected watcher to be called")
	}
}

func TestUnregisterWatcher(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	watcher := &mockConfigWatcher{}
	manager.AddWatcher(watcher)
	manager.RemoveWatcher(watcher)

	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if watcher.called {
		t.Error("Expected watcher to not be called after unregister")
	}
}

func TestGetDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	manager, err := NewFileConfigManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	config := manager.getDefaultConfig()
	if config == nil {
		t.Error("Expected default config to be returned")
	}

	if config.LLM.Provider == "" {
		t.Error("Expected default LLM provider to be set")
	}

	if config.WebSocket.Port == 0 {
		t.Error("Expected default WebSocket port to be set")
	}

	if !config.Skills.Enabled {
		t.Error("Expected skills to be enabled by default")
	}
}

func TestLoadFromFileInvalid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	invalidContent := `invalid yaml content`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager, err := NewFileConfigManager(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if manager != nil {
		t.Error("Expected manager to be nil for invalid YAML")
	}
}

type mockConfigWatcher struct {
	called bool
}

func (m *mockConfigWatcher) OnConfigChange(config *Config) {
	m.called = true
}
