package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wjffsx/miniclaw_go/internal/bus"
)

const (
	defaultMaxClients = 10
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxMessageSize    = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketConn interface {
	SetReadLimit(limit int64)
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
}

type Client struct {
	conn   WebSocketConn
	chatID string
	send   chan []byte
	server *Server
	mu     sync.Mutex
}

type Server struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	messageBus bus.MessageBus
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	started    bool
}

type Message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	ChatID  string `json:"chat_id,omitempty"`
}

type Config struct {
	Port       int
	MaxClients int
}

func NewServer(cfg *Config, messageBus bus.MessageBus, ctx context.Context) *Server {
	serverCtx, cancel := context.WithCancel(ctx)

	return &Server{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		messageBus: messageBus,
		ctx:        serverCtx,
		cancel:     cancel,
	}
}

func (s *Server) Start(port int) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	log.Printf("Starting WebSocket server on port %d...", port)

	go s.run()

	addr := fmt.Sprintf(":%d", port)
	log.Printf("WebSocket server listening on %s", addr)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", s.handleWebSocket)
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.mu.Unlock()

	log.Println("Stopping WebSocket server...")
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Server) run() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("WebSocket server stopped")
			return
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected: %s", client.chatID)

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				s.mu.Lock()
				delete(s.clients, client)
				s.mu.Unlock()
				close(client.send)
				log.Printf("Client disconnected: %s", client.chatID)
			}

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
		chatID: fmt.Sprintf("ws_%d", time.Now().UnixNano()),
	}

	s.register <- client

	go s.writePump(client)
	go s.readPump(client)
}

func (s *Server) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid JSON message: %v", err)
			continue
		}

		if msg.Type == "message" && msg.Content != "" {
			chatID := client.chatID
			if msg.ChatID != "" {
				chatID = msg.ChatID
				client.mu.Lock()
				client.chatID = chatID
				client.mu.Unlock()
			}

			log.Printf("WS message from %s: %.40s...", chatID, msg.Content)

			busMsg := &bus.Message{
				ID:      fmt.Sprintf("websocket-%d", time.Now().UnixNano()),
				Channel: bus.ChannelWebSocket,
				ChatID:  chatID,
				Content: msg.Content,
			}

			if err := s.messageBus.Publish(s.ctx, bus.ChannelWebSocket, busMsg); err != nil {
				log.Printf("Failed to publish message to bus: %v", err)
			}
		}
	}
}

func (s *Server) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) SendToClient(chatID, text string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client.chatID == chatID {
			resp := Message{
				Type:    "response",
				Content: text,
				ChatID:  chatID,
			}

			data, err := json.Marshal(resp)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			select {
			case client.send <- data:
				return nil
			default:
				return fmt.Errorf("client send buffer full")
			}
		}
	}

	return fmt.Errorf("client not found: %s", chatID)
}

func (s *Server) Broadcast(text string) error {
	resp := Message{
		Type:    "response",
		Content: text,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case s.broadcast <- data:
		return nil
	default:
		return fmt.Errorf("broadcast buffer full")
	}
}

func (s *Server) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

func NewClient(conn WebSocketConn, chatID string, server *Server) *Client {
	return &Client{
		conn:   conn,
		chatID: chatID,
		send:   make(chan []byte, 256),
		server: server,
	}
}
