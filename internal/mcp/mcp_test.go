package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestNewHTTPTransport(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}

	transport := NewHTTPTransport("http://example.com", headers, 30)

	if transport == nil {
		t.Error("Expected transport to be created")
	}

	if transport.endpoint != "http://example.com" {
		t.Errorf("Expected endpoint 'http://example.com', got '%s'", transport.endpoint)
	}

	if transport.headers["Authorization"] != "Bearer test-token" {
		t.Error("Expected Authorization header")
	}

	if transport.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", transport.timeout)
	}
}

func TestNewProtocol(t *testing.T) {
	config := &ClientConfig{
		Name:      "test",
		Endpoint:  "http://example.com",
		Transport: "http",
	}

	protocol, err := NewProtocol(config)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if protocol == nil {
		t.Error("Expected protocol to be created")
	}
}

func TestNewProtocolNilConfig(t *testing.T) {
	_, err := NewProtocol(nil)

	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestNewProtocolEmptyEndpoint(t *testing.T) {
	config := &ClientConfig{
		Name: "test",
	}

	_, err := NewProtocol(config)

	if err == nil {
		t.Error("Expected error for empty endpoint")
	}
}

func TestNewClient(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, err := NewClient(config)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Error("Expected client to be created")
	}

	if client.config.Name != "test-client" {
		t.Errorf("Expected name 'test-client', got '%s'", client.config.Name)
	}
}

func TestNewClientNilConfig(t *testing.T) {
	_, err := NewClient(nil)

	if err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestNewClientEmptyName(t *testing.T) {
	config := &ClientConfig{
		Endpoint: "http://example.com",
	}

	_, err := NewClient(config)

	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestClientIsConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	if client.IsConnected() {
		t.Error("Expected client to not be connected initially")
	}
}

func TestClientGetStatus(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	status := client.GetStatus()

	if status == nil {
		t.Error("Expected status to be returned")
	}

	if status.Connected {
		t.Error("Expected client to not be connected")
	}

	if status.State != StateDisconnected {
		t.Errorf("Expected state %s, got %s", StateDisconnected, status.State)
	}
}

func TestClientGetConfig(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	retrievedConfig := client.GetConfig()

	if retrievedConfig == nil {
		t.Error("Expected config to be returned")
	}

	if retrievedConfig.Name != "test-client" {
		t.Errorf("Expected name 'test-client', got '%s'", retrievedConfig.Name)
	}
}

func TestNewMCPManager(t *testing.T) {
	registry := tools.NewToolRegistry()

	manager := NewMCPManager(registry)

	if manager == nil {
		t.Error("Expected manager to be created")
	}

	if manager.registry != registry {
		t.Error("Expected registry to be set")
	}
}

func TestMCPManagerAddClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	err := manager.AddClient(client, adapterConfig)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	retrievedClient, exists := manager.GetClient("test-client")

	if !exists {
		t.Error("Expected client to exist")
	}

	if retrievedClient == nil {
		t.Error("Expected client to be returned")
	}
}

func TestMCPManagerAddDuplicateClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	err := manager.AddClient(client, adapterConfig)

	if err == nil {
		t.Error("Expected error for duplicate client")
	}
}

func TestMCPManagerListClients(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	statuses := manager.ListClients()

	if len(statuses) != 1 {
		t.Errorf("Expected 1 client, got %d", len(statuses))
	}

	if statuses[0].State != StateDisconnected {
		t.Errorf("Expected state %s, got %s", StateDisconnected, statuses[0].State)
	}
}

func TestMCPManagerRemoveClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	err := manager.RemoveClient("test-client")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, exists := manager.GetClient("test-client")

	if exists {
		t.Error("Expected client to be removed")
	}
}

func TestMCPManagerRemoveNonExistentClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	err := manager.RemoveClient("non-existent")

	if err == nil {
		t.Error("Expected error for non-existent client")
	}
}

func TestMCPManagerGetContext(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	ctx := manager.GetContext()

	if ctx == nil {
		t.Error("Expected context to be returned")
	}
}

func TestMCPWrappedTool(t *testing.T) {
	wrapper := &MCPToolWrapper{
		name: "test-tool",
	}

	tool := &MCPWrappedTool{
		name:        "mcp_test-tool",
		description: "Test tool",
		wrapper:     wrapper,
	}

	if tool.Name() != "mcp_test-tool" {
		t.Errorf("Expected name 'mcp_test-tool', got '%s'", tool.Name())
	}

	if tool.Description() != "Test tool" {
		t.Errorf("Expected description 'Test tool', got '%s'", tool.Description())
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Expected parameters to be returned")
	}
}

func TestNewAdapter(t *testing.T) {
	registry := tools.NewToolRegistry()
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	adapter, err := NewAdapter(client, adapterConfig, registry)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if adapter == nil {
		t.Error("Expected adapter to be created")
	}

	if adapter.config.Prefix != "mcp_" {
		t.Errorf("Expected prefix 'mcp_', got '%s'", adapter.config.Prefix)
	}
}

func TestNewAdapterNilClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	_, err := NewAdapter(nil, adapterConfig, registry)

	if err == nil {
		t.Error("Expected error for nil client")
	}
}

func TestNewAdapterNilConfig(t *testing.T) {
	registry := tools.NewToolRegistry()
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	adapter, err := NewAdapter(client, nil, registry)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if adapter == nil {
		t.Error("Expected adapter to be created")
	}

	if adapter.config.Prefix != "mcp_" {
		t.Errorf("Expected default prefix 'mcp_', got '%s'", adapter.config.Prefix)
	}
}

