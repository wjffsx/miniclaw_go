package cli

import (
	"context"
	"fmt"
	"testing"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	if cli == nil {
		t.Error("Expected CLI to be created")
	}

	if cli.commands == nil {
		t.Error("Expected commands map to be initialized")
	}

	if len(cli.commands) == 0 {
		t.Error("Expected default commands to be registered")
	}
}

func TestRegisterCommand(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	initialCount := len(cli.commands)

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	if len(cli.commands) != initialCount+1 {
		t.Errorf("Expected %d commands, got %d", initialCount+1, len(cli.commands))
	}

	if _, exists := cli.commands["test"]; !exists {
		t.Error("Expected command to be registered")
	}
}

func TestGetCommand(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	cmd, exists := cli.GetCommand("test")
	if !exists {
		t.Error("Expected command to exist")
	}

	if cmd.Name != "test" {
		t.Errorf("Expected command name 'test', got '%s'", cmd.Name)
	}
}

func TestGetCommandNonExistent(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	_, exists := cli.GetCommand("nonexistent")
	if exists {
		t.Error("Expected command to not exist")
	}
}

func TestListCommands(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	commands := cli.ListCommands()
	if len(commands) == 0 {
		t.Error("Expected at least one command")
	}
}

func TestExecuteCommand(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	err := cli.ExecuteCommand("test", []string{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExecuteCommandNonExistent(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	err := cli.ExecuteCommand("nonexistent", []string{})
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
}

func TestExecuteCommandWithArgs(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	var receivedArgs []string
	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { receivedArgs = args; return nil },
		Usage:       "test [args]",
	})

	args := []string{"arg1", "arg2"}
	err := cli.ExecuteCommand("test", args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(receivedArgs) != 2 {
		t.Errorf("Expected 2 args, got %d", len(receivedArgs))
	}
}

func TestParseInput(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	input := "test arg1 arg2"
	cmd, args := cli.ParseInput(input)

	if cmd != "test" {
		t.Errorf("Expected command 'test', got '%s'", cmd)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != "arg1" {
		t.Errorf("Expected arg 'arg1', got '%s'", args[0])
	}
}

func TestParseInputEmpty(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cmd, args := cli.ParseInput("")

	if cmd != "" {
		t.Errorf("Expected empty command, got '%s'", cmd)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestParseInputWhitespace(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	input := "   test   arg1   arg2   "
	cmd, args := cli.ParseInput(input)

	if cmd != "test" {
		t.Errorf("Expected command 'test', got '%s'", cmd)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestCmdHelp(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	err := cli.cmdHelp([]string{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCmdHelpSpecific(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	err := cli.cmdHelp([]string{"test"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestCmdHelpNonExistent(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	err := cli.cmdHelp([]string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
}

func TestCmdExit(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	err := cli.cmdExit([]string{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetChatID(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	chatID := cli.GetChatID()
	if chatID != "cli" {
		t.Errorf("Expected chat ID 'cli', got '%s'", chatID)
	}
}

func TestSetChatID(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.SetChatID("test-chat-id")
	chatID := cli.GetChatID()

	if chatID != "test-chat-id" {
		t.Errorf("Expected chat ID 'test-chat-id', got '%s'", chatID)
	}
}

func TestHandleInput(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	err := cli.HandleInput("test arg1 arg2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHandleInputEmpty(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	err := cli.HandleInput("")
	if err != nil {
		t.Errorf("Expected no error for empty input, got %v", err)
	}
}

func TestHandleInputNonExistent(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	err := cli.HandleInput("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
}

func TestHandleInputWithPrompt(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	input := Prompt + "test arg1 arg2"
	err := cli.HandleInput(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHandleInputWithPromptAndWhitespace(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return nil },
		Usage:       "test [args]",
	})

	input := Prompt + "   test   arg1   arg2   "
	err := cli.HandleInput(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHandleInputError(t *testing.T) {
	cli := NewCLI(nil, context.Background())

	cli.RegisterCommand("test", Command{
		Name:        "test",
		Description: "Test command",
		Handler:     func(args []string) error { return fmt.Errorf("error") },
		Usage:       "test [args]",
	})

	err := cli.HandleInput("test")
	if err == nil {
		t.Error("Expected error")
	}
}
