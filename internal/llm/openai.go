package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	config      *Config
	httpClient  *http.Client
	baseURL     string
	rateLimiter *RateLimiter
	monitor     *Monitor
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewOpenAIProvider(config *Config) *OpenAIProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:     baseURL,
		rateLimiter: NewRateLimiter(60, time.Minute),
		monitor:     NewMonitor(),
	}
}

func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	p.rateLimiter.Wait()

	startTime := time.Now()
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				p.monitor.RecordRequest("openai", time.Since(startTime), 0, ctx.Err())
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		resp, err := p.doRequest(ctx, req)
		if err == nil {
			p.monitor.RecordRequest("openai", time.Since(startTime), resp.Usage.TotalTokens, nil)
			return resp, nil
		}

		lastErr = err

		if IsRetryableError(err) {
			continue
		}

		break
	}

	p.monitor.RecordRequest("openai", time.Since(startTime), 0, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (p *OpenAIProvider) doRequest(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = p.config.MaxTokens
	}

	openAIReq := &OpenAIRequest{
		Model:       p.config.Model,
		Messages:    make([]OpenAIMessage, 0),
		MaxTokens:   req.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      false,
	}

	for _, msg := range req.Messages {
		openAIReq.Messages = append(openAIReq.Messages, OpenAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

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

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	content := ""
	if len(openAIResp.Choices) > 0 {
		content = openAIResp.Choices[0].Message.Content
	}

	return &CompletionResponse{
		Content: content,
		Usage: Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req *CompletionRequest, callback func(chunk string) error) error {
	p.rateLimiter.Wait()

	if req.MaxTokens == 0 {
		req.MaxTokens = p.config.MaxTokens
	}

	openAIReq := &OpenAIRequest{
		Model:       p.config.Model,
		Messages:    make([]OpenAIMessage, 0),
		MaxTokens:   req.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      true,
	}

	for _, msg := range req.Messages {
		openAIReq.Messages = append(openAIReq.Messages, OpenAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return HandleHTTPError(resp.StatusCode, string(body))
	}

	scanner := newLineScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "data: [DONE]" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if choices, ok := event["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							if err := callback(content); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (p *OpenAIProvider) GetModel() string {
	return p.config.Model
}
