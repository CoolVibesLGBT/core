package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestLifecycleRequiresInitializedNotification(t *testing.T) {
	server := NewMCPServer()
	server.RegisterTool(NewTool(
		ToolDefinition{
			Name:        "test.echo",
			Description: "Echo a value.",
			InputSchema: JSONSchema{Type: "object"},
		},
		func(ctx context.Context, req CallToolRequest) (any, error) {
			return map[string]any{"echo": req.Arguments["value"]}, nil
		},
	))

	connection := server.NewConnection()

	initMessage := JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}`),
	}

	initResponse := server.HandleMessage(context.Background(), connection, initMessage)
	if initResponse == nil || initResponse.Error != nil {
		t.Fatalf("initialize failed: %#v", initResponse)
	}

	toolsListBeforeInitialized := server.HandleMessage(context.Background(), connection, JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	if toolsListBeforeInitialized == nil || toolsListBeforeInitialized.Error == nil {
		t.Fatalf("expected tools/list to fail before notifications/initialized, got %#v", toolsListBeforeInitialized)
	}

	server.HandleMessage(context.Background(), connection, JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/initialized",
	})

	toolsListAfterInitialized := server.HandleMessage(context.Background(), connection, JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`3`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	if toolsListAfterInitialized == nil || toolsListAfterInitialized.Error != nil {
		t.Fatalf("expected tools/list to succeed after notifications/initialized, got %#v", toolsListAfterInitialized)
	}
}
