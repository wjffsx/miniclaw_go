package bus

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryMessageBus_PublishAndSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewInMemoryMessageBus(ctx)
	bus.Start()
	defer bus.Close()

	received := make(chan *Message, 1)

	handler := func(ctx context.Context, msg *Message) error {
		received <- msg
		return nil
	}

	handlerID, err := bus.Subscribe(ChannelTelegram, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if handlerID == "" {
		t.Fatal("Handler ID should not be empty")
	}

	msg := &Message{
		ID:      "test-id",
		ChatID:  "test-chat",
		Content: "test message",
	}

	err = bus.Publish(ctx, ChannelTelegram, msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	select {
	case receivedMsg := <-received:
		if receivedMsg.ID != msg.ID {
			t.Errorf("Expected message ID %s, got %s", msg.ID, receivedMsg.ID)
		}
		if receivedMsg.Content != msg.Content {
			t.Errorf("Expected content %s, got %s", msg.Content, receivedMsg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestInMemoryMessageBus_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewInMemoryMessageBus(ctx)
	bus.Start()
	defer bus.Close()

	handler := func(ctx context.Context, msg *Message) error {
		return nil
	}

	handlerID, err := bus.Subscribe(ChannelTelegram, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = bus.Unsubscribe(ChannelTelegram, handlerID)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	err = bus.Unsubscribe(ChannelTelegram, handlerID)
	if err == nil {
		t.Fatal("Expected error when unsubscriving non-existent handler")
	}

	if err != ErrHandlerNotFound {
		t.Errorf("Expected ErrHandlerNotFound, got %v", err)
	}
}

func TestInMemoryMessageBus_MultipleSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewInMemoryMessageBus(ctx)
	bus.Start()
	defer bus.Close()

	receivedCount := 0
	received := make(chan bool, 2)

	handler1 := func(ctx context.Context, msg *Message) error {
		t.Logf("Handler 1 received message: %s", msg.ID)
		received <- true
		return nil
	}

	handler2 := func(ctx context.Context, msg *Message) error {
		t.Logf("Handler 2 received message: %s", msg.ID)
		received <- true
		return nil
	}

	handlerID1, err := bus.Subscribe(ChannelTelegram, handler1)
	if err != nil {
		t.Fatalf("Failed to subscribe handler1: %v", err)
	}
	t.Logf("Handler 1 subscribed with ID: %s", handlerID1)

	handlerID2, err := bus.Subscribe(ChannelTelegram, handler2)
	if err != nil {
		t.Fatalf("Failed to subscribe handler2: %v", err)
	}
	t.Logf("Handler 2 subscribed with ID: %s", handlerID2)

	msg := &Message{
		ID:      "test-id",
		ChatID:  "test-chat",
		Content: "test message",
	}

	err = bus.Publish(ctx, ChannelTelegram, msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}
	t.Logf("Message published: %s", msg.ID)

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 2; i++ {
		select {
		case <-received:
			receivedCount++
			t.Logf("Received count: %d", receivedCount)
		case <-time.After(3 * time.Second):
			t.Fatalf("Timeout waiting for message. Received count: %d", receivedCount)
		}
	}

	if receivedCount != 2 {
		t.Errorf("Expected 2 handlers to receive message, got %d", receivedCount)
	}
}

func TestInMemoryMessageBus_DifferentChannels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewInMemoryMessageBus(ctx)
	bus.Start()
	defer bus.Close()

	telegramReceived := false
	websocketReceived := false

	telegramHandler := func(ctx context.Context, msg *Message) error {
		if msg.Channel == ChannelTelegram {
			telegramReceived = true
		}
		return nil
	}

	websocketHandler := func(ctx context.Context, msg *Message) error {
		if msg.Channel == ChannelWebSocket {
			websocketReceived = true
		}
		return nil
	}

	_, err := bus.Subscribe(ChannelTelegram, telegramHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe telegram handler: %v", err)
	}

	_, err = bus.Subscribe(ChannelWebSocket, websocketHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe websocket handler: %v", err)
	}

	msg := &Message{
		ID:      "test-id",
		ChatID:  "test-chat",
		Content: "test message",
	}

	err = bus.Publish(ctx, ChannelTelegram, msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if !telegramReceived {
		t.Error("Telegram handler should have received the message")
	}

	if websocketReceived {
		t.Error("WebSocket handler should not have received the message")
	}
}

func TestMessage_Timestamp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := NewInMemoryMessageBus(ctx)
	bus.Start()
	defer bus.Close()

	handler := func(ctx context.Context, msg *Message) error {
		if msg.Timestamp.IsZero() {
			t.Error("Message timestamp should not be zero")
		}
		return nil
	}

	_, err := bus.Subscribe(ChannelTelegram, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	msg := &Message{
		ID:      "test-id",
		ChatID:  "test-chat",
		Content: "test message",
	}

	err = bus.Publish(ctx, ChannelTelegram, msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}
