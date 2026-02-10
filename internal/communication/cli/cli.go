package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wjffsx/miniclaw_go/internal/bus"
)

const (
	prompt = "mimi> "
)

type CLI struct {
	scanner    *bufio.Scanner
	messageBus bus.MessageBus
	ctx        context.Context
	commands   map[string]Command
	chatID     string
}

type Command struct {
	Name        string
	Description string
	Handler     func(args []string) error
	Usage       string
}

func NewCLI(messageBus bus.MessageBus, ctx context.Context) *CLI {
	cli := &CLI{
		scanner:    bufio.NewScanner(os.Stdin),
		messageBus: messageBus,
		ctx:        ctx,
		commands:   make(map[string]Command),
		chatID:     "cli",
	}

	cli.registerCommands()
	return cli
}

func (c *CLI) registerCommands() {
	c.commands["help"] = Command{
		Name:        "help",
		Description: "Show available commands",
		Handler:     c.cmdHelp,
		Usage:       "help [command]",
	}

	c.commands["send"] = Command{
		Name:        "send",
		Description: "Send a message to the agent",
		Handler:     c.cmdSend,
		Usage:       "send <message>",
	}

	c.commands["config"] = Command{
		Name:        "config",
		Description: "Show current configuration",
		Handler:     c.cmdConfig,
		Usage:       "config",
	}

	c.commands["exit"] = Command{
		Name:        "exit",
		Description: "Exit the CLI",
		Handler:     c.cmdExit,
		Usage:       "exit",
	}

	c.commands["quit"] = Command{
		Name:        "quit",
		Description: "Exit the CLI",
		Handler:     c.cmdExit,
		Usage:       "quit",
	}
}

func (c *CLI) Start() error {
	fmt.Println("MiniClaw CLI")
	fmt.Println("Type 'help' for available commands")
	fmt.Println()

	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("CLI stopped")
			return nil
		default:
			fmt.Print(prompt)
			if !c.scanner.Scan() {
				fmt.Println()
				return nil
			}

			line := strings.TrimSpace(c.scanner.Text())
			if line == "" {
				continue
			}

			args := strings.Fields(line)
			if len(args) == 0 {
				continue
			}

			cmdName := strings.ToLower(args[0])
			cmd, ok := c.commands[cmdName]
			if !ok {
				fmt.Printf("Unknown command: %s\n", cmdName)
				fmt.Println("Type 'help' for available commands")
				continue
			}

			if err := cmd.Handler(args[1:]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

func (c *CLI) Stop() error {
	return nil
}

func (c *CLI) cmdHelp(args []string) error {
	if len(args) > 0 {
		cmdName := strings.ToLower(args[0])
		cmd, ok := c.commands[cmdName]
		if !ok {
			fmt.Printf("Unknown command: %s\n", cmdName)
			return nil
		}
		fmt.Printf("Usage: %s\n", cmd.Usage)
		fmt.Printf("  %s\n", cmd.Description)
		return nil
	}

	fmt.Println("Available commands:")
	for _, cmd := range c.commands {
		fmt.Printf("  %-15s - %s\n", cmd.Name, cmd.Description)
	}
	fmt.Println()
	fmt.Println("Use 'help <command>' for more information about a command")
	return nil
}

func (c *CLI) cmdSend(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: send <message>")
	}

	message := strings.Join(args, " ")

	msg := &bus.Message{
		ID:      fmt.Sprintf("cli-%d", 0),
		Channel: bus.ChannelCLI,
		ChatID:  c.chatID,
		Content: message,
	}

	if err := c.messageBus.Publish(c.ctx, bus.ChannelCLI, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf("Message sent: %s\n", message)
	return nil
}

func (c *CLI) cmdConfig(args []string) error {
	fmt.Println("Current configuration:")
	fmt.Println("  (Configuration display not implemented yet)")
	return nil
}

func (c *CLI) cmdExit(args []string) error {
	fmt.Println("Exiting...")
	return fmt.Errorf("exit")
}
