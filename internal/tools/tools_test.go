package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	t.Run("RegisterTool", func(t *testing.T) {
		tool := NewGetTimeTool()
		err := registry.Register(tool)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("RegisterDuplicateTool", func(t *testing.T) {
		tool := NewGetTimeTool()
		err := registry.Register(tool)
		if err == nil {
			t.Error("expected error for duplicate tool")
		}

		if toolErr, ok := err.(*ToolError); ok {
			if toolErr.Code != "DUPLICATE_TOOL" {
				t.Errorf("expected DUPLICATE_TOOL error, got %s", toolErr.Code)
			}
		} else {
			t.Error("expected ToolError")
		}
	})

	t.Run("RegisterToolWithEmptyName", func(t *testing.T) {
		params := json.RawMessage(`{"type": "object"}`)
		tool := NewBaseTool("", "test", params, func(ctx context.Context, params map[string]interface{}) (string, error) {
			return "test", nil
		})

		err := registry.Register(tool)
		if err == nil {
			t.Error("expected error for empty tool name")
		}

		if toolErr, ok := err.(*ToolError); ok {
			if toolErr.Code != "INVALID_NAME" {
				t.Errorf("expected INVALID_NAME error, got %s", toolErr.Code)
			}
		} else {
			t.Error("expected ToolError")
		}
	})

	t.Run("GetTool", func(t *testing.T) {
		tool, exists := registry.Get("get_time")
		if !exists {
			t.Error("expected tool to exist")
		}
		if tool == nil {
			t.Error("expected non-nil tool")
		}
		if tool.Name() != "get_time" {
			t.Errorf("expected tool name 'get_time', got '%s'", tool.Name())
		}
	})

	t.Run("GetNonExistentTool", func(t *testing.T) {
		_, exists := registry.Get("nonexistent")
		if exists {
			t.Error("expected tool to not exist")
		}
	})

	t.Run("ListTools", func(t *testing.T) {
		tools := registry.List()
		if len(tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("GetSchemas", func(t *testing.T) {
		schemas := registry.GetSchemas()
		if len(schemas) != 1 {
			t.Errorf("expected 1 schema, got %d", len(schemas))
		}
		if schemas[0].Name != "get_time" {
			t.Errorf("expected schema name 'get_time', got '%s'", schemas[0].Name)
		}
	})

	t.Run("UnregisterTool", func(t *testing.T) {
		registry.Unregister("get_time")
		_, exists := registry.Get("get_time")
		if exists {
			t.Error("expected tool to not exist after unregister")
		}
	})
}

func TestToolExecutor(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(NewGetTimeTool())
	registry.Register(NewEchoTool())

	executor := NewToolExecutor(registry)
	ctx := context.Background()

	t.Run("ExecuteExistingTool", func(t *testing.T) {
		call, err := executor.Execute(ctx, "get_time", map[string]interface{}{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Name != "get_time" {
			t.Errorf("expected tool name 'get_time', got '%s'", call.Name)
		}
		if call.Result == "" {
			t.Error("expected non-empty result")
		}
		if call.Error != "" {
			t.Errorf("expected no error, got '%s'", call.Error)
		}
	})

	t.Run("ExecuteNonExistentTool", func(t *testing.T) {
		_, err := executor.Execute(ctx, "nonexistent", map[string]interface{}{})
		if err == nil {
			t.Error("expected error for nonexistent tool")
		}

		if toolErr, ok := err.(*ToolError); ok {
			if toolErr.Code != "TOOL_NOT_FOUND" {
				t.Errorf("expected TOOL_NOT_FOUND error, got %s", toolErr.Code)
			}
		} else {
			t.Error("expected ToolError")
		}
	})

	t.Run("ExecuteEchoTool", func(t *testing.T) {
		params := map[string]interface{}{
			"message": "Hello, World!",
		}
		call, err := executor.Execute(ctx, "echo", params)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Result != "Echo: Hello, World!" {
			t.Errorf("expected 'Echo: Hello, World!', got '%s'", call.Result)
		}
	})

	t.Run("ExecuteEchoToolWithInvalidParam", func(t *testing.T) {
		params := map[string]interface{}{
			"message": 123,
		}
		call, err := executor.Execute(ctx, "echo", params)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Error == "" {
			t.Error("expected error for invalid parameter")
		}
	})

	t.Run("ExecuteMultiple", func(t *testing.T) {
		calls := []ToolCall{
			{Name: "get_time", Input: map[string]interface{}{}},
			{Name: "echo", Input: map[string]interface{}{"message": "test"}},
		}
		results, err := executor.ExecuteMultiple(ctx, calls)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		if results[0].Name != "get_time" {
			t.Errorf("expected first result name 'get_time', got '%s'", results[0].Name)
		}
		if results[1].Result != "Echo: test" {
			t.Errorf("expected second result 'Echo: test', got '%s'", results[1].Result)
		}
	})
}

func TestBuiltInTools(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewToolExecutor(registry)
	ctx := context.Background()

	t.Run("GetTimeTool", func(t *testing.T) {
		tool := NewGetTimeTool()
		err := registry.Register(tool)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		call, err := executor.Execute(ctx, "get_time", map[string]interface{}{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Result == "" {
			t.Error("expected non-empty result")
		}
		if call.Error != "" {
			t.Errorf("expected no error, got '%s'", call.Error)
		}
	})

	t.Run("EchoTool", func(t *testing.T) {
		tool := NewEchoTool()
		err := registry.Register(tool)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		params := map[string]interface{}{
			"message": "Test message",
		}
		call, err := executor.Execute(ctx, "echo", params)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Result != "Echo: Test message" {
			t.Errorf("expected 'Echo: Test message', got '%s'", call.Result)
		}
	})

	t.Run("CalculateTool", func(t *testing.T) {
		tool := NewCalculateTool()
		err := registry.Register(tool)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		params := map[string]interface{}{
			"expression": "42",
		}
		call, err := executor.Execute(ctx, "calculate", params)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if call.Result == "" {
			t.Error("expected non-empty result")
		}
	})
}

func TestToolError(t *testing.T) {
	t.Run("ToolErrorWithoutWrap", func(t *testing.T) {
		err := &ToolError{
			Code:    "TEST_ERROR",
			Message: "test error message",
		}
		if err.Error() != "test error message" {
			t.Errorf("expected 'test error message', got '%s'", err.Error())
		}
	})

	t.Run("ToolErrorWithWrap", func(t *testing.T) {
		wrappedErr := &ToolError{
			Code:    "WRAPPED_ERROR",
			Message: "wrapped error message",
			Err:     &ToolError{Code: "INNER_ERROR", Message: "inner error"},
		}
		expected := "wrapped error message: inner error"
		if wrappedErr.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, wrappedErr.Error())
		}
	})

	t.Run("ToolErrorUnwrap", func(t *testing.T) {
		innerErr := &ToolError{Code: "INNER_ERROR", Message: "inner error"}
		err := &ToolError{
			Code:    "OUTER_ERROR",
			Message: "outer error message",
			Err:     innerErr,
		}
		if err.Unwrap() != innerErr {
			t.Error("expected unwrapped error to be inner error")
		}
	})
}

func TestBaseTool(t *testing.T) {
	params := json.RawMessage(`{"type": "object"}`)
	tool := NewBaseTool(
		"test_tool",
		"test description",
		params,
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			return "test result", nil
		},
	)

	t.Run("Name", func(t *testing.T) {
		if tool.Name() != "test_tool" {
			t.Errorf("expected 'test_tool', got '%s'", tool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		if tool.Description() != "test description" {
			t.Errorf("expected 'test description', got '%s'", tool.Description())
		}
	})

	t.Run("Parameters", func(t *testing.T) {
		if tool.Parameters() == nil {
			t.Error("expected non-nil parameters")
		}
	})

	t.Run("Execute", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != "test result" {
			t.Errorf("expected 'test result', got '%s'", result)
		}
	})
}

func TestTimedTool(t *testing.T) {
	params := json.RawMessage(`{"type": "object"}`)
	baseTool := NewBaseTool(
		"timed_tool",
		"timed description",
		params,
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			return "timed result", nil
		},
	)

	timedTool := NewTimedTool(baseTool)

	t.Run("Name", func(t *testing.T) {
		if timedTool.Name() != "timed_tool" {
			t.Errorf("expected 'timed_tool', got '%s'", timedTool.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		if timedTool.Description() != "timed description" {
			t.Errorf("expected 'timed description', got '%s'", timedTool.Description())
		}
	})

	t.Run("Parameters", func(t *testing.T) {
		if timedTool.Parameters() == nil {
			t.Error("expected non-nil parameters")
		}
	})

	t.Run("ExecuteSuccess", func(t *testing.T) {
		result, err := timedTool.Execute(context.Background(), map[string]interface{}{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result != "timed result" {
			t.Errorf("expected 'timed result', got '%s'", result)
		}
	})

	t.Run("ExecuteFailure", func(t *testing.T) {
		errorTool := NewBaseTool(
			"error_tool",
			"error description",
			params,
			func(ctx context.Context, params map[string]interface{}) (string, error) {
				return "", &ToolError{Code: "TEST_ERROR", Message: "test error"}
			},
		)

		timedErrorTool := NewTimedTool(errorTool)
		_, err := timedErrorTool.Execute(context.Background(), map[string]interface{}{})
		if err == nil {
			t.Error("expected error")
		}

		if toolErr, ok := err.(*ToolError); ok {
			if toolErr.Code != "EXECUTION_FAILED" {
				t.Errorf("expected EXECUTION_FAILED error, got %s", toolErr.Code)
			}
		} else {
			t.Error("expected ToolError")
		}
	})
}
