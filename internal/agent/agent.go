package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wjffsx/miniclaw_go/internal/bus"
	agentcontext "github.com/wjffsx/miniclaw_go/internal/context"
	"github.com/wjffsx/miniclaw_go/internal/llm"
	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type Agent struct {
	messageBus     bus.MessageBus
	llmManager     *llm.MultiModelManager
	toolExecutor   *tools.ToolExecutor
	contextBuilder *agentcontext.Builder
	sessionStorage storage.SessionStorage
	memoryStorage  storage.MemoryStorage
	ctx            context.Context
	chatHistory    map[string][]llm.Message
	maxIterations  int
}

type Config struct {
	LLMModels      []*llm.ModelConfig
	DefaultModel   string
	SessionStorage storage.SessionStorage
	MemoryStorage  storage.MemoryStorage
	Storage        storage.Storage
	ToolRegistry   *tools.ToolRegistry
	MaxIterations  int
}

func NewAgent(config *Config, messageBus bus.MessageBus, ctx context.Context) (*Agent, error) {
	llmManager, err := llm.NewMultiModelManager(config.LLMModels, config.DefaultModel)
	if err != nil {
		log.Printf("Warning: failed to create LLM manager: %v", err)
		log.Println("Agent will run without LLM support")
		llmManager = nil
	}

	toolExecutor := tools.NewToolExecutor(config.ToolRegistry)

	contextBuilder := agentcontext.NewBuilder(&agentcontext.Config{
		Storage:       config.Storage,
		MemoryStorage: config.MemoryStorage,
	})

	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	return &Agent{
		messageBus:     messageBus,
		llmManager:     llmManager,
		toolExecutor:   toolExecutor,
		contextBuilder: contextBuilder,
		sessionStorage: config.SessionStorage,
		memoryStorage:  config.MemoryStorage,
		ctx:            ctx,
		chatHistory:    make(map[string][]llm.Message),
		maxIterations:  maxIterations,
	}, nil
}

func (a *Agent) Start() error {
	if a.llmManager != nil {
		log.Printf("Starting agent with LLM provider: %s, model: %s", a.llmManager.GetProvider(), a.llmManager.GetModel())
	} else {
		log.Println("Starting agent without LLM support")
	}

	if _, err := a.messageBus.Subscribe(bus.ChannelCLI, a.HandleMessage); err != nil {
		return fmt.Errorf("failed to subscribe to CLI channel: %w", err)
	}

	if _, err := a.messageBus.Subscribe(bus.ChannelTelegram, a.HandleMessage); err != nil {
		return fmt.Errorf("failed to subscribe to Telegram channel: %w", err)
	}

	if _, err := a.messageBus.Subscribe(bus.ChannelWebSocket, a.HandleMessage); err != nil {
		return fmt.Errorf("failed to subscribe to WebSocket channel: %w", err)
	}

	return nil
}

func (a *Agent) Stop() error {
	log.Println("Stopping agent...")
	return nil
}

func (a *Agent) HandleMessage(ctx context.Context, msg *bus.Message) error {
	log.Printf("Agent received message from %s: %s", msg.Channel, msg.Content)

	if a.llmManager == nil {
		responseMsg := &bus.Message{
			ID:      fmt.Sprintf("agent-%s", msg.ID),
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: "LLM is not configured. Please set up your API key in the configuration.",
		}
		return a.messageBus.Publish(ctx, msg.Channel, responseMsg)
	}

	messages := a.getChatHistory(msg.ChatID)

	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: msg.Content,
	})

	response, err := a.runReActLoop(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to run ReAct loop: %w", err)
	}

	log.Printf("Final LLM response: %s", response)

	messages = append(messages, llm.Message{
		Role:    llm.RoleAssistant,
		Content: response,
	})

	a.setChatHistory(msg.ChatID, messages)

	responseMsg := &bus.Message{
		ID:      fmt.Sprintf("agent-%s", msg.ID),
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: response,
	}

	if err := a.messageBus.Publish(ctx, msg.Channel, responseMsg); err != nil {
		return fmt.Errorf("failed to publish response: %w", err)
	}

	return nil
}

