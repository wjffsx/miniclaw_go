package telegram

import (
	"context"
	"log"

	"github.com/wjffsx/miniclaw_go/internal/bus"
)

type Handler struct {
	bot *Bot
}

func NewHandler(bot *Bot) *Handler {
	return &Handler{
		bot: bot,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg *bus.Message) error {
	if msg.Channel != bus.ChannelTelegram {
		return nil
	}

	log.Printf("Sending message to Telegram chat %s: %.40s...", msg.ChatID, msg.Content)

	if err := h.bot.SendMessage(msg.ChatID, msg.Content); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
		return err
	}

	return nil
}
