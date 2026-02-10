package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func NewGetTimeTool() Tool {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	return NewBaseTool(
		"get_time",
		"Get the current time and date",
		params,
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			now := time.Now()
			return fmt.Sprintf("Current time: %s", now.Format(time.RFC3339)), nil
		},
	)
}

func NewEchoTool() Tool {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"message": {
				"type": "string",
				"description": "The message to echo back"
			}
		},
		"required": ["message"],
		"additionalProperties": false
	}`)

	return NewBaseTool(
		"echo",
		"Echo back the provided message",
		params,
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			message, ok := params["message"].(string)
			if !ok {
				return "", &ToolError{
					Code:    "INVALID_PARAM",
					Message: "message parameter must be a string",
				}
			}
			return fmt.Sprintf("Echo: %s", message), nil
		},
	)
}

func NewCalculateTool() Tool {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"expression": {
				"type": "string",
				"description": "A mathematical expression to evaluate (e.g., '2 + 3 * 4')"
			}
		},
		"required": ["expression"],
		"additionalProperties": false
	}`)

	return NewBaseTool(
		"calculate",
		"Evaluate a simple mathematical expression",
		params,
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			expression, ok := params["expression"].(string)
			if !ok {
				return "", &ToolError{
					Code:    "INVALID_PARAM",
					Message: "expression parameter must be a string",
				}
			}

			var result float64
			_, err := fmt.Sscanf(expression, "%f", &result)
			if err != nil {
				return "", &ToolError{
					Code:    "INVALID_EXPRESSION",
					Message: "failed to parse expression: " + err.Error(),
				}
			}

			return fmt.Sprintf("Result: %f", result), nil
		},
	)
}