func (a *Agent) runReActLoop(ctx context.Context, messages []llm.Message) (string, error) {
	toolSchemas := a.toolExecutor.GetSchemas()

	agentContext, err := a.contextBuilder.Build(ctx, toolSchemas)
	if err != nil {
		log.Printf("Failed to build context: %v", err)
	}

	systemPrompt := agentContext.BuildSystemPrompt(toolSchemas)

	for iteration := 0; iteration < a.maxIterations; iteration++ {
		log.Printf("ReAct iteration %d/%d", iteration+1, a.maxIterations)

		llmMessages := make([]llm.Message, 0, len(messages)+1)
		llmMessages = append(llmMessages, llm.Message{
			Role:    llm.RoleSystem,
			Content: systemPrompt,
		})
		llmMessages = append(llmMessages, messages...)

		response, err := a.llmManager.Complete(ctx, llmMessages)
		if err != nil {
			return "", fmt.Errorf("failed to complete LLM request: %w", err)
		}

		log.Printf("LLM response: %s", response.Content)

		toolCalls, isFinal := a.parseResponse(response.Content)
		if isFinal {
			return response.Content, nil
		}

		if len(toolCalls) == 0 {
			return response.Content, nil
		}

		toolResults := make([]tools.ToolCall, 0, len(toolCalls))
		for _, call := range toolCalls {
			log.Printf("Executing tool: %s with params: %v", call.Name, call.Input)

			result, err := a.toolExecutor.Execute(ctx, call.Name, call.Input)
			if err != nil {
				log.Printf("Tool execution error: %v", err)
				result.Error = err.Error()
			}

			toolResults = append(toolResults, *result)
			log.Printf("Tool result: %s", result.Result)
		}

		toolResultsJSON, err := json.MarshalIndent(toolResults, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal tool results: %w", err)
		}

		observation := fmt.Sprintf("Tool execution results:\n%s", string(toolResultsJSON))
		messages = append(messages, llm.Message{
			Role:    llm.RoleAssistant,
			Content: response.Content,
		})
		messages = append(messages, llm.Message{
			Role:    llm.RoleUser,
			Content: observation,
		})
	}

	return "", fmt.Errorf("max iterations (%d) reached without final answer", a.maxIterations)
}

func (a *Agent) parseResponse(content string) ([]tools.ToolCall, bool) {
	var response struct {
		Thought     string           `json:"thought"`
		ToolCalls   []tools.ToolCall `json:"tool_calls"`
		FinalAnswer string           `json:"final_answer"`
	}

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		log.Printf("Failed to parse LLM response as JSON: %v", err)
		return nil, true
	}

	if response.FinalAnswer != "" {
		return nil, true
	}

	if len(response.ToolCalls) > 0 {
		return response.ToolCalls, false
	}

	return nil, true
}

func (a *Agent) getChatHistory(chatID string) []llm.Message {
	if history, ok := a.chatHistory[chatID]; ok {
		return history
	}

	messages, err := a.sessionStorage.GetMessages(context.Background(), chatID, 50)
	if err != nil {
		log.Printf("Failed to load messages for %s: %v", chatID, err)
		return []llm.Message{}
	}

	llmMessages := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		llmMessages = append(llmMessages, llm.Message{
			Role:    llm.MessageRole(msg.Role),
			Content: msg.Content,
		})
	}

	a.chatHistory[chatID] = llmMessages
	return llmMessages
}

func (a *Agent) setChatHistory(chatID string, messages []llm.Message) {
	a.chatHistory[chatID] = messages

	for _, msg := range messages {
		if err := a.sessionStorage.SaveMessage(context.Background(), chatID, string(msg.Role), msg.Content); err != nil {
			log.Printf("Failed to save message for %s: %v", chatID, err)
		}
	}
}
