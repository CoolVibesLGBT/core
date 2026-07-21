package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestServeStdioReturnsWhenContextIsCanceledWhileInputBlocks(t *testing.T) {
	server := NewMCPServer()
	reader, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- server.ServeStdio(ctx, reader, io.Discard)
	}()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeStdio() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeStdio did not return after cancellation")
	}
	_ = writer.Close()
	_ = reader.Close()
}

func TestLifecycleRequiresInitializedNotification(t *testing.T) {
	server := NewMCPServer()
	if err := server.RegisterTool(NewTool(
		ToolDefinition{
			Name:        "test.echo",
			Description: "Echo a value.",
			InputSchema: JSONSchema{Type: "object"},
		},
		func(ctx context.Context, req CallToolRequest) (any, error) {
			return map[string]any{"echo": req.Arguments["value"]}, nil
		},
	)); err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

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
	if connection.ProtocolVersion() != LatestProtocolVersion {
		t.Fatalf("expected negotiated protocol version %q, got %q", LatestProtocolVersion, connection.ProtocolVersion())
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

func TestInitializeNegotiatesUnsupportedProtocolVersion(t *testing.T) {
	server := NewMCPServer()
	connection := server.NewConnection()

	response := server.HandleMessage(context.Background(), connection, JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion":"2099-01-01",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}`),
	})
	if response == nil || response.Error != nil {
		t.Fatalf("expected initialize negotiation to succeed, got %#v", response)
	}

	result, ok := response.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", response.Result)
	}
	if result.ProtocolVersion != LatestProtocolVersion {
		t.Fatalf("expected negotiated protocol version %q, got %q", LatestProtocolVersion, result.ProtocolVersion)
	}
	if connection.ProtocolVersion() != LatestProtocolVersion {
		t.Fatalf("expected connection protocol version %q, got %q", LatestProtocolVersion, connection.ProtocolVersion())
	}
}

func TestHandleMessageRejectsNullID(t *testing.T) {
	server := NewMCPServer()
	connection := server.NewConnection()

	response := server.HandleMessage(context.Background(), connection, JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`null`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}`),
	})
	if response == nil || response.Error == nil {
		t.Fatalf("expected invalid id error, got %#v", response)
	}
	if response.Error.Code != InvalidRequestCode {
		t.Fatalf("expected invalid request code, got %d", response.Error.Code)
	}
	if string(response.ID) != "null" {
		t.Fatalf("expected error id to be null, got %q", string(response.ID))
	}
	if connection.ProtocolVersion() != "" {
		t.Fatalf("expected connection to remain uninitialized, got protocol %q", connection.ProtocolVersion())
	}
}

func TestServeStdioSupportsBatchForProtocol20250326(t *testing.T) {
	server := NewMCPServer()
	if err := server.RegisterTool(NewTool(
		ToolDefinition{
			Name:        "test.echo",
			Description: "Echo a value.",
			InputSchema: JSONSchema{Type: "object"},
		},
		func(ctx context.Context, req CallToolRequest) (any, error) {
			return map[string]any{"echo": req.Arguments["value"]}, nil
		},
	)); err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`[{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}},{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test.echo","arguments":{"value":"hello"}}}]`,
		"",
	}, "\n")

	var output bytes.Buffer
	if err := server.ServeStdio(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("ServeStdio() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), output.String())
	}

	var initializeResponse JSONRPCMessage
	if err := json.Unmarshal([]byte(lines[0]), &initializeResponse); err != nil {
		t.Fatalf("decode initialize response: %v", err)
	}
	if initializeResponse.Error != nil {
		t.Fatalf("initialize response error: %#v", initializeResponse.Error)
	}

	var batchResponse []JSONRPCMessage
	if err := json.Unmarshal([]byte(lines[1]), &batchResponse); err != nil {
		t.Fatalf("decode batch response: %v", err)
	}
	if len(batchResponse) != 2 {
		t.Fatalf("expected 2 batch responses, got %d", len(batchResponse))
	}
	if batchResponse[0].Error != nil {
		t.Fatalf("tools/list error: %#v", batchResponse[0].Error)
	}
	if batchResponse[1].Error != nil {
		t.Fatalf("tools/call error: %#v", batchResponse[1].Error)
	}
}

func TestRegistryRegisterReturnsErrorOnDuplicateName(t *testing.T) {
	registry := NewRegistry()
	tool := NewTool(
		ToolDefinition{Name: "test.echo", InputSchema: JSONSchema{Type: "object"}},
		func(ctx context.Context, req CallToolRequest) (any, error) { return "ok", nil },
	)

	if err := registry.Register(tool); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := registry.Register(tool); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}
