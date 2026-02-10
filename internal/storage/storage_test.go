package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorage(t *testing.T) {
	tempDir := t.TempDir()
	fs := NewFileStorage(tempDir)

	ctx := context.Background()

	t.Run("WriteFile", func(t *testing.T) {
		data := []byte("test content")
		err := fs.WriteFile(ctx, "test.txt", data)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		data, err := fs.ReadFile(ctx, "test.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if string(data) != "test content" {
			t.Errorf("expected 'test content', got '%s'", string(data))
		}
	})

	t.Run("FileExists", func(t *testing.T) {
		exists, err := fs.FileExists(ctx, "test.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	t.Run("FileNotExists", func(t *testing.T) {
		exists, err := fs.FileExists(ctx, "nonexistent.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if exists {
			t.Error("expected file to not exist")
		}
	})

	t.Run("WriteSubdir", func(t *testing.T) {
		data := []byte("subdir content")
		err := fs.WriteFile(ctx, "subdir/test.txt", data)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("ReadSubdir", func(t *testing.T) {
		data, err := fs.ReadFile(ctx, "subdir/test.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if string(data) != "subdir content" {
			t.Errorf("expected 'subdir content', got '%s'", string(data))
		}
	})

	t.Run("ListFiles", func(t *testing.T) {
		files, err := fs.ListFiles(ctx, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(files) < 2 {
			t.Errorf("expected at least 2 files, got %d", len(files))
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		err := fs.DeleteFile(ctx, "test.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		exists, err := fs.FileExists(ctx, "test.txt")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if exists {
			t.Error("expected file to not exist after deletion")
		}
	})
}

func TestFileSystemSessionStorage(t *testing.T) {
	tempDir := t.TempDir()
	ss := NewFileSystemSessionStorage(tempDir)

	ctx := context.Background()

	chatID := "test-chat-123"

	t.Run("SaveMessage", func(t *testing.T) {
		err := ss.SaveMessage(ctx, chatID, "user", "Hello, world!")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = ss.SaveMessage(ctx, chatID, "assistant", "Hi there!")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetMessages", func(t *testing.T) {
		messages, err := ss.GetMessages(ctx, chatID, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messages))
		}

		if messages[0].Role != "user" || messages[0].Content != "Hello, world!" {
			t.Errorf("unexpected first message: %+v", messages[0])
		}

		if messages[1].Role != "assistant" || messages[1].Content != "Hi there!" {
			t.Errorf("unexpected second message: %+v", messages[1])
		}
	})

	t.Run("GetMessagesWithLimit", func(t *testing.T) {
		err := ss.SaveMessage(ctx, chatID, "user", "Third message")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		messages, err := ss.GetMessages(ctx, chatID, 2)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("expected 2 messages with limit, got %d", len(messages))
		}

		if messages[0].Content != "Hi there!" {
			t.Errorf("expected 'Hi there!' as first message with limit, got '%s'", messages[0].Content)
		}
	})

	t.Run("GetMessagesNonExistent", func(t *testing.T) {
		messages, err := ss.GetMessages(ctx, "nonexistent-chat", 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("expected 0 messages for nonexistent chat, got %d", len(messages))
		}
	})

	t.Run("ListSessions", func(t *testing.T) {
		sessions, err := ss.ListSessions(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("expected 1 session, got %d", len(sessions))
		}
		if sessions[0] != chatID {
			t.Errorf("expected chat ID '%s', got '%s'", chatID, sessions[0])
		}
	})

	t.Run("ClearSession", func(t *testing.T) {
		err := ss.ClearSession(ctx, chatID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		messages, err := ss.GetMessages(ctx, chatID, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(messages) != 0 {
			t.Errorf("expected 0 messages after clear, got %d", len(messages))
		}
	})
}

func TestFileSystemMemoryStorage(t *testing.T) {
	tempDir := t.TempDir()
	ms := NewFileSystemMemoryStorage(tempDir)

	ctx := context.Background()

	t.Run("SetMemory", func(t *testing.T) {
		content := "This is a test memory."
		err := ms.SetMemory(ctx, content)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetMemory", func(t *testing.T) {
		content, err := ms.GetMemory(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if content != "This is a test memory." {
			t.Errorf("expected 'This is a test memory.', got '%s'", content)
		}
	})

	t.Run("SetDailyNote", func(t *testing.T) {
		date := "2024-01-15"
		content := "Today I learned about Go storage."
		err := ms.SetDailyNote(ctx, date, content)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetDailyNote", func(t *testing.T) {
		date := "2024-01-15"
		content, err := ms.GetDailyNote(ctx, date)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if content != "Today I learned about Go storage." {
			t.Errorf("expected 'Today I learned about Go storage.', got '%s'", content)
		}
	})

	t.Run("GetDailyNoteNonExistent", func(t *testing.T) {
		date := "2024-01-16"
		content, err := ms.GetDailyNote(ctx, date)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if content != "" {
			t.Errorf("expected empty content for nonexistent note, got '%s'", content)
		}
	})

	t.Run("SetConfig", func(t *testing.T) {
		err := ms.SetConfig(ctx, "test_key", "test_value")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		value, err := ms.GetConfig(ctx, "test_key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if value != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", value)
		}
	})

	t.Run("GetConfigNonExistent", func(t *testing.T) {
		value, err := ms.GetConfig(ctx, "nonexistent_key")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if value != "" {
			t.Errorf("expected empty value for nonexistent key, got '%s'", value)
		}
	})

	t.Run("SetMultipleConfig", func(t *testing.T) {
		err := ms.SetConfig(ctx, "key1", "value1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		err = ms.SetConfig(ctx, "key2", "value2")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		value1, err := ms.GetConfig(ctx, "key1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if value1 != "value1" {
			t.Errorf("expected 'value1', got '%s'", value1)
		}

		value2, err := ms.GetConfig(ctx, "key2")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if value2 != "value2" {
			t.Errorf("expected 'value2', got '%s'", value2)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	fs := NewFileStorage(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("ReadFileCancelled", func(t *testing.T) {
		_, err := fs.ReadFile(ctx, "test.txt")
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("WriteFileCancelled", func(t *testing.T) {
		err := fs.WriteFile(ctx, "test.txt", []byte("test"))
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	fs := NewFileStorage(tempDir)

	ctx := context.Background()

	err := fs.WriteFile(ctx, "config/app.yaml", []byte("app config"))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = fs.WriteFile(ctx, "memory/MEMORY.md", []byte("memory content"))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = fs.WriteFile(ctx, "sessions/chat1/messages.jsonl", []byte("session data"))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	configPath := filepath.Join(tempDir, "config", "app.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to exist")
	}

	memoryPath := filepath.Join(tempDir, "memory", "MEMORY.md")
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		t.Error("expected memory file to exist")
	}

	sessionPath := filepath.Join(tempDir, "sessions", "chat1", "messages.jsonl")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("expected session file to exist")
	}
}
