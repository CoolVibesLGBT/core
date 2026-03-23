package mcp

import (
	"context"
	"encoding/json"
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
	"2024-11-05",
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
	return len(m.ID) > 0
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

type ConnectionState struct {
	ProtocolVersion    string
	Initialized        bool
	ClientInfo         Implementation
	ClientCapabilities map[string]any
}
