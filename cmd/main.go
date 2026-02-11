package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/agent"
	"github.com/wjffsx/miniclaw_go/internal/bus"
	"github.com/wjffsx/miniclaw_go/internal/communication/telegram"
	"github.com/wjffsx/miniclaw_go/internal/communication/websocket"
	"github.com/wjffsx/miniclaw_go/internal/config"
	"github.com/wjffsx/miniclaw_go/internal/filetools"
	"github.com/wjffsx/miniclaw_go/internal/llm"
	"github.com/wjffsx/miniclaw_go/internal/mcp"
	"github.com/wjffsx/miniclaw_go/internal/memory"
	"github.com/wjffsx/miniclaw_go/internal/scheduler"
	"github.com/wjffsx/miniclaw_go/internal/search"
	"github.com/wjffsx/miniclaw_go/internal/skills"
	"github.com/wjffsx/miniclaw_go/internal/storage"
	"github.com/wjffsx/miniclaw_go/internal/tools"
)

const (
	version = "0.1.0"
)

var (
	telegramBot     *telegram.Bot
	websocketServer *websocket.Server
	agentService    *agent.Agent
	skillWatcher    *skills.SkillFileWatcher
	mcpManager      *mcp.MCPManager
	taskManager     *scheduler.TaskManager
)

