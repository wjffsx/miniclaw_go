package telegram

import (
	"context"
	"testing"
)

func TestNewBot(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	if bot == nil {
		t.Error("Expected bot to be created")
	}

	if bot.token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", bot.token)
	}
}

func TestBotStart(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	err := bot.Start()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !bot.IsRunning() {
		t.Error("Expected bot to be running")
	}

	bot.Stop()
}

func TestBotStop(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	bot.Start()

	err := bot.Stop()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bot.IsRunning() {
		t.Error("Expected bot to be stopped")
	}
}

func TestBotIsRunning(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	if bot.IsRunning() {
		t.Error("Expected bot to not be running initially")
	}

	bot.Start()

	if !bot.IsRunning() {
		t.Error("Expected bot to be running")
	}

	bot.Stop()
}

func TestBotGetToken(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	token := bot.GetToken()
	if token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", token)
	}
}

func TestBotSetPollTimeout(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	bot.SetPollTimeout(60)

	if bot.pollTimeout != 60 {
		t.Errorf("Expected poll timeout 60, got %d", bot.pollTimeout)
	}
}

func TestBotSetPollInterval(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	bot.SetPollInterval(5)

	if bot.pollInterval.Seconds() != 5 {
		t.Errorf("Expected poll interval 5, got %d", bot.pollInterval)
	}
}

func TestBotHandleUpdate(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	update := &Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 1,
			From: &User{
				ID:        123,
				FirstName: "Test",
			},
			Chat: &Chat{
				ID:   123456,
				Type: "private",
			},
			Date: 1234567890,
			Text: "test",
		},
	}

	bot.handleUpdate(update)
}

func TestBotHandleUpdateNilMessage(t *testing.T) {
	bot := NewBot(&Config{Token: "test-token"}, nil, context.Background())

	update := &Update{
		UpdateID: 1,
		Message:  nil,
	}

	bot.handleUpdate(update)
}
