package handlers

import (
	"bytes"
	"core/mcp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func HandleMCPTransport(server *mcp.MCPServer) fiber.Handler {
	allowedOrigins := collectMCPAllowedOrigins()

	return func(c fiber.Ctx) error {
		if !isAllowedMCPOrigin(c, allowedOrigins) {
			return c.SendStatus(fiber.StatusForbidden)
		}

		switch c.Method() {
		case fiber.MethodGet:
			c.Set(fiber.HeaderAllow, strings.Join([]string{fiber.MethodPost, fiber.MethodDelete}, ", "))
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		case fiber.MethodPost:
			return handleMCPPost(c, server)
		case fiber.MethodDelete:
			return handleMCPDelete(c, server)
		default:
			c.Set(fiber.HeaderAllow, strings.Join([]string{fiber.MethodPost, fiber.MethodDelete}, ", "))
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
	}
}

func handleMCPPost(c fiber.Ctx, server *mcp.MCPServer) error {
	rawBody := bytes.TrimSpace(c.Body())
	messages, isBatch, err := mcp.DecodeWireMessages(rawBody)
	if err != nil {
		code := mcp.ParseErrorCode
		message := "parse error"
		if errors.Is(err, mcp.ErrEmptyBatch) {
			code = mcp.InvalidRequestCode
			message = "invalid request"
		}
		return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(nil, code, message, map[string]any{"details": err.Error()}))
	}

	message := messages[0]
	for index := range messages {
		if messages[index].JSONRPC == "" {
			messages[index].JSONRPC = mcp.JSONRPCVersion
		}
	}

	if !isBatch && message.IsResponse() {
		return c.SendStatus(fiber.StatusAccepted)
	}

	sessionID := strings.TrimSpace(c.Get("MCP-Session-Id"))
	protocolHeader := strings.TrimSpace(c.Get("MCP-Protocol-Version"))

	var (
		connection   *mcp.ConnectionState
		newSessionID string
		ok           bool
	)

	if isBatch {
		if sessionID == "" {
			return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(nil, mcp.InvalidRequestCode, "MCP-Session-Id header is required", nil))
		}

		connection, ok = server.GetHTTPSession(sessionID)
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}
		state := connection.Snapshot()
		if !state.Initialized || !server.SupportsBatch(state.ProtocolVersion) {
			return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(nil, mcp.InvalidRequestCode, "JSON-RPC batches are only supported for initialized sessions using protocol version 2025-03-26", nil))
		}
	} else if message.Method == "initialize" {
		if sessionID != "" {
			return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(message.ID, mcp.InvalidRequestCode, "initialize must not include MCP-Session-Id", nil))
		}

		newSessionID, connection = server.NewHTTPSession()
	} else {
		if sessionID == "" {
			return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(message.ID, mcp.InvalidRequestCode, "MCP-Session-Id header is required", nil))
		}

		connection, ok = server.GetHTTPSession(sessionID)
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}
	}

	if err := validateMCPProtocolHeader(server, connection, protocolHeader); err != nil {
		if newSessionID != "" {
			server.DeleteHTTPSession(newSessionID)
		}
		return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(message.ID, mcp.InvalidRequestCode, err.Error(), nil))
	}

	responses := server.HandleMessages(c.Context(), connection, messages)
	if len(responses) == 0 {
		if isBatch || message.IsNotification() {
			return c.SendStatus(fiber.StatusAccepted)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	if newSessionID != "" {
		if responses[0].Error != nil {
			server.DeleteHTTPSession(newSessionID)
		} else {
			c.Set("MCP-Session-Id", newSessionID)
		}
	}

	if connection != nil {
		if protocolVersion := connection.ProtocolVersion(); protocolVersion != "" {
			c.Set("MCP-Protocol-Version", protocolVersion)
		}
	}

	return writeJSONRPCMessages(c, fiber.StatusOK, responses, isBatch)
}

func handleMCPDelete(c fiber.Ctx, server *mcp.MCPServer) error {
	sessionID := strings.TrimSpace(c.Get("MCP-Session-Id"))
	if sessionID == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	if !server.DeleteHTTPSession(sessionID) {
		return c.SendStatus(fiber.StatusNotFound)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func writeJSONRPC(c fiber.Ctx, status int, message *mcp.JSONRPCMessage) error {
	return writeJSONRPCMessages(c, status, []*mcp.JSONRPCMessage{message}, false)
}

func writeJSONRPCMessages(c fiber.Ctx, status int, messages []*mcp.JSONRPCMessage, isBatch bool) error {
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	if isBatch {
		return c.Status(status).JSON(messages)
	}
	return c.Status(status).JSON(messages[0])
}

func validateMCPProtocolHeader(server *mcp.MCPServer, connection *mcp.ConnectionState, protocolHeader string) error {
	if protocolHeader == "" {
		return nil
	}

	if connection == nil {
		return nil
	}

	expected := connection.ProtocolVersion()
	if expected == "" {
		return nil
	}

	if !server.SupportsProtocolVersion(protocolHeader) {
		return fmt.Errorf("unsupported MCP-Protocol-Version: %s", protocolHeader)
	}

	if expected != protocolHeader {
		return fmt.Errorf("protocol version mismatch: expected %s, got %s", expected, protocolHeader)
	}

	return nil
}

func collectMCPAllowedOrigins() map[string]struct{} {
	allowed := make(map[string]struct{})

	for _, raw := range []string{
		os.Getenv("APP_BASE_URL"),
		os.Getenv("SITE_URI"),
	} {
		if normalized, ok := normalizeOrigin(raw); ok {
			allowed[normalized] = struct{}{}
		}
	}

	for _, raw := range strings.Split(os.Getenv("MCP_ALLOWED_ORIGINS"), ",") {
		if normalized, ok := normalizeOrigin(raw); ok {
			allowed[normalized] = struct{}{}
		}
	}

	return allowed
}

func isAllowedMCPOrigin(c fiber.Ctx, allowed map[string]struct{}) bool {
	origin := strings.TrimSpace(c.Get("Origin"))
	if origin == "" {
		return true
	}

	normalizedOrigin, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}

	if baseURL, ok := normalizeOrigin(c.BaseURL()); ok && normalizedOrigin == baseURL {
		return true
	}

	_, ok = allowed[normalizedOrigin]
	return ok
}

func normalizeOrigin(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}
