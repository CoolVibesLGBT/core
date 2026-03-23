package mcp

import (
	"bytes"
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
	ErrEmptyBatch           = errors.New("empty batch")
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

func DecodeWireMessages(raw []byte) ([]JSONRPCMessage, bool, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, false, fmt.Errorf("empty payload")
	}

	if raw[0] == '[' {
		var messages []JSONRPCMessage
		if err := json.Unmarshal(raw, &messages); err != nil {
			return nil, true, err
		}
		if len(messages) == 0 {
			return nil, true, ErrEmptyBatch
		}
		return messages, true, nil
	}

	var message JSONRPCMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, false, err
	}

	return []JSONRPCMessage{message}, false, nil
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
