package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type ClientConfig struct {
	Name       string
	Type       string
	Endpoint   string
	Transport  string
	Headers    map[string]string
	Timeout    int
	MaxRetries int
	RetryDelay int
}

type MCPClient struct {
	config      *ClientConfig
	protocol    Protocol
	connected   bool
	mu          sync.RWMutex
	tools       map[string]*MCPTool
	initialized bool
	ctx         context.Context
	cancel      context.CancelFunc
}

type MCPTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type ClientState string

const (
	StateDisconnected ClientState = "disconnected"
	StateConnecting   ClientState = "connecting"
	StateConnected    ClientState = "connected"
	StateError        ClientState = "error"
)

type ClientStatus struct {
	State     ClientState
	Connected bool
	ToolCount int
	Error     string
}

func NewClient(config *ClientConfig) (*MCPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("client config cannot be nil")
	}

	if config.Name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &MCPClient{
		config: config,
		tools:  make(map[string]*MCPTool),
		ctx:    ctx,
		cancel: cancel,
	}

	return client, nil
}

func (c *MCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("client already connected")
	}

	protocol, err := NewProtocol(c.config)
	if err != nil {
		return fmt.Errorf("failed to create protocol: %w", err)
	}

	c.protocol = protocol

	if err := c.protocol.Connect(ctx); err != nil {
		c.connected = false
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.connected = true

	if err := c.initializeTools(ctx); err != nil {
		c.protocol.Close()
		c.connected = false
		return fmt.Errorf("failed to initialize tools: %w", err)
	}

	c.initialized = true

	return nil
}

func (c *MCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.protocol != nil {
		if err := c.protocol.Close(); err != nil {
			return err
		}
	}

	c.connected = false
	c.initialized = false
	c.tools = make(map[string]*MCPTool)

	return nil
}

func (c *MCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *MCPClient) GetStatus() *ClientStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := StateDisconnected
	if c.connected {
		state = StateConnected
	}

	return &ClientStatus{
		State:     state,
		Connected: c.connected,
		ToolCount: len(c.tools),
	}
}

func (c *MCPClient) GetTools() []*MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tools := make([]*MCPTool, 0, len(c.tools))
	for _, tool := range c.tools {
		tools = append(tools, tool)
	}

	return tools
}

func (c *MCPClient) GetTool(name string) (*MCPTool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tool, exists := c.tools[name]
	return tool, exists
}

func (c *MCPClient) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolCall, error) {
	c.mu.RLock()
	if !c.connected || !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not connected or initialized")
	}

	_, exists := c.tools[name]
	if !exists {
		c.mu.RUnlock()
		return nil, fmt.Errorf("tool %s not found", name)
	}
	c.mu.RUnlock()

	result, err := c.protocol.CallTool(ctx, name, params)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	return result, nil
}

func (c *MCPClient) initializeTools(ctx context.Context) error {
	toolsList, err := c.protocol.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	c.tools = make(map[string]*MCPTool)

	for _, tool := range toolsList {
		c.tools[tool.Name] = tool
	}

	return nil
}

func (c *MCPClient) Close() error {
	c.cancel()
	return c.Disconnect()
}

func (c *MCPClient) GetConfig() *ClientConfig {
	return c.config
}

func (c *MCPClient) ListResources(ctx context.Context) ([]map[string]interface{}, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	return c.protocol.ListResources(ctx)
}

func (c *MCPClient) ReadResource(ctx context.Context, uri string) (string, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	return c.protocol.ReadResource(ctx, uri)
}

func (c *MCPClient) ListPrompts(ctx context.Context) ([]map[string]interface{}, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	return c.protocol.ListPrompts(ctx)
}

func (c *MCPClient) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	return c.protocol.GetPrompt(ctx, name, args)
}

func (c *MCPClient) SendNotification(ctx context.Context, method string, params map[string]interface{}) error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	return c.protocol.SendNotification(ctx, method, params)
}

func (c *MCPClient) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := map[string]interface{}{
		"name":       c.config.Name,
		"type":       c.config.Type,
		"endpoint":   c.config.Endpoint,
		"connected":  c.connected,
		"tool_count": len(c.tools),
	}

	return json.Marshal(status)
}
