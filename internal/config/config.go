package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Telegram  TelegramConfig
	WebSocket WebSocketConfig
	LLM       LLMConfig
	Storage   StorageConfig
	Tools     ToolsConfig
	Search    SearchConfig
	Proxy     ProxyConfig
}

type TelegramConfig struct {
	Enabled bool
	Token   string
	Webhook string
}

type WebSocketConfig struct {
	Enabled bool
	Port    int
	Host    string
}

type LLMConfig struct {
	Provider     string
	APIKey       string
	Model        string
	MaxTokens    int
	Temperature  float64
	LocalModel   LocalModelConfig
	Models       []ModelConfig
	DefaultModel string
}

type ModelConfig struct {
	Name        string
	Provider    string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	LocalModel  LocalModelConfig
}

type LocalModelConfig struct {
	Enabled bool
	Path    string
	Type    string
}

type StorageConfig struct {
	BasePath string
}

type ToolsConfig struct {
	WebSearch WebSearchConfig
}

type SearchConfig struct {
	BraveAPIKey string
}

type WebSearchConfig struct {
	Enabled  bool
	APIKey   string
	Provider string
}

type ProxyConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
}

type ConfigManager interface {
	GetConfig() *Config
	Reload() error
	Save() error
	GetString(key string) (string, error)
	GetInt(key string) (int, error)
	GetBool(key string) (bool, error)
	SetString(key string, value string) error
	SetInt(key string, value int) error
	SetBool(key string, value bool) error
}

type FileConfigManager struct {
	mu       sync.RWMutex
	config   *Config
	path     string
	watchers []ConfigWatcher
}

type ConfigWatcher interface {
	OnConfigChange(config *Config)
}

func NewFileConfigManager(path string) (*FileConfigManager, error) {
	cm := &FileConfigManager{
		path:     path,
		watchers: make([]ConfigWatcher, 0),
	}

	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cm, nil
}

func (cm *FileConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, err := cm.loadFromFile()
	if err != nil {
		return err
	}

	cm.config = config

	for _, watcher := range cm.watchers {
		watcher.OnConfigChange(cm.config)
	}

	return nil
}

func (cm *FileConfigManager) loadFromFile() (*Config, error) {
	if _, err := os.Stat(cm.path); os.IsNotExist(err) {
		return cm.getDefaultConfig(), nil
	}

	return cm.getDefaultConfig(), nil
}

func (cm *FileConfigManager) getDefaultConfig() *Config {
	return &Config{
		Telegram: TelegramConfig{
			Enabled: true,
		},
		WebSocket: WebSocketConfig{
			Enabled: true,
			Port:    18789,
			Host:    "0.0.0.0",
		},
		LLM: LLMConfig{
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-5",
			MaxTokens:   4096,
			Temperature: 0.7,
			LocalModel: LocalModelConfig{
				Enabled: false,
				Path:    "./models/llama-2-7b.gguf",
				Type:    "llama",
			},
		},
		Storage: StorageConfig{
			BasePath: "./data",
		},
		Tools: ToolsConfig{
			WebSearch: WebSearchConfig{
				Enabled:  false,
				Provider: "brave",
			},
		},
		Search: SearchConfig{
			BraveAPIKey: "",
		},
		Proxy: ProxyConfig{
			Enabled: false,
		},
	}
}

func (cm *FileConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *FileConfigManager) Reload() error {
	return cm.Load()
}

func (cm *FileConfigManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	dir := filepath.Dir(cm.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

func (cm *FileConfigManager) GetString(key string) (string, error) {
	return "", nil
}

func (cm *FileConfigManager) GetInt(key string) (int, error) {
	return 0, nil
}

func (cm *FileConfigManager) GetBool(key string) (bool, error) {
	return false, nil
}

func (cm *FileConfigManager) SetString(key string, value string) error {
	return nil
}

func (cm *FileConfigManager) SetInt(key string, value int) error {
	return nil
}

func (cm *FileConfigManager) SetBool(key string, value bool) error {
	return nil
}

func (cm *FileConfigManager) AddWatcher(watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.watchers = append(cm.watchers, watcher)
}

func (cm *FileConfigManager) RemoveWatcher(watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, w := range cm.watchers {
		if w == watcher {
			cm.watchers = append(cm.watchers[:i], cm.watchers[i+1:]...)
			break
		}
	}
}
