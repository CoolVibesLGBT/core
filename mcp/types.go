package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
)

const (
	JSONRPCVersion         = "2.0"
	LatestProtocolVersion  = "2025-11-25"
	DefaultProtocolVersion = "2025-03-26"
)

var SupportedProtocolVersions = []string{
	LatestProtocolVersion,
	"2025-06-18",
	DefaultProtocolVersion,
}

type CallToolRequest struct {
	Source    string         `json:"source,omitempty"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	Content           []TextContent  `json:"content"`
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
}

type ListToolsResult struct {
	Tools      []ToolDefinition `json:"tools"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

type ToolHandler func(context.Context, CallToolRequest) (any, error)

type JSONSchema struct {
	Type                 string         `json:"type,omitempty"`
	Description          string         `json:"description,omitempty"`
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	AdditionalProperties any            `json:"additionalProperties,omitempty"`
	Items                any            `json:"items,omitempty"`
	Enum                 []string       `json:"enum,omitempty"`
}

type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   bool   `json:"openWorldHint,omitempty"`
}

type ToolDefinition struct {
	Name         string           `json:"name"`
	Title        string           `json:"title,omitempty"`
	Description  string           `json:"description,omitempty"`
	InputSchema  JSONSchema       `json:"inputSchema"`
	OutputSchema *JSONSchema      `json:"outputSchema,omitempty"`
	Annotations  *ToolAnnotations `json:"annotations,omitempty"`
}

type Tool interface {
	Definition() ToolDefinition
	Call(context.Context, CallToolRequest) (any, error)
}

type FuncTool struct {
	definition ToolDefinition
	handler    ToolHandler
}

func NewTool(definition ToolDefinition, handler ToolHandler) Tool {
	return &FuncTool{
		definition: definition,
		handler:    handler,
	}
}

func (t *FuncTool) Definition() ToolDefinition {
	return t.definition
}

func (t *FuncTool) Name() string {
	return t.definition.Name
}

func (t *FuncTool) Call(ctx context.Context, req CallToolRequest) (any, error) {
	return t.handler(ctx, req)
}

type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

type InitializeRequestParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	ClientInfo      Implementation `json:"clientInfo"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type ListToolsParams struct {
	Cursor string `json:"cursor,omitempty"`
}

type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

func (m JSONRPCMessage) HasID() bool {
	switch classifyJSONRPCID(m.ID) {
	case jsonRPCIDKindString, jsonRPCIDKindNumber:
		return true
	default:
		return false
	}
}

func (m JSONRPCMessage) HasInvalidID() bool {
	switch classifyJSONRPCID(m.ID) {
	case jsonRPCIDKindNull, jsonRPCIDKindInvalid:
		return true
	default:
		return false
	}
}

func (m JSONRPCMessage) InvalidRequestID() json.RawMessage {
	if m.HasInvalidID() {
		return json.RawMessage("null")
	}
	if m.HasID() {
		return cloneID(m.ID)
	}
	return nil
}

func (m JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.HasID()
}

func (m JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && !m.HasID()
}

func (m JSONRPCMessage) IsResponse() bool {
	return m.Method == "" && m.HasID() && (m.Result != nil || m.Error != nil)
}

type ConnectionSnapshot struct {
	ProtocolVersion    string
	Initialized        bool
	ClientInfo         Implementation
	ClientCapabilities map[string]any
}

type ConnectionState struct {
	mu sync.RWMutex

	protocolVersion    string
	initialized        bool
	clientInfo         Implementation
	clientCapabilities map[string]any
}

func (c *ConnectionState) Snapshot() ConnectionSnapshot {
	if c == nil {
		return ConnectionSnapshot{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return ConnectionSnapshot{
		ProtocolVersion:    c.protocolVersion,
		Initialized:        c.initialized,
		ClientInfo:         c.clientInfo,
		ClientCapabilities: cloneStringAnyMap(c.clientCapabilities),
	}
}

func (c *ConnectionState) ProtocolVersion() string {
	if c == nil {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.protocolVersion
}

func (c *ConnectionState) IsReady() bool {
	if c == nil {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.protocolVersion != "" && c.initialized
}

func (c *ConnectionState) TryInitialize(protocolVersion string, clientInfo Implementation, clientCapabilities map[string]any) bool {
	if c == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.protocolVersion != "" {
		return false
	}

	c.protocolVersion = protocolVersion
	c.clientInfo = clientInfo
	c.clientCapabilities = cloneStringAnyMap(clientCapabilities)
	c.initialized = false
	return true
}

func (c *ConnectionState) MarkInitialized() bool {
	if c == nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.protocolVersion == "" {
		return false
	}

	c.initialized = true
	return true
}

type jsonRPCIDKind uint8

const (
	jsonRPCIDKindMissing jsonRPCIDKind = iota
	jsonRPCIDKindString
	jsonRPCIDKindNumber
	jsonRPCIDKindNull
	jsonRPCIDKindInvalid
)

func classifyJSONRPCID(raw json.RawMessage) jsonRPCIDKind {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return jsonRPCIDKindMissing
	}

	if bytes.Equal(trimmed, []byte("null")) {
		return jsonRPCIDKindNull
	}

	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err == nil {
			return jsonRPCIDKindString
		}
		return jsonRPCIDKindInvalid
	}

	if trimmed[0] == '-' || (trimmed[0] >= '0' && trimmed[0] <= '9') {
		var value json.Number
		if err := json.Unmarshal(trimmed, &value); err == nil {
			return jsonRPCIDKindNumber
		}
	}

	return jsonRPCIDKindInvalid
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