func TestNewAdapterEmptyPrefix(t *testing.T) {
	registry := tools.NewToolRegistry()
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "",
	}

	adapter, err := NewAdapter(client, adapterConfig, registry)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if adapter.config.Prefix != "mcp_" {
		t.Errorf("Expected default prefix 'mcp_', got '%s'", adapter.config.Prefix)
	}
}

func TestAdapterGetClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	adapter, _ := NewAdapter(client, adapterConfig, registry)

	retrievedClient := adapter.GetClient()

	if retrievedClient == nil {
		t.Error("Expected client to be returned")
	}

	if retrievedClient != client {
		t.Error("Expected same client to be returned")
	}
}

func TestAdapterGetConfig(t *testing.T) {
	registry := tools.NewToolRegistry()
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	adapter, _ := NewAdapter(client, adapterConfig, registry)

	retrievedConfig := adapter.GetConfig()

	if retrievedConfig == nil {
		t.Error("Expected config to be returned")
	}

	if retrievedConfig.Prefix != "mcp_" {
		t.Errorf("Expected prefix 'mcp_', got '%s'", retrievedConfig.Prefix)
	}
}

func TestMCPToolWrapperGetMetadata(t *testing.T) {
	wrapper := &MCPToolWrapper{
		name: "test-tool",
		metadata: map[string]interface{}{
			"client_name": "test-client",
			"type":        "mcp",
		},
	}

	metadata := wrapper.GetMetadata()

	if metadata == nil {
		t.Error("Expected metadata to be returned")
	}

	if metadata["client_name"] != "test-client" {
		t.Error("Expected client_name to be 'test-client'")
	}
}

func TestMCPManagerClose(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	err := manager.Close()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := manager.GetContext()
	select {
	case <-ctx.Done():
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestMCPManagerDisconnectAll(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	err := manager.DisconnectAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestMCPManagerDisconnectClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	err := manager.DisconnectClient("test-client")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestMCPManagerDisconnectNonExistentClient(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	err := manager.DisconnectClient("non-existent")

	if err == nil {
		t.Error("Expected error for non-existent client")
	}
}

func TestMCPManagerGetAdapter(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)
	adapterConfig := &AdapterConfig{
		ClientName: "test-client",
		Prefix:     "mcp_",
	}

	manager.AddClient(client, adapterConfig)

	adapter, exists := manager.GetAdapter("test-client")

	if !exists {
		t.Error("Expected adapter to exist")
	}

	if adapter == nil {
		t.Error("Expected adapter to be returned")
	}
}

func TestMCPManagerGetNonExistentAdapter(t *testing.T) {
	registry := tools.NewToolRegistry()
	manager := NewMCPManager(registry)

	_, exists := manager.GetAdapter("non-existent")

	if exists {
		t.Error("Expected adapter to not exist")
	}
}

func TestClientMarshalJSON(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	data, err := json.Marshal(client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if data == nil {
		t.Error("Expected JSON data to be returned")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result["name"] != "test-client" {
		t.Errorf("Expected name 'test-client', got '%v'", result["name"])
	}
}

func TestHTTPTransportSendRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	headers := map[string]string{
		"Authorization": "Bearer test-token",
	}

	transport := NewHTTPTransport(server.URL, headers, 30)

	ctx := context.Background()
	payload := map[string]interface{}{
		"test": "value",
	}

	response, err := transport.sendRequest(ctx, "test", payload)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Error("Expected response to be returned")
	}
}

func TestHTTPTransportSendRequestError(t *testing.T) {
	transport := NewHTTPTransport("http://invalid-url-that-does-not-exist.local", nil, 30)

	ctx := context.Background()
	payload := map[string]interface{}{
		"test": "value",
	}

	_, err := transport.sendRequest(ctx, "test", payload)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestClientClose(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	err := client.Close()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := client.ctx
	select {
	case <-ctx.Done():
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestClientListResourcesNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	_, err := client.ListResources(ctx)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestClientReadResourceNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	_, err := client.ReadResource(ctx, "test://resource")

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestClientListPromptsNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	_, err := client.ListPrompts(ctx)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestClientGetPromptNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	_, err := client.GetPrompt(ctx, "test-prompt", nil)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestClientSendNotificationNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	err := client.SendNotification(ctx, "test/method", nil)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestClientDisconnectNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	err := client.Disconnect()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClientGetToolsNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	tools := client.GetTools()

	if tools == nil {
		t.Error("Expected tools to be returned")
	}

	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}
}

func TestClientGetToolNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	_, exists := client.GetTool("test-tool")

	if exists {
		t.Error("Expected tool to not exist")
	}
}

func TestClientExecuteToolNotConnected(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	ctx := context.Background()
	_, err := client.ExecuteTool(ctx, "test-tool", nil)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}

func TestMCPWrappedToolExecute(t *testing.T) {
	config := &ClientConfig{
		Name:     "test-client",
		Endpoint: "http://example.com",
	}

	client, _ := NewClient(config)

	wrapper := &MCPToolWrapper{
		name:   "test-tool",
		client: client,
	}

	tool := &MCPWrappedTool{
		name:        "mcp_test-tool",
		description: "Test tool",
		wrapper:     wrapper,
	}

	ctx := context.Background()
	_, err := tool.Execute(ctx, nil)

	if err == nil {
		t.Error("Expected error for not connected client")
	}
}
