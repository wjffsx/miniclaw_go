package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type MCPWrappedTool struct {
	name        string
	description string
	schema      map[string]interface{}
	wrapper     *MCPToolWrapper
}

func (t *MCPWrappedTool) Name() string {
	return t.name
}

func (t *MCPWrappedTool) Description() string {
	return t.description
}

func (t *MCPWrappedTool) Parameters() json.RawMessage {
	if t.schema == nil {
		return json.RawMessage("{}")
	}
	schemaBytes, err := json.Marshal(t.schema)
	if err != nil {
		return json.RawMessage("{}")
	}
	return json.RawMessage(schemaBytes)
}

func (t *MCPWrappedTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	result, err := t.wrapper.Execute(ctx, params)
	if err != nil {
		return "", err
	}

	if resultStr, ok := result.(string); ok {
		return resultStr, nil
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}

type AdapterConfig struct {
	ClientName  string
	Prefix      string
	Description string
}

type MCPAdapter struct {
	client   *MCPClient
	config   *AdapterConfig
	registry *tools.ToolRegistry
	mu       sync.RWMutex
}

func NewAdapter(client *MCPClient, config *AdapterConfig, registry *tools.ToolRegistry) (*MCPAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if config == nil {
		config = &AdapterConfig{
			Prefix:      "mcp_",
			Description: "MCP tool",
		}
	}

	if config.Prefix == "" {
		config.Prefix = "mcp_"
	}

	return &MCPAdapter{
		client:   client,
		config:   config,
		registry: registry,
	}, nil
}

func (a *MCPAdapter) RegisterTools(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mcpTools := a.client.GetTools()

	for _, mcpTool := range mcpTools {
		toolName := a.config.Prefix + mcpTool.Name

		description := mcpTool.Description
		if a.config.Description != "" {
			description = fmt.Sprintf("%s: %s", a.config.Description, mcpTool.Description)
		}

		wrappedTool := &MCPToolWrapper{
			client: a.client,
			name:   mcpTool.Name,
			metadata: map[string]interface{}{
				"client_name": a.client.GetConfig().Name,
				"type":        "mcp",
			},
		}

		tool := &MCPWrappedTool{
			name:        toolName,
			description: description,
			schema:      mcpTool.InputSchema,
			wrapper:     wrappedTool,
		}

		if err := a.registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", toolName, err)
		}
	}

	return nil
}

func (a *MCPAdapter) UnregisterTools() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mcpTools := a.client.GetTools()

	for _, mcpTool := range mcpTools {
		toolName := a.config.Prefix + mcpTool.Name

		a.registry.Unregister(toolName)
	}

	return nil
}

func (a *MCPAdapter) RefreshTools(ctx context.Context) error {
	if err := a.UnregisterTools(); err != nil {
		return fmt.Errorf("failed to unregister existing tools: %w", err)
	}

	return a.RegisterTools(ctx)
}

func (a *MCPAdapter) GetClient() *MCPClient {
	return a.client
}

func (a *MCPAdapter) GetConfig() *AdapterConfig {
	return a.config
}

type MCPToolWrapper struct {
	client   *MCPClient
	name     string
	metadata map[string]interface{}
}

func (w *MCPToolWrapper) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	result, err := w.client.ExecuteTool(ctx, w.name, params)
	if err != nil {
		return nil, err
	}

	if result.Error != "" {
		return nil, fmt.Errorf("tool execution failed: %s", result.Error)
	}

	return result.Result, nil
}

func (w *MCPToolWrapper) GetMetadata() map[string]interface{} {
	return w.metadata
}

type MCPManager struct {
	clients  map[string]*MCPClient
	adapters map[string]*MCPAdapter
	registry *tools.ToolRegistry
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewMCPManager(registry *tools.ToolRegistry) *MCPManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &MCPManager{
		clients:  make(map[string]*MCPClient),
		adapters: make(map[string]*MCPAdapter),
		registry: registry,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (m *MCPManager) AddClient(client *MCPClient, adapterConfig *AdapterConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := client.GetConfig().Name

	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("client %s already exists", name)
	}

	adapter, err := NewAdapter(client, adapterConfig, m.registry)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	m.clients[name] = client
	m.adapters[name] = adapter

	return nil
}

func (m *MCPManager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	if err := adapter.UnregisterTools(); err != nil {
		return fmt.Errorf("failed to unregister tools: %w", err)
	}

	client, exists := m.clients[name]
	if exists {
		if err := client.Close(); err != nil {
			return fmt.Errorf("failed to close client: %w", err)
		}
	}

	delete(m.clients, name)
	delete(m.adapters, name)

	return nil
}

func (m *MCPManager) ConnectClient(ctx context.Context, name string) error {
	m.mu.RLock()
	client, exists := m.clients[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if exists {
		if err := adapter.RegisterTools(ctx); err != nil {
			return fmt.Errorf("failed to register tools: %w", err)
		}
	}

	return nil
}

func (m *MCPManager) DisconnectClient(name string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	if err := adapter.UnregisterTools(); err != nil {
		return fmt.Errorf("failed to unregister tools: %w", err)
	}

	m.mu.RLock()
	client, exists := m.clients[name]
	m.mu.RUnlock()

	if exists {
		if err := client.Disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect client: %w", err)
		}
	}

	return nil
}

func (m *MCPManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.ConnectClient(ctx, name); err != nil {
			return fmt.Errorf("failed to connect client %s: %w", name, err)
		}
	}

	return nil
}

func (m *MCPManager) DisconnectAll() error {
	m.mu.RLock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.DisconnectClient(name); err != nil {
			return fmt.Errorf("failed to disconnect client %s: %w", name, err)
		}
	}

	return nil
}

func (m *MCPManager) GetClient(name string) (*MCPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[name]
	return client, exists
}

func (m *MCPManager) GetAdapter(name string) (*MCPAdapter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[name]
	return adapter, exists
}

func (m *MCPManager) ListClients() []*ClientStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]*ClientStatus, 0, len(m.clients))
	for _, client := range m.clients {
		statuses = append(statuses, client.GetStatus())
	}

	return statuses
}

func (m *MCPManager) Close() error {
	m.cancel()
	return m.DisconnectAll()
}

func (m *MCPManager) GetContext() context.Context {
	return m.ctx
}
