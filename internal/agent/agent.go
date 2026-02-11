package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/wjffsx/miniclaw_go/internal/bus"
	agentcontext "github.com/wjffsx/miniclaw_go/internal/context"
	"github.com/wjffsx/miniclaw_go/internal/llm"
	"github.com/wjffsx/miniclaw_go/internal/mcp"
	"github.com/wjffsx/miniclaw_go/internal/scheduler"
	"github.com/wjffsx/miniclaw_go/internal/skills"
	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type Agent struct {
	messageBus     bus.MessageBus
	llmManager     *llm.MultiModelManager
	toolExecutor   *tools.ToolExecutor
	contextBuilder *agentcontext.Builder
	skillSelector  *skills.SkillSelector
	mcpManager     *mcp.MCPManager
	taskManager    *scheduler.TaskManager
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
	SkillRegistry  *skills.SkillRegistry
	SkillConfig    *skills.SkillConfig
	MCPManager     *mcp.MCPManager
	TaskManager    *scheduler.TaskManager
	MaxIterations  int
}

func NewAgent(config *Config, messageBus bus.MessageBus, ctx context.Context) (*Agent, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

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

	var skillSelector *skills.SkillSelector
	if config.SkillRegistry != nil {
		selectionConfig := &skills.SelectionConfig{
			Method:    "hybrid",
			Threshold: 0.5,
			MaxActive: 5,
		}
		if config.SkillConfig != nil {
			selectionConfig = &config.SkillConfig.Selection
		}
		skillSelector = skills.NewSkillSelector(config.SkillRegistry, nil, selectionConfig)
		log.Printf("Skill selector initialized with method: %s", selectionConfig.Method)
	}

	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	return &Agent{
		messageBus:     messageBus,
		llmManager:     llmManager,
		toolExecutor:   toolExecutor,
		contextBuilder: contextBuilder,
		skillSelector:  skillSelector,
		mcpManager:     config.MCPManager,
		taskManager:    config.TaskManager,
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
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

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

	response, err := a.runReActLoop(ctx, messages, msg.Content)
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

func (a *Agent) runReActLoop(ctx context.Context, messages []llm.Message, userMessage string) (string, error) {
	toolSchemas := a.toolExecutor.GetSchemas()

	agentContext, err := a.contextBuilder.Build(ctx, toolSchemas)
	if err != nil {
		log.Printf("Failed to build context: %v", err)
	}

	systemPrompt := agentContext.BuildSystemPrompt(toolSchemas)

	if a.skillSelector != nil {
		selectedSkills, err := a.skillSelector.Select(ctx, userMessage)
		if err != nil {
			log.Printf("Failed to select skills: %v", err)
		} else if len(selectedSkills) > 0 {
			log.Printf("Selected %d skills: %v", len(selectedSkills), getSkillNames(selectedSkills))
			skillContext := a.buildSkillContext(selectedSkills)
			systemPrompt += "\n\n" + skillContext
		}
	}

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

func (a *Agent) buildSkillContext(selectedSkills []*skills.Skill) string {
	var builder strings.Builder

	builder.WriteString("## Active Skills\n\n")
	builder.WriteString("The following skills have been activated for this conversation:\n\n")

	for _, skill := range selectedSkills {
		builder.WriteString(fmt.Sprintf("### %s\n", skill.Name))
		builder.WriteString(fmt.Sprintf("**Description**: %s\n", skill.Description))
		if skill.Category != "" {
			builder.WriteString(fmt.Sprintf("**Category**: %s\n", skill.Category))
		}
		if len(skill.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("**Tags**: %v\n", skill.Tags))
		}
		builder.WriteString(fmt.Sprintf("**Instructions**:\n%s\n\n", skill.Content))
	}

	builder.WriteString("Use these skills as guidelines when responding to the user. Adapt your approach based on the specific requirements of each skill.\n")

	return builder.String()
}

func getSkillNames(skills []*skills.Skill) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	return names
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

func (a *Agent) GetChatHistory(chatID string) []llm.Message {
	return a.getChatHistory(chatID)
}

func (a *Agent) ClearChatHistory(chatID string) {
	a.chatHistory[chatID] = []llm.Message{}
}

func (a *Agent) SetMaxIterations(maxIterations int) {
	a.maxIterations = maxIterations
}

func (a *Agent) GetMaxIterations() int {
	return a.maxIterations
}

func (a *Agent) GetToolExecutor() *tools.ToolExecutor {
	return a.toolExecutor
}

func (a *Agent) GetSkillSelector() *skills.SkillSelector {
	return a.skillSelector
}

func (a *Agent) GetMCPManager() *mcp.MCPManager {
	return a.mcpManager
}

func (a *Agent) GetTaskManager() *scheduler.TaskManager {
	return a.taskManager
}

func (a *Agent) setChatHistory(chatID string, messages []llm.Message) {
	a.chatHistory[chatID] = messages

	for _, msg := range messages {
		if err := a.sessionStorage.SaveMessage(context.Background(), chatID, string(msg.Role), msg.Content); err != nil {
			log.Printf("Failed to save message for %s: %v", chatID, err)
		}
	}
}
