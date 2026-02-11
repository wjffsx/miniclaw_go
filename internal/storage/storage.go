package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Storage interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte) error
	DeleteFile(ctx context.Context, path string) error
	ListFiles(ctx context.Context, prefix string) ([]string, error)
	FileExists(ctx context.Context, path string) (bool, error)
}

type SessionStorage interface {
	SaveMessage(ctx context.Context, chatID string, role string, content string) error
	GetMessages(ctx context.Context, chatID string, limit int) ([]Message, error)
	ClearSession(ctx context.Context, chatID string) error
	ListSessions(ctx context.Context) ([]string, error)
}

type MemoryStorage interface {
	GetMemory(ctx context.Context) (string, error)
	SetMemory(ctx context.Context, content string) error
	GetDailyNote(ctx context.Context, date string) (string, error)
	SetDailyNote(ctx context.Context, date string, content string) error
	GetConfig(ctx context.Context, key string) (string, error)
	SetConfig(ctx context.Context, key string, value string) error
}

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{
		basePath: basePath,
	}
}

func (fs *FileStorage) ReadFile(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := filepath.Join(fs.basePath, path)
	return os.ReadFile(fullPath)
}

func (fs *FileStorage) WriteFile(ctx context.Context, path string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(fs.basePath, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

func (fs *FileStorage) DeleteFile(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := filepath.Join(fs.basePath, path)
	return os.Remove(fullPath)
}

func (fs *FileStorage) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := filepath.Join(fs.basePath, prefix)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	var files []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(fs.basePath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

func (fs *FileStorage) FileExists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	fullPath := filepath.Join(fs.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type FileSystemSessionStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewFileSystemSessionStorage(basePath string) *FileSystemSessionStorage {
	return &FileSystemSessionStorage{
		basePath: basePath,
	}
}

func (s *FileSystemSessionStorage) SaveMessage(ctx context.Context, chatID string, role string, content string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionDir := filepath.Join(s.basePath, "sessions", chatID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	sessionFile := filepath.Join(sessionDir, "messages.jsonl")

	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: 0,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msgData = append(msgData, '\n')

	file, err := os.OpenFile(sessionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(msgData); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (s *FileSystemSessionStorage) GetMessages(ctx context.Context, chatID string, limit int) ([]Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionFile := filepath.Join(s.basePath, "sessions", chatID, "messages.jsonl")

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	messages := make([]Message, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		messages = append(messages, msg)
	}

	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

func (s *FileSystemSessionStorage) ClearSession(ctx context.Context, chatID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sessionDir := filepath.Join(s.basePath, "sessions", chatID)
	return os.RemoveAll(sessionDir)
}

func (s *FileSystemSessionStorage) ListSessions(ctx context.Context) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionsDir := filepath.Join(s.basePath, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			sessions = append(sessions, entry.Name())
		}
	}

	return sessions, nil
}

type FileSystemMemoryStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewFileSystemMemoryStorage(basePath string) *FileSystemMemoryStorage {
	return &FileSystemMemoryStorage{
		basePath: basePath,
	}
}

func (m *FileSystemMemoryStorage) GetMemory(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	memoryFile := filepath.Join(m.basePath, "memory", "MEMORY.md")

	data, err := os.ReadFile(memoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read memory file: %w", err)
	}

	return string(data), nil
}

func (m *FileSystemMemoryStorage) SetMemory(ctx context.Context, content string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	memoryDir := filepath.Join(m.basePath, "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	return os.WriteFile(memoryFile, []byte(content), 0644)
}

func (m *FileSystemMemoryStorage) GetDailyNote(ctx context.Context, date string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	noteFile := filepath.Join(m.basePath, "memory", date+".md")

	data, err := os.ReadFile(noteFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read daily note: %w", err)
	}

	return string(data), nil
}

func (m *FileSystemMemoryStorage) SetDailyNote(ctx context.Context, date string, content string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	memoryDir := filepath.Join(m.basePath, "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	noteFile := filepath.Join(memoryDir, date+".md")

	return os.WriteFile(noteFile, []byte(content), 0644)
}

func (m *FileSystemMemoryStorage) GetConfig(ctx context.Context, key string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	configFile := filepath.Join(m.basePath, "config", "config.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if value, ok := config[key]; ok {
		return value, nil
	}

	return "", nil
}

func (m *FileSystemMemoryStorage) SetConfig(ctx context.Context, key string, value string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	configDir := filepath.Join(m.basePath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.json")

	var config map[string]string

	data, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		config = make(map[string]string)
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	config[key] = value

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configFile, configData, 0644)
}
