package websocket

import (
	"context"
	"log"

	"github.com/wjffsx/miniclaw_go/internal/bus"
)

type Handler struct {
	server *Server
}

func NewHandler(server *Server) *Handler {
	return &Handler{
		server: server,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg *bus.Message) error {
	if msg.Channel != bus.ChannelWebSocket {
		return nil
	}

	log.Printf("Sending message to WebSocket client %s: %.40s...", msg.ChatID, msg.Content)

	if err := h.server.SendToClient(msg.ChatID, msg.Content); err != nil {
		log.Printf("Failed to send message to WebSocket: %v", err)
		return err
	}

	return nil
}
