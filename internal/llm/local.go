package llm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type LocalProvider struct {
	config      *Config
	modelPath   string
	modelType   string
	chatHistory map[string][]Message
}

type LocalModelConfig struct {
	Enabled bool
	Path    string
	Type    string
	Params  LocalModelParams
}

type LocalModelParams struct {
	ContextSize int
	NGPULayers  int
	Threads     int
	BatchSize   int
	Temperature float64
	TopP        float64
	TopK        int
}

func NewLocalProvider(config *Config) *LocalProvider {
	return &LocalProvider{
		config:      config,
		modelPath:   config.LocalModel.Path,
		modelType:   config.LocalModel.Type,
		chatHistory: make(map[string][]Message),
	}
}

func (p *LocalProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if _, err := exec.LookPath("llama-cli"); err != nil {
		return nil, fmt.Errorf("llama-cli not found. Please install llama.cpp: %w", err)
	}

	args := p.buildArgs(req)

	cmd := exec.CommandContext(ctx, "llama-cli", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run llama-cli: %w, output: %s", err, string(output))
	}

	response := p.parseOutput(string(output))

	return &CompletionResponse{
		Content: response,
		Usage: Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}, nil
}

func (p *LocalProvider) GetModel() string {
	return p.modelPath
}

func (p *LocalProvider) buildArgs(req *CompletionRequest) []string {
	args := []string{
		"-m", p.modelPath,
		"--ctx-size", "2048",
		"--threads", "4",
		"--batch-size", "512",
		"--temp", fmt.Sprintf("%.2f", p.config.Temperature),
		"--top-p", "0.95",
		"--top-k", "40",
		"--n-predict", fmt.Sprintf("%d", req.MaxTokens),
	}

	if req.MaxTokens == 0 {
		args = append(args, "--n-predict", "512")
	}

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			args = append(args, "--system", msg.Content)
		} else if msg.Role == RoleUser {
			args = append(args, "--prompt", msg.Content)
		} else if msg.Role == RoleAssistant {
			args = append(args, "--prompt", msg.Content)
		}
	}

	args = append(args, "--color", "false")

	return args
}

func (p *LocalProvider) parseOutput(output string) string {
	lines := strings.Split(output, "\n")
	var response strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "main:") || strings.HasPrefix(line, "llama_print_timings:") {
			continue
		}
		if line != "" {
			response.WriteString(line)
			response.WriteString("\n")
		}
	}

	return strings.TrimSpace(response.String())
}

func (p *LocalProvider) IsAvailable() bool {
	if _, err := exec.LookPath("llama-cli"); err != nil {
		return false
	}

	if _, err := exec.LookPath("llama-server"); err != nil {
		return false
	}

	return true
}

func (p *LocalProvider) StartServer(ctx context.Context, port int) error {
	args := []string{
		"-m", p.modelPath,
		"--port", fmt.Sprintf("%d", port),
		"--ctx-size", "2048",
		"--threads", "4",
		"--host", "0.0.0.0",
	}

	cmd := exec.CommandContext(ctx, "llama-server", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	time.Sleep(2 * time.Second)
	return nil
}

func (p *LocalProvider) StreamComplete(ctx context.Context, req *CompletionRequest, callback func(chunk string) error) error {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return err
	}

	return callback(resp.Content)
}
