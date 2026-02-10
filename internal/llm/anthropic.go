package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicProvider struct {
	config      *Config
	httpClient  *http.Client
	rateLimiter *RateLimiter
	monitor     *Monitor
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func NewAnthropicProvider(config *Config) *AnthropicProvider {
	return &AnthropicProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		rateLimiter: NewRateLimiter(50, time.Minute),
		monitor:     NewMonitor(),
	}
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	p.rateLimiter.Wait()

	startTime := time.Now()
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				p.monitor.RecordRequest("anthropic", time.Since(startTime), 0, ctx.Err())
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		resp, err := p.doRequest(ctx, req)
		if err == nil {
			p.monitor.RecordRequest("anthropic", time.Since(startTime), resp.Usage.TotalTokens, nil)
			return resp, nil
		}

		lastErr = err

		if IsRetryableError(err) {
			continue
		}

		break
	}

	p.monitor.RecordRequest("anthropic", time.Since(startTime), 0, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (p *AnthropicProvider) doRequest(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = p.config.MaxTokens
	}

	anthropicReq := &AnthropicRequest{
		Model:     p.config.Model,
		MaxTokens: req.MaxTokens,
		Messages:  make([]AnthropicMessage, 0),
		Stream:    false,
	}

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-dangerous-direct-browser-access", "false")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, HandleHTTPError(resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	content := ""
	if len(anthropicResp.Content) > 0 {
		content = anthropicResp.Content[0].Text
	}

	return &CompletionResponse{
		Content: content,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) StreamComplete(ctx context.Context, req *CompletionRequest, callback func(chunk string) error) error {
	p.rateLimiter.Wait()

	if req.MaxTokens == 0 {
		req.MaxTokens = p.config.MaxTokens
	}

	anthropicReq := &AnthropicRequest{
		Model:     p.config.Model,
		MaxTokens: req.MaxTokens,
		Messages:  make([]AnthropicMessage, 0),
		Stream:    true,
	}

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-dangerous-direct-browser-access", "false")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleHTTPError(resp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream: %w", err)
		}

		if eventType, ok := event["type"].(string); ok && eventType == "content_block_delta" {
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					if err := callback(text); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (p *AnthropicProvider) GetModel() string {
	return p.config.Model
}
