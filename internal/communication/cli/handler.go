package cli

import (
	"context"
	"fmt"
	"log"

	"github.com/wjffsx/miniclaw_go/internal/bus"
)

type Handler struct {
	cli *CLI
}

func NewHandler(cli *CLI) *Handler {
	return &Handler{
		cli: cli,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg *bus.Message) error {
	if msg.Channel != bus.ChannelCLI {
		return nil
	}

	log.Printf("CLI received response: %.40s...", msg.Content)

	fmt.Printf("\nResponse: %s\n%s", msg.Content, prompt)
	return nil
}