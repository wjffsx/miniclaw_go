package tools

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Input    map[string]interface{} `json:"input"`
	Result   string                 `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int64                  `json:"duration,omitempty"`
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) error {
	if tool.Name() == "" {
		return &ToolError{
			Code:    "INVALID_NAME",
			Message: "tool name cannot be empty",
		}
	}

	if _, exists := r.tools[tool.Name()]; exists {
		return &ToolError{
			Code:    "DUPLICATE_TOOL",
			Message: "tool with name '" + tool.Name() + "' already registered",
		}
	}

	r.tools[tool.Name()] = tool
	return nil
}

func (r *ToolRegistry) Unregister(name string) {
	delete(r.tools, name)
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *ToolRegistry) GetSchemas() []ToolSchema {
	schemas := make([]ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return schemas
}

type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolExecutor struct {
	registry *ToolRegistry
}

func NewToolExecutor(registry *ToolRegistry) *ToolExecutor {
	return &ToolExecutor{
		registry: registry,
	}
}

func (e *ToolExecutor) Execute(ctx context.Context, name string, params map[string]interface{}) (*ToolCall, error) {
	tool, exists := e.registry.Get(name)
	if !exists {
		return nil, &ToolError{
			Code:    "TOOL_NOT_FOUND",
			Message: "tool '" + name + "' not found",
		}
	}

	call := &ToolCall{
		ID:    generateID(),
		Name:  name,
		Input: params,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		call.Error = err.Error()
		return call, nil
	}

	call.Result = result
	return call, nil
}

func (e *ToolExecutor) ExecuteMultiple(ctx context.Context, calls []ToolCall) ([]ToolCall, error) {
	results := make([]ToolCall, 0, len(calls))

	for _, call := range calls {
		result, err := e.Execute(ctx, call.Name, call.Input)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}

func (e *ToolExecutor) GetSchemas() []ToolSchema {
	return e.registry.GetSchemas()
}

type ToolError struct {
	Code    string
	Message string
	Err     error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ToolError) Unwrap() error {
	return e.Err
}

func AsToolError(err error, toolErr **ToolError) bool {
	te, ok := err.(*ToolError)
	if ok {
		*toolErr = te
	}
	return ok
}
