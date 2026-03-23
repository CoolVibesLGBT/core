package handlers

import (
	"context"
	"core/mcp"
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
		case fiber.MethodPost:
			return handleMCPPost(c, server)
		case fiber.MethodDelete:
			return handleMCPDelete(c, server)
		default:
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
	}
}

func handleMCPPost(c fiber.Ctx, server *mcp.MCPServer) error {
	var message mcp.JSONRPCMessage
	if err := c.Bind().JSON(&message); err != nil {
		return writeJSONRPC(c, fiber.StatusBadRequest, mcp.NewJSONRPCErrorMessage(nil, mcp.ParseErrorCode, "parse error", map[string]any{"details": err.Error()}))
	}

	if message.JSONRPC == "" {
		message.JSONRPC = mcp.JSONRPCVersion
	}

	if message.IsResponse() {
		return c.SendStatus(fiber.StatusAccepted)
	}

	sessionID := strings.TrimSpace(c.Get("MCP-Session-Id"))
	protocolHeader := strings.TrimSpace(c.Get("MCP-Protocol-Version"))

	var (
		connection   *mcp.ConnectionState
		newSessionID string
		ok           bool
	)

	if message.Method == "initialize" {
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

	response := server.HandleMessage(context.Background(), connection, message)
	if response == nil {
		if message.IsNotification() {
			return c.SendStatus(fiber.StatusAccepted)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	if newSessionID != "" {
		if response.Error != nil {
			server.DeleteHTTPSession(newSessionID)
		} else {
			c.Set("MCP-Session-Id", newSessionID)
		}
	}

	if connection != nil && connection.ProtocolVersion != "" {
		c.Set("MCP-Protocol-Version", connection.ProtocolVersion)
	}

	return writeJSONRPC(c, fiber.StatusOK, response)
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
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Status(status).JSON(message)
}

func validateMCPProtocolHeader(server *mcp.MCPServer, connection *mcp.ConnectionState, protocolHeader string) error {
	if protocolHeader == "" {
		if connection != nil && connection.ProtocolVersion != "" {
			return nil
		}
		return nil
	}

	if !server.SupportsProtocolVersion(protocolHeader) {
		return fmt.Errorf("unsupported MCP-Protocol-Version: %s", protocolHeader)
	}

	if connection != nil && connection.ProtocolVersion != "" && connection.ProtocolVersion != protocolHeader {
		return fmt.Errorf("protocol version mismatch: expected %s, got %s", connection.ProtocolVersion, protocolHeader)
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
