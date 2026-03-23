package handlers

import (
	"context"
	"core/mcp"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestMCPHTTPTransportLifecycle(t *testing.T) {
	server := mcp.NewMCPServer()
	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        "test.echo",
			Description: "Echo a value.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
			},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			return map[string]any{"echo": req.Arguments["value"]}, nil
		},
	))

	app := fiber.New()
	app.All("/mcp", HandleMCPTransport(server))

	initializePayload := `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}
	}`

	initializeResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", initializePayload, nil)
	if initializeResponse.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", initializeResponse.StatusCode)
	}

	sessionID := initializeResponse.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		t.Fatal("expected MCP-Session-Id header")
	}
	if protocolVersion := initializeResponse.Header.Get("MCP-Protocol-Version"); protocolVersion != mcp.LatestProtocolVersion {
		t.Fatalf("expected negotiated protocol header %q, got %q", mcp.LatestProtocolVersion, protocolVersion)
	}

	initializedResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"method":"notifications/initialized"
	}`, map[string]string{
		"MCP-Session-Id":       sessionID,
		"MCP-Protocol-Version": "2025-11-25",
	})
	if initializedResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("initialized notification status = %d", initializedResponse.StatusCode)
	}

	toolsListResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/list",
		"params":{}
	}`, map[string]string{
		"MCP-Session-Id":       sessionID,
		"MCP-Protocol-Version": "2025-11-25",
	})
	if toolsListResponse.StatusCode != http.StatusOK {
		t.Fatalf("tools/list status = %d", toolsListResponse.StatusCode)
	}

	var toolsListBody struct {
		Result struct {
			Tools []mcp.ToolDefinition `json:"tools"`
		} `json:"result"`
	}
	decodeJSONBody(t, toolsListResponse, &toolsListBody)
	if len(toolsListBody.Result.Tools) != 1 || toolsListBody.Result.Tools[0].Name != "test.echo" {
		t.Fatalf("unexpected tools/list response: %#v", toolsListBody.Result.Tools)
	}

	toolsCallResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{"name":"test.echo","arguments":{"value":"hello"}}
	}`, map[string]string{
		"MCP-Session-Id":       sessionID,
		"MCP-Protocol-Version": "2025-11-25",
	})
	if toolsCallResponse.StatusCode != http.StatusOK {
		t.Fatalf("tools/call status = %d", toolsCallResponse.StatusCode)
	}

	var toolsCallBody struct {
		Result mcp.CallToolResult `json:"result"`
	}
	decodeJSONBody(t, toolsCallResponse, &toolsCallBody)
	if toolsCallBody.Result.StructuredContent["echo"] != "hello" {
		t.Fatalf("unexpected tools/call structured content: %#v", toolsCallBody.Result.StructuredContent)
	}

	deleteResponse := performMCPRequest(t, app, http.MethodDelete, "/mcp", "", map[string]string{
		"MCP-Session-Id": sessionID,
	})
	if deleteResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", deleteResponse.StatusCode)
	}
}

func TestMCPHTTPTransportNegotiatesUnsupportedProtocolVersion(t *testing.T) {
	server := mcp.NewMCPServer()

	app := fiber.New()
	app.All("/mcp", HandleMCPTransport(server))

	initializeResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2099-01-01",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}
	}`, map[string]string{
		"MCP-Protocol-Version": "2099-01-01",
	})
	if initializeResponse.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", initializeResponse.StatusCode)
	}

	var body struct {
		Result mcp.InitializeResult `json:"result"`
	}
	decodeJSONBody(t, initializeResponse, &body)
	if body.Result.ProtocolVersion != mcp.LatestProtocolVersion {
		t.Fatalf("expected negotiated protocol version %q, got %q", mcp.LatestProtocolVersion, body.Result.ProtocolVersion)
	}
	if protocolVersion := initializeResponse.Header.Get("MCP-Protocol-Version"); protocolVersion != mcp.LatestProtocolVersion {
		t.Fatalf("expected negotiated protocol header %q, got %q", mcp.LatestProtocolVersion, protocolVersion)
	}
	if sessionID := initializeResponse.Header.Get("MCP-Session-Id"); sessionID == "" {
		t.Fatal("expected MCP-Session-Id header")
	}
}

