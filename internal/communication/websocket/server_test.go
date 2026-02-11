package websocket

import (
	"context"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	if server == nil {
		t.Error("Expected server to be created")
	}

	if server.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if server.register == nil {
		t.Error("Expected register channel to be initialized")
	}

	if server.unregister == nil {
		t.Error("Expected unregister channel to be initialized")
	}

	if server.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}
}

func TestServerStart(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	err := server.Start(8081)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	server.Stop()
}

func TestServerStartAlreadyRunning(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	err := server.Start(8082)
	if err != nil {
		t.Fatalf("Expected no error on first start, got %v", err)
	}

	err = server.Start(8083)
	if err == nil {
		t.Error("Expected error when starting already running server")
	}

	server.Stop()
}

func TestServerStop(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	server.Start(8084)

	err := server.Stop()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestServerStopNotRunning(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	err := server.Stop()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestServerHandleWebSocket(t *testing.T) {
	server := NewServer(nil, nil, context.Background())
	server.Start(8085)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	if !server.IsRunning() {
		t.Error("Expected server to be running")
	}
}

func TestServerBroadcast(t *testing.T) {
	server := NewServer(nil, nil, context.Background())
	server.Start(8086)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	message := "test message"
	err := server.Broadcast(message)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestServerGetClientCount(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	count := server.GetClientCount()
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestServerIsRunning(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	if server.IsRunning() {
		t.Error("Expected server to not be running initially")
	}

	server.Start(8087)

	if !server.IsRunning() {
		t.Error("Expected server to be running")
	}

	server.Stop()
}

func TestNewClient(t *testing.T) {
	server := NewServer(nil, nil, context.Background())

	conn := &mockConn{}
	client := NewClient(conn, "test-chat-id", server)

	if client == nil {
		t.Error("Expected client to be created")
	}

	if client.chatID != "test-chat-id" {
		t.Errorf("Expected chat ID 'test-chat-id', got '%s'", client.chatID)
	}

	if client.send == nil {
		t.Error("Expected send channel to be initialized")
	}
}

func TestClientReadPump(t *testing.T) {
	server := NewServer(nil, nil, context.Background())
	conn := &mockConn{}
	_ = NewClient(conn, "test-chat-id", server)

	time.Sleep(100 * time.Millisecond)
}

func TestClientWritePump(t *testing.T) {
	server := NewServer(nil, nil, context.Background())
	conn := &mockConn{}
	client := NewClient(conn, "test-chat-id", server)

	client.send <- []byte("test message")

	time.Sleep(100 * time.Millisecond)
}

func TestClientClose(t *testing.T) {
	server := NewServer(nil, nil, context.Background())
	conn := &mockConn{}
	client := NewClient(conn, "test-chat-id", server)

	close(client.send)

	time.Sleep(100 * time.Millisecond)
}

type mockConn struct{}

func (m *mockConn) SetReadLimit(limit int64) {}

func (m *mockConn) ReadMessage() (messageType int, p []byte, err error) {
	return 1, []byte("test"), nil
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetPongHandler(h func(appData string) error) {}
func (m *mockConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}
