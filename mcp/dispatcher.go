package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603
)

var (
	ErrToolRequired         = errors.New("tool required")
	ErrToolNotFound         = errors.New("tool not found")
	ErrServerNotInitialized = errors.New("server not initialized")
)

func NormalizeToolName(target string, action string) string {
	target = strings.TrimSpace(target)
	action = strings.TrimSpace(action)

	switch {
	case target == "":
		return action
	case action == "":
		return target
	case strings.HasPrefix(action, target+"."):
		return action
	default:
		return target + "." + action
	}
}

func DecodeArguments[T any](arguments map[string]any) (T, error) {
	var out T

	if len(arguments) == 0 {
		return out, nil
	}

	raw, err := json.Marshal(arguments)
	if err != nil {
		return out, fmt.Errorf("marshal arguments: %w", err)
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode arguments: %w", err)
	}

	return out, nil
}

func ToCallToolResult(result any) (CallToolResult, error) {
	switch value := result.(type) {
	case CallToolResult:
		if len(value.Content) == 0 {
			value.Content = []TextContent{{Type: "text", Text: ""}}
		}
		return value, nil
	case *CallToolResult:
		if value == nil {
			return NewTextResult(""), nil
		}
		if len(value.Content) == 0 {
			value.Content = []TextContent{{Type: "text", Text: ""}}
		}
		return *value, nil
	case string:
		return NewTextResult(value), nil
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return CallToolResult{}, fmt.Errorf("marshal tool result: %w", err)
	}

	textResult := NewTextResult(string(raw))

	var structured map[string]any
	if err := json.Unmarshal(raw, &structured); err == nil {
		textResult.StructuredContent = structured
	}

	return textResult, nil
}

func NewTextResult(text string) CallToolResult {
	return CallToolResult{
		Content: []TextContent{
			{Type: "text", Text: text},
		},
	}
}

func NewErrorResult(err error) CallToolResult {
	return CallToolResult{
		Content: []TextContent{
			{Type: "text", Text: err.Error()},
		},
		IsError: true,
	}
}

func NewJSONRPCResult(id json.RawMessage, result any) *JSONRPCMessage {
	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      cloneID(id),
		Result:  result,
	}
}

func NewJSONRPCErrorMessage(id json.RawMessage, code int, message string, data any) *JSONRPCMessage {
	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      cloneID(id),
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func cloneID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return nil
	}

	out := make([]byte, len(id))
	copy(out, id)
	return out
}

func DecodeParams[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 {
		return out, nil
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode params: %w", err)
	}

	return out, nil
}
