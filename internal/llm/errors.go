package llm

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidModel      = errors.New("invalid model")
	ErrContextLength     = errors.New("context length exceeded")
	ErrServerUnavailable = errors.New("server unavailable")
	ErrTimeout           = errors.New("request timeout")
	ErrConnectionError   = errors.New("connection error")
	ErrInvalidRequest    = errors.New("invalid request")
)

type LLMError struct {
	Code    string
	Message string
	Err     error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

func NewLLMError(code, message string, err error) *LLMError {
	return &LLMError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func HandleHTTPError(statusCode int, body string) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return NewLLMError("AUTH_ERROR", "Invalid API key", ErrInvalidAPIKey)
	case http.StatusTooManyRequests:
		return NewLLMError("RATE_LIMIT", "Rate limit exceeded", ErrRateLimitExceeded)
	case http.StatusBadRequest:
		return NewLLMError("BAD_REQUEST", "Invalid request", ErrInvalidRequest)
	case http.StatusNotFound:
		return NewLLMError("NOT_FOUND", "Model not found", ErrInvalidModel)
	case http.StatusRequestEntityTooLarge:
		return NewLLMError("CONTEXT_LENGTH", "Context length exceeded", ErrContextLength)
	case http.StatusInternalServerError:
		return NewLLMError("SERVER_ERROR", "Internal server error", ErrServerUnavailable)
	case http.StatusServiceUnavailable:
		return NewLLMError("SERVICE_UNAVAILABLE", "Service unavailable", ErrServerUnavailable)
	case http.StatusGatewayTimeout:
		return NewLLMError("TIMEOUT", "Request timeout", ErrTimeout)
	default:
		return NewLLMError("UNKNOWN", fmt.Sprintf("HTTP error %d: %s", statusCode, body), nil)
	}
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == "RATE_LIMIT" ||
			llmErr.Code == "TIMEOUT" ||
			llmErr.Code == "SERVER_ERROR" ||
			llmErr.Code == "SERVICE_UNAVAILABLE"
	}

	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrServerUnavailable) ||
		errors.Is(err, ErrRateLimitExceeded)
}

func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == "AUTH_ERROR"
	}

	return errors.Is(err, ErrInvalidAPIKey)
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Code == "BAD_REQUEST" ||
			llmErr.Code == "NOT_FOUND" ||
			llmErr.Code == "CONTEXT_LENGTH"
	}

	return errors.Is(err, ErrInvalidRequest) ||
		errors.Is(err, ErrInvalidModel) ||
		errors.Is(err, ErrContextLength)
}