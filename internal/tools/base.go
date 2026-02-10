package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type BaseTool struct {
	name        string
	description string
	parameters  json.RawMessage
	executeFunc func(ctx context.Context, params map[string]interface{}) (string, error)
}

func NewBaseTool(name, description string, parameters json.RawMessage, executeFunc func(ctx context.Context, params map[string]interface{}) (string, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		parameters:  parameters,
		executeFunc: executeFunc,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) Parameters() json.RawMessage {
	return t.parameters
}

func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	return t.executeFunc(ctx, params)
}

type TimedTool struct {
	tool Tool
}

func NewTimedTool(tool Tool) *TimedTool {
	return &TimedTool{
		tool: tool,
	}
}

func (t *TimedTool) Name() string {
	return t.tool.Name()
}

func (t *TimedTool) Description() string {
	return t.tool.Description()
}

func (t *TimedTool) Parameters() json.RawMessage {
	return t.tool.Parameters()
}

func (t *TimedTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	start := time.Now()
	result, err := t.tool.Execute(ctx, params)
	duration := time.Since(start)

	if err != nil {
		return "", &ToolError{
			Code:    "EXECUTION_FAILED",
			Message: "tool execution failed after " + duration.String(),
			Err:     err,
		}
	}

	return result, nil
}