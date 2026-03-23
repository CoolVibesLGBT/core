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
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("decode body: %v\npayload=%s", err, string(payload))
	}
}
