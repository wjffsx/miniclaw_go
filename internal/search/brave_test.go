package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wjffsx/miniclaw_go/internal/tools"
)

func TestNewBraveSearchClient(t *testing.T) {
	config := &SearchConfig{
		APIKey:     "test-api-key",
		MaxResults: 5,
		Timeout:    10,
	}

	client := NewBraveSearchClient(config)
	if client == nil {
		t.Fatal("NewBraveSearchClient returned nil")
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", client.apiKey)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewBraveSearchClient_DefaultConfig(t *testing.T) {
	client := NewBraveSearchClient(nil)
	if client == nil {
		t.Fatal("NewBraveSearchClient returned nil")
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestBraveSearchClient_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-api-key" {
			t.Error("API key header not set correctly")
		}

		response := SearchResponse{
			Query: struct {
				Original string `json:"original"`
			}{
				Original: "test query",
			},
			Web: struct {
				Results []SearchResult `json:"results"`
			}{
				Results: []SearchResult{
					{
						Title:   "Test Result 1",
						URL:     "https://example.com/1",
						Snippet: "This is a test result",
					},
					{
						Title:   "Test Result 2",
						URL:     "https://example.com/2",
						Snippet: "Another test result",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &SearchConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	client := NewBraveSearchClient(config)

	ctx := context.Background()
	results, err := client.Search(ctx, "test query", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Test Result 1" {
		t.Errorf("Expected title 'Test Result 1', got '%s'", results[0].Title)
	}
}

func TestBraveSearchClient_Search_EmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SearchResponse{
			Web: struct {
				Results []SearchResult `json:"results"`
			}{
				Results: []SearchResult{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &SearchConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	client := NewBraveSearchClient(config)

	ctx := context.Background()
	results, err := client.Search(ctx, "empty query", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestBraveSearchClient_Search_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &SearchConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	client := NewBraveSearchClient(config)

	ctx := context.Background()
	_, err := client.Search(ctx, "test query", 10)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestWebSearchTool_Name(t *testing.T) {
	tool := NewWebSearchTool(nil)
	if tool.Name() != "web_search" {
		t.Errorf("Expected name 'web_search', got '%s'", tool.Name())
	}
}

func TestWebSearchTool_Description(t *testing.T) {
	tool := NewWebSearchTool(nil)
	if tool.Description() == "" {
		t.Error("Description is empty")
	}
}

func TestWebSearchTool_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SearchResponse{
			Query: struct {
				Original string `json:"original"`
			}{
				Original: "test query",
			},
			Web: struct {
				Results []SearchResult `json:"results"`
			}{
				Results: []SearchResult{
					{
						Title:   "Test Result",
						URL:     "https://example.com",
						Snippet: "Test snippet",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &SearchConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	client := NewBraveSearchClient(config)

	tool := NewWebSearchTool(client)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test query",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}

	if !contains(result, "Test Result") {
		t.Error("Result does not contain expected content")
	}
}

func TestWebSearchTool_Execute_EmptyQuery(t *testing.T) {
	config := &SearchConfig{
		APIKey: "test-api-key",
	}
	client := NewBraveSearchClient(config)

	tool := NewWebSearchTool(client)

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"query": "",
	})
	if err == nil {
		t.Error("Expected error for empty query, got nil")
	}

	var toolErr *tools.ToolError
	if !tools.AsToolError(err, &toolErr) {
		t.Error("Expected ToolError")
	}

	if toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected error code 'INVALID_PARAM', got '%s'", toolErr.Code)
	}
}

func TestWebSearchTool_Execute_MissingQuery(t *testing.T) {
	config := &SearchConfig{
		APIKey: "test-api-key",
	}
	client := NewBraveSearchClient(config)

	tool := NewWebSearchTool(client)

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing query, got nil")
	}

	var toolErr *tools.ToolError
	if !tools.AsToolError(err, &toolErr) {
		t.Error("Expected ToolError")
	}

	if toolErr.Code != "INVALID_PARAM" {
		t.Errorf("Expected error code 'INVALID_PARAM', got '%s'", toolErr.Code)
	}
}

func TestWebSearchTool_Execute_CustomCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SearchResponse{
			Web: struct {
				Results []SearchResult `json:"results"`
			}{
				Results: []SearchResult{
					{Title: "Result 1", URL: "https://example.com/1", Snippet: "Snippet 1"},
					{Title: "Result 2", URL: "https://example.com/2", Snippet: "Snippet 2"},
					{Title: "Result 3", URL: "https://example.com/3", Snippet: "Snippet 3"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &SearchConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	}
	client := NewBraveSearchClient(config)

	tool := NewWebSearchTool(client)

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test",
		"count": float64(3),
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result is empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
