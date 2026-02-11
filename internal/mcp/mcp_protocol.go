package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type Protocol interface {
	Connect(ctx context.Context) error
	Close() error
	ListTools(ctx context.Context) ([]*MCPTool, error)
	CallTool(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolCall, error)
	ListResources(ctx context.Context) ([]map[string]interface{}, error)
	ReadResource(ctx context.Context, uri string) (string, error)
	ListPrompts(ctx context.Context) ([]map[string]interface{}, error)
	GetPrompt(ctx context.Context, name string, args map[string]interface{}) (string, error)
	SendNotification(ctx context.Context, method string, params map[string]interface{}) error
}

type HTTPTransport struct {
	client   *http.Client
	endpoint string
	headers  map[string]string
	timeout  time.Duration
}

func NewHTTPTransport(endpoint string, headers map[string]string, timeout int) *HTTPTransport {
	return &HTTPTransport{
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		endpoint: endpoint,
		headers:  headers,
		timeout:  time.Duration(timeout) * time.Second,
	}
}

func (t *HTTPTransport) sendRequest(ctx context.Context, method string, payload map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return responseBody, nil
}

type JSONRPCProtocol struct {
	transport *HTTPTransport
	requestID int
}

func NewProtocol(config *ClientConfig) (Protocol, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	timeout := 30
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	transport := NewHTTPTransport(config.Endpoint, config.Headers, timeout)

	return &JSONRPCProtocol{
		transport: transport,
		requestID: 0,
	}, nil
}

func (p *JSONRPCProtocol) Connect(ctx context.Context) error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "miniclaw-go",
				"version": "0.1.0",
			},
		},
	}

	_, err := p.transport.sendRequest(ctx, "initialize", payload)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	return nil
}

func (p *JSONRPCProtocol) Close() error {
	return nil
}

func (p *JSONRPCProtocol) ListTools(ctx context.Context) ([]*MCPTool, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	response, err := p.transport.sendRequest(ctx, "tools/list", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	var result struct {
		Result struct {
			Tools []struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				InputSchema map[string]interface{} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	tools := make([]*MCPTool, 0, len(result.Result.Tools))
	for _, tool := range result.Result.Tools {
		tools = append(tools, &MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	return tools, nil
}

func (p *JSONRPCProtocol) CallTool(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolCall, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": params,
		},
	}

	response, err := p.transport.sendRequest(ctx, "tools/call", payload)
	if err != nil {
		return &tools.ToolCall{
			Name:  name,
			Input: params,
			Error: err.Error(),
		}, nil
	}

	var result struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return &tools.ToolCall{
			Name:  name,
			Input: params,
			Error: fmt.Sprintf("failed to unmarshal response: %v", err),
		}, nil
	}

	if result.Result.IsError {
		errorMsg := "unknown error"
		if len(result.Result.Content) > 0 {
			errorMsg = result.Result.Content[0].Text
		}
		return &tools.ToolCall{
			Name:  name,
			Input: params,
			Error: errorMsg,
		}, nil
	}

	resultText := ""
	for _, content := range result.Result.Content {
		if content.Type == "text" {
			resultText += content.Text + "\n"
		}
	}

	return &tools.ToolCall{
		Name:   name,
		Input:  params,
		Result: resultText,
	}, nil
}

func (p *JSONRPCProtocol) ListResources(ctx context.Context) ([]map[string]interface{}, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	response, err := p.transport.sendRequest(ctx, "resources/list", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	var result struct {
		Result struct {
			Resources []map[string]interface{} `json:"resources"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Result.Resources, nil
}

func (p *JSONRPCProtocol) ReadResource(ctx context.Context, uri string) (string, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": uri,
		},
	}

	response, err := p.transport.sendRequest(ctx, "resources/read", payload)
	if err != nil {
		return "", fmt.Errorf("failed to read resource: %w", err)
	}

	var result struct {
		Result struct {
			Contents []struct {
				URI      string `json:"uri"`
				MimeType string `json:"mimeType"`
				Text     string `json:"text"`
			} `json:"contents"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Result.Contents) == 0 {
		return "", fmt.Errorf("no content returned")
	}

	return result.Result.Contents[0].Text, nil
}

func (p *JSONRPCProtocol) ListPrompts(ctx context.Context) ([]map[string]interface{}, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "prompts/list",
		"params":  map[string]interface{}{},
	}

	response, err := p.transport.sendRequest(ctx, "prompts/list", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	var result struct {
		Result struct {
			Prompts []map[string]interface{} `json:"prompts"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Result.Prompts, nil
}

func (p *JSONRPCProtocol) GetPrompt(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.nextRequestID(),
		"method":  "prompts/get",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}

	response, err := p.transport.sendRequest(ctx, "prompts/get", payload)
	if err != nil {
		return "", fmt.Errorf("failed to get prompt: %w", err)
	}

	var result struct {
		Result struct {
			Description string `json:"description"`
			Messages    []struct {
				Role    string `json:"role"`
				Content struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		} `json:"result"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	promptText := ""
	for _, msg := range result.Result.Messages {
		if msg.Content.Type == "text" {
			promptText += msg.Content.Text + "\n"
		}
	}

	return promptText, nil
}

func (p *JSONRPCProtocol) SendNotification(ctx context.Context, method string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	_, err := p.transport.sendRequest(ctx, method, payload)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

func (p *JSONRPCProtocol) nextRequestID() int {
	p.requestID++
	return p.requestID
}
