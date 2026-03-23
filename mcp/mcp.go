package mcp

import (
	"context"
	"core/constants"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type MCPServer struct {
	registry *Registry
	router   *Router
	mu       sync.RWMutex
	sessions map[string]*ConnectionState

	serverInfo                Implementation
	instructions              string
	supportedProtocolVersions []string
}

func NewMCPServer() *MCPServer {
	registry := NewRegistry()
	router := NewRouter(registry)

	return &MCPServer{
		registry: registry,
		router:   router,
		sessions: make(map[string]*ConnectionState),
		serverInfo: Implementation{
			Name:    "coolvibes-core",
			Title:   constants.APPLICATION_NAME + " MCP",
			Version: "dev",
		},
		instructions:              "Use the available tools to fetch domain data and generate text. Call initialize first, then notifications/initialized, then tools/list or tools/call.",
		supportedProtocolVersions: append([]string(nil), SupportedProtocolVersions...),
	}
}

func (r *MCPServer) Registry() *Registry {
	return r.registry
}

func (r *MCPServer) Router() *Router {
	return r.router
}

func (r *MCPServer) RegisterTool(tool Tool) {
	r.registry.Register(tool)
}

func (r *MCPServer) Call(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
	return r.router.Call(ctx, req)
}

func (r *MCPServer) ListTools() []ToolDefinition {
	return r.router.List()
}

func (r *MCPServer) SetInstructions(instructions string) {
	r.instructions = strings.TrimSpace(instructions)
}

func (r *MCPServer) SupportedProtocolVersions() []string {
	return append([]string(nil), r.supportedProtocolVersions...)
}

func (r *MCPServer) SupportsProtocolVersion(version string) bool {
	for _, supported := range r.supportedProtocolVersions {
		if supported == version {
			return true
		}
	}
	return false
}

func (r *MCPServer) NewConnection() *ConnectionState {
	return &ConnectionState{}
}

func (r *MCPServer) NewHTTPSession() (string, *ConnectionState) {
	sessionID := uuid.NewString()
	connection := r.NewConnection()

	r.mu.Lock()
	r.sessions[sessionID] = connection
	r.mu.Unlock()

	return sessionID, connection
}

func (r *MCPServer) GetHTTPSession(sessionID string) (*ConnectionState, bool) {
	r.mu.RLock()
	connection, ok := r.sessions[sessionID]
	r.mu.RUnlock()
	return connection, ok
}

func (r *MCPServer) DeleteHTTPSession(sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[sessionID]; !ok {
		return false
	}

	delete(r.sessions, sessionID)
	return true
}

func (r *MCPServer) HandleMessage(ctx context.Context, conn *ConnectionState, msg JSONRPCMessage) *JSONRPCMessage {
	if msg.JSONRPC != "" && msg.JSONRPC != JSONRPCVersion {
		if msg.HasID() {
			return NewJSONRPCErrorMessage(msg.ID, InvalidRequestCode, "jsonrpc must be 2.0", nil)
		}
		return nil
	}

	if msg.Method == "" {
		if msg.IsResponse() {
			return nil
		}
		if msg.HasID() {
			return NewJSONRPCErrorMessage(msg.ID, InvalidRequestCode, "method is required", nil)
		}
		return nil
	}

	switch msg.Method {
	case "initialize":
		return r.handleInitialize(conn, msg)
	case "notifications/initialized":
		if conn != nil && conn.ProtocolVersion != "" {
			conn.Initialized = true
		}
		return nil
	case "notifications/cancelled":
		return nil
	case "ping":
		if conn == nil || conn.ProtocolVersion == "" {
			return NewJSONRPCErrorMessage(msg.ID, InvalidRequestCode, ErrServerNotInitialized.Error(), nil)
		}
		return NewJSONRPCResult(msg.ID, map[string]any{})
	default:
		if conn == nil || conn.ProtocolVersion == "" || !conn.Initialized {
			if msg.HasID() {
				return NewJSONRPCErrorMessage(msg.ID, InvalidRequestCode, ErrServerNotInitialized.Error(), nil)
			}
			return nil
		}

		return r.handleOperation(ctx, msg)
	}
}

func (r *MCPServer) handleInitialize(conn *ConnectionState, msg JSONRPCMessage) *JSONRPCMessage {
	if !msg.HasID() {
		return nil
	}

	if conn == nil {
		return NewJSONRPCErrorMessage(msg.ID, InternalErrorCode, "connection state is required", nil)
	}

	if conn.ProtocolVersion != "" {
		return NewJSONRPCErrorMessage(msg.ID, InvalidRequestCode, "connection is already initialized", nil)
	}

	params, err := DecodeParams[InitializeRequestParams](msg.Params)
	if err != nil {
		return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "invalid initialize params", map[string]any{"details": err.Error()})
	}

	requestedVersion := strings.TrimSpace(params.ProtocolVersion)
	if requestedVersion == "" {
		return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "protocolVersion is required", nil)
	}

	if !r.SupportsProtocolVersion(requestedVersion) {
		return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "unsupported protocol version", map[string]any{
			"requested": requestedVersion,
			"supported": r.SupportedProtocolVersions(),
		})
	}

	conn.ProtocolVersion = requestedVersion
	conn.ClientInfo = params.ClientInfo
	conn.ClientCapabilities = params.Capabilities
	conn.Initialized = false

	return NewJSONRPCResult(msg.ID, InitializeResult{
		ProtocolVersion: requestedVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo:   r.serverInfo,
		Instructions: r.instructions,
	})
}

func (r *MCPServer) handleOperation(ctx context.Context, msg JSONRPCMessage) *JSONRPCMessage {
	if !msg.HasID() {
		return nil
	}

	switch msg.Method {
	case "tools/list":
		_, err := DecodeParams[ListToolsParams](msg.Params)
		if err != nil {
			return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "invalid tools/list params", map[string]any{"details": err.Error()})
		}

		return NewJSONRPCResult(msg.ID, ListToolsResult{
			Tools: r.ListTools(),
		})
	case "tools/call":
		params, err := DecodeParams[CallToolParams](msg.Params)
		if err != nil {
			return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "invalid tools/call params", map[string]any{"details": err.Error()})
		}

		toolName := strings.TrimSpace(params.Name)
		if toolName == "" {
			return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, "tool name is required", nil)
		}

		result, err := r.Call(ctx, CallToolRequest{
			Source:    "mcp",
			Tool:      toolName,
			Arguments: params.Arguments,
		})
		if err != nil {
			return NewJSONRPCErrorMessage(msg.ID, InvalidParamsCode, err.Error(), nil)
		}

		return NewJSONRPCResult(msg.ID, result)
	default:
		return NewJSONRPCErrorMessage(msg.ID, MethodNotFoundCode, "method not found", nil)
	}
}
