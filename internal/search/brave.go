package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

type BraveSearchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type SearchConfig struct {
	APIKey     string
	BaseURL    string
	MaxResults int
	Timeout    time.Duration
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"description"`
}

type SearchResponse struct {
	Query struct {
		Original string `json:"original"`
	} `json:"query"`
	Web struct {
		Results []SearchResult `json:"results"`
	} `json:"web"`
}

func NewBraveSearchClient(config *SearchConfig) *BraveSearchClient {
	if config == nil {
		config = &SearchConfig{
			MaxResults: 10,
			Timeout:    30 * time.Second,
		}
	}

	if config.MaxResults <= 0 {
		config.MaxResults = 10
	}

	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.search.brave.com/res/v1/web/search"
	}

	return &BraveSearchClient{
		apiKey:  config.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (c *BraveSearchClient) Search(ctx context.Context, query string, count int) ([]SearchResult, error) {
	if count <= 0 {
		count = 10
	}

	if count > 20 {
		count = 20
	}

	searchURL := fmt.Sprintf("%s?q=%s&count=%d", c.baseURL, url.QueryEscape(query), count)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return searchResp.Web.Results, nil
}

type WebSearchTool struct {
	client *BraveSearchClient
}

func NewWebSearchTool(client *BraveSearchClient) *WebSearchTool {
	return &WebSearchTool{
		client: client,
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for information using Brave Search API"
}

func (t *WebSearchTool) Parameters() json.RawMessage {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			},
			"count": {
				"type": "integer",
				"description": "Number of results to return (1-20, default 10)",
				"default": 10,
				"minimum": 1,
				"maximum": 20
			}
		},
		"required": ["query"],
		"additionalProperties": false
	}`)
	return params
}

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "query parameter must be a string",
		}
	}

	if query == "" {
		return "", &tools.ToolError{
			Code:    "INVALID_PARAM",
			Message: "query parameter cannot be empty",
		}
	}

	count := 10
	if c, ok := params["count"].(float64); ok {
		count = int(c)
		if count < 1 {
			count = 1
		}
		if count > 20 {
			count = 20
		}
	}

	results, err := t.client.Search(ctx, query, count)
	if err != nil {
		return "", &tools.ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "failed to perform web search",
			Err:     err,
		}
	}

	if len(results) == 0 {
		return "No search results found", nil
	}

	output := fmt.Sprintf("Found %d search results for '%s':\n\n", len(results), query)
	for i, result := range results {
		output += fmt.Sprintf("%d. %s\n", i+1, result.Title)
		output += fmt.Sprintf("   URL: %s\n", result.URL)
		output += fmt.Sprintf("   %s\n\n", result.Snippet)
	}

	return output, nil
}