func main() {
	log.Printf("MiniClaw Go v%s starting...", version)
	log.Println("========================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configMgr, err := config.NewFileConfigManager("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	cfg := configMgr.GetConfig()
	log.Printf("Configuration loaded successfully")
	log.Printf("Telegram: %v", cfg.Telegram.Enabled)
	log.Printf("WebSocket: %v", cfg.WebSocket.Enabled)
	log.Printf("LLM Provider: %s", cfg.LLM.Provider)

	messageBus := bus.NewInMemoryMessageBus(ctx)
	messageBus.Start()
	defer messageBus.Close()
	log.Println("Message bus started")

	sessionStorage := storage.NewFileSystemSessionStorage(cfg.Storage.BasePath + "/sessions")
	memoryStorage := storage.NewFileSystemMemoryStorage(cfg.Storage.BasePath + "/memory")
	fileStorage := storage.NewFileStorage(cfg.Storage.BasePath)

	log.Printf("Storage initialized at: %s", cfg.Storage.BasePath)

	if err := initializeCommunication(ctx, messageBus, cfg); err != nil {
		log.Fatalf("Failed to initialize communication: %v", err)
	}

	if err := initializeAgent(ctx, messageBus, cfg, sessionStorage, memoryStorage, fileStorage); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	log.Println("========================================")
	log.Println("MiniClaw Go is ready!")
	log.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := gracefulShutdown(shutdownCtx, messageBus); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("MiniClaw Go stopped gracefully")
}

func initializeCommunication(ctx context.Context, messageBus bus.MessageBus, cfg *config.Config) error {
	if cfg.Telegram.Enabled {
		log.Println("Initializing Telegram bot...")

		tgCfg := &telegram.Config{
			Token: cfg.Telegram.Token,
		}

		telegramBot = telegram.NewBot(tgCfg, messageBus, ctx)

		handler := telegram.NewHandler(telegramBot)

		if _, err := messageBus.Subscribe(bus.ChannelTelegram, handler.HandleMessage); err != nil {
			log.Printf("Failed to subscribe Telegram handler: %v", err)
		}

		if err := telegramBot.Start(); err != nil {
			log.Printf("Failed to start Telegram bot: %v", err)
		}
	}

	if cfg.WebSocket.Enabled {
		log.Printf("Initializing WebSocket server on %s:%d...", cfg.WebSocket.Host, cfg.WebSocket.Port)

		wsCfg := &websocket.Config{
			Port:       cfg.WebSocket.Port,
			MaxClients: 10,
		}

		websocketServer = websocket.NewServer(wsCfg, messageBus, ctx)

		handler := websocket.NewHandler(websocketServer)

		if _, err := messageBus.Subscribe(bus.ChannelWebSocket, handler.HandleMessage); err != nil {
			log.Printf("Failed to subscribe WebSocket handler: %v", err)
		}

		if err := websocketServer.Start(cfg.WebSocket.Port); err != nil {
			log.Printf("Failed to start WebSocket server: %v", err)
		}
	}

	return nil
}

func initializeAgent(ctx context.Context, messageBus bus.MessageBus, cfg *config.Config, sessionStorage storage.SessionStorage, memoryStorage storage.MemoryStorage, fileStorage storage.Storage) error {
	log.Println("Initializing agent service...")

	toolRegistry := tools.NewToolRegistry()

	getTimeTool := tools.NewGetTimeTool()
	if err := toolRegistry.Register(getTimeTool); err != nil {
		log.Printf("Failed to register get_time tool: %v", err)
	}

	echoTool := tools.NewEchoTool()
	if err := toolRegistry.Register(echoTool); err != nil {
		log.Printf("Failed to register echo tool: %v", err)
	}

	calculateTool := tools.NewCalculateTool()
	if err := toolRegistry.Register(calculateTool); err != nil {
		log.Printf("Failed to register calculate tool: %v", err)
	}

	memoryManager := memory.NewManager(memoryStorage)
	memoryTools := memory.NewMemoryTools(memoryManager)
	for _, memTool := range memoryTools {
		if err := toolRegistry.Register(memTool); err != nil {
			log.Printf("Failed to register %s tool: %v", memTool.Name(), err)
		}
	}

	fileTools := filetools.NewFileTools(fileStorage)
	for _, fileTool := range fileTools {
		if err := toolRegistry.Register(fileTool); err != nil {
			log.Printf("Failed to register %s tool: %v", fileTool.Name(), err)
		}
	}

	if cfg.Search.BraveAPIKey != "" {
		searchConfig := &search.SearchConfig{
			APIKey: cfg.Search.BraveAPIKey,
		}
		searchClient := search.NewBraveSearchClient(searchConfig)
		webSearchTool := search.NewWebSearchTool(searchClient)
		if err := toolRegistry.Register(webSearchTool); err != nil {
			log.Printf("Failed to register web_search tool: %v", err)
		}
	}

	log.Printf("Registered %d tools", len(toolRegistry.List()))

	var skillRegistry *skills.SkillRegistry
	var skillConfig *skills.SkillConfig

	if cfg.Skills.Enabled {
		log.Println("Initializing skills system...")
		skillRegistry = skills.NewSkillRegistry(fileStorage)

		if err := skillRegistry.LoadFromDirectory(ctx, cfg.Skills.Directory); err != nil {
			log.Printf("Failed to load skills from directory: %v", err)
		} else {
			log.Printf("Loaded %d skills", skillRegistry.Count())
		}

		if cfg.Skills.AutoReload {
			watcher, err := skills.NewSkillFileWatcher(skillRegistry, skills.NewSkillParser(fileStorage))
			if err != nil {
				log.Printf("Failed to create skill file watcher: %v", err)
			} else {
				skillWatcher = watcher
				if err := skillWatcher.WatchDirectory(cfg.Skills.Directory); err != nil {
					log.Printf("Failed to watch skills directory: %v", err)
				}
				log.Println("Skill file watcher started")
			}
		}

		skillConfig = &skills.SkillConfig{
			Directory:  cfg.Skills.Directory,
			AutoReload: cfg.Skills.AutoReload,
			MaxActive:  cfg.Skills.MaxActive,
			Selection: skills.SelectionConfig{
				Method:    cfg.Skills.Selection.Method,
				Threshold: cfg.Skills.Selection.Threshold,
			},
		}
	}

	var mcpManager *mcp.MCPManager
	if cfg.MCP.Enabled {
		log.Println("Initializing MCP manager...")
		mcpManager = mcp.NewMCPManager(toolRegistry)

		for _, clientConfig := range cfg.MCP.Clients {
			mcpClientConfig := &mcp.ClientConfig{
				Name:      clientConfig.Name,
				Type:      clientConfig.Type,
				Endpoint:  clientConfig.Endpoint,
				Transport: clientConfig.Transport,
				Headers:   clientConfig.Headers,
				Timeout:   clientConfig.Timeout,
			}

			mcpClient, err := mcp.NewClient(mcpClientConfig)
			if err != nil {
				log.Printf("Failed to create MCP client %s: %v", clientConfig.Name, err)
				continue
			}

			adapterConfig := &mcp.AdapterConfig{
				ClientName:  clientConfig.Name,
				Prefix:      "mcp_" + clientConfig.Name + "_",
				Description: "MCP tool from " + clientConfig.Name,
			}

			if err := mcpManager.AddClient(mcpClient, adapterConfig); err != nil {
				log.Printf("Failed to add MCP client %s: %v", clientConfig.Name, err)
				continue
			}

			log.Printf("Added MCP client: %s", clientConfig.Name)
		}

		if err := mcpManager.ConnectAll(ctx); err != nil {
			log.Printf("Failed to connect MCP clients: %v", err)
		} else {
			log.Printf("MCP manager initialized with %d clients", len(cfg.MCP.Clients))
		}
	}

	var taskManager *scheduler.TaskManager
	if cfg.Scheduler.Enabled {
		log.Println("Initializing task scheduler...")
		sched := scheduler.NewScheduler(&scheduler.SchedulerConfig{
			TickInterval: time.Duration(cfg.Scheduler.TickInterval) * time.Second,
		})

		taskManager = scheduler.NewTaskManager(sched, &scheduler.TaskManagerConfig{
			TasksFile: cfg.Scheduler.TasksFile,
		})

		if cfg.Scheduler.AutoStart {
			if err := sched.Start(); err != nil {
				log.Printf("Failed to start scheduler: %v", err)
			} else {
				log.Println("Task scheduler started")
			}

			if err := taskManager.Start(); err != nil {
				log.Printf("Failed to start task manager: %v", err)
			}
		}
	}

	llmModels := make([]*llm.ModelConfig, 0)

	if len(cfg.LLM.Models) > 0 {
		for _, modelConfig := range cfg.LLM.Models {
			llmModels = append(llmModels, &llm.ModelConfig{
				Name:        modelConfig.Name,
				Provider:    modelConfig.Provider,
				APIKey:      modelConfig.APIKey,
				Model:       modelConfig.Model,
				MaxTokens:   modelConfig.MaxTokens,
				Temperature: modelConfig.Temperature,
				LocalModel: llm.LocalModelConfig{
					Enabled: modelConfig.LocalModel.Enabled,
					Path:    modelConfig.LocalModel.Path,
					Type:    modelConfig.LocalModel.Type,
				},
			})
		}
	} else {
		llmModels = append(llmModels, &llm.ModelConfig{
			Name:        "default",
			Provider:    cfg.LLM.Provider,
			APIKey:      cfg.LLM.APIKey,
			Model:       cfg.LLM.Model,
			MaxTokens:   cfg.LLM.MaxTokens,
			Temperature: cfg.LLM.Temperature,
			LocalModel: llm.LocalModelConfig{
				Enabled: cfg.LLM.LocalModel.Enabled,
				Path:    cfg.LLM.LocalModel.Path,
				Type:    cfg.LLM.LocalModel.Type,
			},
		})
	}

	defaultModel := cfg.LLM.DefaultModel
	if defaultModel == "" {
		defaultModel = "default"
	}

	agentConfig := &agent.Config{
		LLMModels:      llmModels,
		DefaultModel:   defaultModel,
		SessionStorage: sessionStorage,
		MemoryStorage:  memoryStorage,
		Storage:        fileStorage,
		ToolRegistry:   toolRegistry,
		SkillRegistry:  skillRegistry,
		SkillConfig:    skillConfig,
		MCPManager:     mcpManager,
		TaskManager:    taskManager,
	}

	var err error
	agentService, err = agent.NewAgent(agentConfig, messageBus, ctx)
	if err != nil {
		return err
	}

	if err := agentService.Start(); err != nil {
		return err
	}

	return nil
}

func gracefulShutdown(ctx context.Context, messageBus bus.MessageBus) error {
	log.Println("Performing graceful shutdown...")

	if telegramBot != nil {
		if err := telegramBot.Stop(); err != nil {
			log.Printf("Error stopping Telegram bot: %v", err)
		}
	}

	if websocketServer != nil {
		if err := websocketServer.Stop(); err != nil {
			log.Printf("Error stopping WebSocket server: %v", err)
		}
	}

	if skillWatcher != nil {
		skillWatcher.Stop()
	}

	if mcpManager != nil {
		if err := mcpManager.Close(); err != nil {
			log.Printf("Error closing MCP manager: %v", err)
		}
	}

	if taskManager != nil {
		if err := taskManager.Stop(); err != nil {
			log.Printf("Error stopping task manager: %v", err)
		}
	}

	if agentService != nil {
		if err := agentService.Stop(); err != nil {
			log.Printf("Error stopping agent: %v", err)
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