func TestMCPHTTPTransportRejectsNullID(t *testing.T) {
	server := mcp.NewMCPServer()

	app := fiber.New()
	app.All("/mcp", HandleMCPTransport(server))

	response := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"id":null,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-11-25",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}
	}`, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status = %d", response.StatusCode)
	}
	if sessionID := response.Header.Get("MCP-Session-Id"); sessionID != "" {
		t.Fatalf("expected no session header, got %q", sessionID)
	}

	var body mcp.JSONRPCMessage
	decodeJSONBody(t, response, &body)
	if body.Error == nil {
		t.Fatalf("expected invalid request error, got %#v", body)
	}
	if body.Error.Code != mcp.InvalidRequestCode {
		t.Fatalf("expected invalid request code, got %d", body.Error.Code)
	}
	if string(body.ID) != "null" {
		t.Fatalf("expected error id null, got %q", string(body.ID))
	}
}

func TestMCPHTTPTransportSupportsBatchForProtocol20250326(t *testing.T) {
	server := mcp.NewMCPServer()
	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        "test.echo",
			Description: "Echo a value.",
			InputSchema: mcp.JSONSchema{Type: "object"},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			return map[string]any{"echo": req.Arguments["value"]}, nil
		},
	))

	app := fiber.New()
	app.All("/mcp", HandleMCPTransport(server))

	initializeResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"id":1,
		"method":"initialize",
		"params":{
			"protocolVersion":"2025-03-26",
			"capabilities":{},
			"clientInfo":{"name":"test-client","version":"1.0.0"}
		}
	}`, nil)
	if initializeResponse.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", initializeResponse.StatusCode)
	}

	sessionID := initializeResponse.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		t.Fatal("expected MCP-Session-Id header")
	}

	initializedResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `{
		"jsonrpc":"2.0",
		"method":"notifications/initialized"
	}`, map[string]string{
		"MCP-Session-Id":       sessionID,
		"MCP-Protocol-Version": "2025-03-26",
	})
	if initializedResponse.StatusCode != http.StatusAccepted {
		t.Fatalf("initialized notification status = %d", initializedResponse.StatusCode)
	}

	batchResponse := performMCPRequest(t, app, http.MethodPost, "/mcp", `[
		{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}},
		{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test.echo","arguments":{"value":"hello"}}}
	]`, map[string]string{
		"MCP-Session-Id":       sessionID,
		"MCP-Protocol-Version": "2025-03-26",
	})
	if batchResponse.StatusCode != http.StatusOK {
		t.Fatalf("batch status = %d", batchResponse.StatusCode)
	}

	var body []mcp.JSONRPCMessage
	decodeJSONBody(t, batchResponse, &body)
	if len(body) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(body))
	}
	if body[0].Error != nil {
		t.Fatalf("tools/list error: %#v", body[0].Error)
	}
	if body[1].Error != nil {
		t.Fatalf("tools/call error: %#v", body[1].Error)
	}
}

func TestMCPHTTPTransportGetReturnsAllowHeader(t *testing.T) {
	server := mcp.NewMCPServer()

	app := fiber.New()
	app.All("/mcp", HandleMCPTransport(server))

	response := performMCPRequest(t, app, http.MethodGet, "/mcp", "", nil)
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d", response.StatusCode)
	}

	if allow := response.Header.Get("Allow"); allow != "POST, DELETE" {
		t.Fatalf("unexpected Allow header = %q", allow)
	}
}

func performMCPRequest(t *testing.T, app *fiber.App, method string, path string, body string, headers map[string]string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	return resp
}

func decodeJSONBody(t *testing.T, response *http.Response, out any) {
	t.Helper()
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close body: %v", err)
		}
	})

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("decode body: %v\npayload=%s", err, string(payload))
	}
}
