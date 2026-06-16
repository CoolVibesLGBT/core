package docs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type AuthMode string

const (
	AuthNone     AuthMode = "none"
	AuthOptional AuthMode = "optional"
	AuthRequired AuthMode = "required"
)

type Param struct {
	Name        string
	In          string
	Type        string
	Format      string
	Description string
	Required    bool
	Array       bool
	Enum        []string
	Default     any
	Example     any
}

type ActionDoc struct {
	Action        string
	Tag           string
	Summary       string
	Description   string
	Auth          AuthMode
	Params        []Param
	UseFilters    bool
	UseContent    bool
	SuccessStatus int
	Deprecated    bool
}

func OpenAPISpec() map[string]any {
	actions := ActionDocuments()
	paths := map[string]any{
		"/api":    packetPathItem(actions, "Main action packet endpoint. Send action and parameters in the request body."),
		"/packet": packetPathItem(actions, "Compatibility action packet endpoint. Send action and parameters in the request body."),
		"/test":   packetPathItem(actions, "Development action packet endpoint. Send action and parameters in the request body."),
		"/mcp": map[string]any{
			"post": operation(
				"MCP",
				"Run MCP JSON-RPC over HTTP",
				"Accepts one JSON-RPC message or a JSON-RPC batch. Non-initialize calls require the MCP-Session-Id header.",
				[]Param{
					headerParam("MCP-Session-Id", "Session id returned by initialize. Required after initialization.", false),
					headerParam("MCP-Protocol-Version", "Optional MCP protocol version guard.", false),
				},
				jsonBody("MCPRequest", map[string]any{
					"oneOf": []any{
						map[string]any{"$ref": "#/components/schemas/JSONRPCMessage"},
						map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/JSONRPCMessage"},
						},
					},
				}),
				AuthNone,
				false,
				200,
			),
			"delete": operation(
				"MCP",
				"Close MCP HTTP session",
				"Deletes an existing MCP HTTP session by MCP-Session-Id.",
				[]Param{headerParam("MCP-Session-Id", "Session id to close.", true)},
				nil,
				AuthNone,
				false,
				204,
			),
		},
		"/webhook/bot/telegram/": map[string]any{
			"post": operation(
				"Webhooks",
				"Telegram bot webhook",
				"Receives Telegram updates. Requires X-Telegram-Bot-Api-Secret-Token to match TELEGRAM_WEBHOOK_SECRET.",
				[]Param{headerParam("X-Telegram-Bot-Api-Secret-Token", "Telegram webhook secret token.", true)},
				jsonBody("TelegramUpdate", map[string]any{"type": "object", "additionalProperties": true}),
				AuthNone,
				false,
				200,
			),
		},
		"/webhook/gateway/stripe/thin": map[string]any{
			"post": operation("Webhooks", "Stripe thin webhook", "Receives Stripe thin event payloads.", nil, jsonBody("StripeThinWebhook", map[string]any{"type": "object", "additionalProperties": true}), AuthNone, false, 200),
		},
		"/webhook/gateway/stripe/snapshot": map[string]any{
			"post": operation("Webhooks", "Stripe snapshot webhook", "Receives Stripe snapshot event payloads.", nil, jsonBody("StripeSnapshotWebhook", map[string]any{"type": "object", "additionalProperties": true}), AuthNone, false, 200),
		},
		"/sitemap.xml":            sitemapOperation("Sitemap index"),
		"/sitemap-posts.xml":      sitemapOperation("Post sitemap"),
		"/sitemap-news.xml":       sitemapOperation("News sitemap"),
		"/sitemap-categories.xml": sitemapOperation("Category sitemap"),
		"/sitemap-images.xml":     sitemapOperation("Image sitemap"),
		"/sitemap-videos.xml":     sitemapOperation("Video sitemap"),
		"/swagger": map[string]any{
			"get": operation("Swagger", "Swagger UI", "Serves Swagger UI for this OpenAPI document.", nil, nil, AuthNone, false, 200),
		},
		"/swagger/openapi.yaml": map[string]any{
			"get": operation("Swagger", "OpenAPI YAML", "Serves the OpenAPI document as YAML.", nil, nil, AuthNone, false, 200),
		},
		"/swagger/openapi.json": map[string]any{
			"get": operation("Swagger", "OpenAPI JSON", "Serves the OpenAPI document as JSON.", nil, nil, AuthNone, false, 200),
		},
		"/docs": map[string]any{
			"get": operation("Swagger", "Swagger UI alias", "Serves Swagger UI for this OpenAPI document.", nil, nil, AuthNone, false, 200),
		},
		"/docs/openapi.yaml": map[string]any{
			"get": operation("Swagger", "OpenAPI YAML alias", "Serves the OpenAPI document as YAML.", nil, nil, AuthNone, false, 200),
		},
		"/docs/openapi.json": map[string]any{
			"get": operation("Swagger", "OpenAPI JSON alias", "Serves the OpenAPI document as JSON.", nil, nil, AuthNone, false, 200),
		},
	}

	for _, action := range actions {
		paths["/api/actions/"+action.Action] = map[string]any{
			"post": actionOperation(action, false),
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "CoolVibes Core API",
			"description": "Action-based Fiber API. Existing clients may call /api, /packet, /test, or the documented /api/actions/{action} aliases.",
			"version":     "1.0.0",
		},
		"servers": []any{
			map[string]any{"url": "/", "description": "Current host"},
		},
		"tags":  tagObjects(actions),
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
					"description":  "Use Authorization: Bearer <jwt>.",
				},
			},
			"schemas": componentSchemas(actions),
			"responses": map[string]any{
				"Success": map[string]any{
					"description": "Successful response.",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"$ref": "#/components/schemas/SuccessResponse"},
						},
					},
				},
				"Error": map[string]any{
					"description": "Error response.",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{"$ref": "#/components/schemas/ErrorResponse"},
						},
					},
				},
			},
		},
	}
}

func OpenAPIJSON() ([]byte, error) {
	return json.MarshalIndent(OpenAPISpec(), "", "  ")
}

func OpenAPIYAML() ([]byte, error) {
	return yaml.Marshal(OpenAPISpec())
}

func actionOperation(action ActionDoc, includeAction bool) map[string]any {
	params := actionParams(action)
	return operation(
		action.Tag,
		action.Summary,
		action.Description,
		queryParams(params),
		formBody(actionSchemaName(action, includeAction), schemaForParams(params, action.Action, includeAction)),
		action.Auth,
		action.Deprecated,
		successStatus(action.SuccessStatus),
	)
}

func packetPathItem(actions []ActionDoc, description string) map[string]any {
	oneOf := make([]any, 0, len(actions))
	for _, action := range actions {
		oneOf = append(oneOf, map[string]any{"$ref": "#/components/schemas/" + actionSchemaName(action, true)})
	}

	body := map[string]any{
		"required": true,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": map[string]any{"oneOf": oneOf},
			},
			"application/x-www-form-urlencoded": map[string]any{
				"schema": map[string]any{"oneOf": oneOf},
			},
			"application/json": map[string]any{
				"schema": map[string]any{
					"type":        "object",
					"description": "JSON packet support is currently used only to read the action field; action handlers generally read parameters from form data.",
					"required":    []string{"action"},
					"properties": map[string]any{
						"action": map[string]any{
							"type": "string",
							"enum": actionNames(actions),
						},
					},
					"additionalProperties": true,
				},
			},
		},
	}

	return map[string]any{
		"get":  operation("Packet", "Dispatch action by query string", description+" For GET, pass action in the query string; most action parameters are still handler-specific.", []Param{queryParam("action", "Registered action command.", "string", true)}, nil, AuthOptional, false, 200),
		"post": operation("Packet", "Dispatch action by form packet", description, nil, body, AuthOptional, false, 200),
	}
}

func operation(tag, summary, description string, params []Param, requestBody map[string]any, auth AuthMode, deprecated bool, status int) map[string]any {
	op := map[string]any{
		"tags":        []string{tag},
		"summary":     summary,
		"description": description,
		"responses": map[string]any{
			fmt.Sprintf("%d", status): map[string]any{"$ref": "#/components/responses/Success"},
			"400":                     map[string]any{"$ref": "#/components/responses/Error"},
			"401":                     map[string]any{"$ref": "#/components/responses/Error"},
			"500":                     map[string]any{"$ref": "#/components/responses/Error"},
		},
	}
	if deprecated {
		op["deprecated"] = true
	}
	if len(params) > 0 {
		op["parameters"] = openAPIParams(params)
	}
	if requestBody != nil {
		op["requestBody"] = requestBody
	}
	switch auth {
	case AuthRequired:
		op["security"] = []any{map[string]any{"bearerAuth": []any{}}}
	case AuthOptional:
		op["security"] = []any{map[string]any{"bearerAuth": []any{}}, map[string]any{}}
	}
	return op
}

func componentSchemas(actions []ActionDoc) map[string]any {
	schemas := map[string]any{
		"SuccessResponse": map[string]any{
			"type":     "object",
			"required": []string{"success", "data"},
			"properties": map[string]any{
				"success": map[string]any{"type": "boolean", "example": true},
				"data":    map[string]any{"description": "Endpoint-specific payload."},
				"message": map[string]any{"type": "string"},
			},
		},
		"ErrorResponse": map[string]any{
			"type":     "object",
			"required": []string{"success", "code", "message"},
			"properties": map[string]any{
				"success": map[string]any{"type": "boolean", "example": false},
				"code":    map[string]any{"type": "string", "example": "INVALID_INPUT"},
				"message": map[string]any{"type": "string"},
			},
		},
		"Cursor": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prev":     map[string]any{"type": "string", "nullable": true},
				"next":     map[string]any{"type": "string", "nullable": true},
				"distance": map[string]any{"type": "number", "format": "double", "nullable": true},
			},
		},
		"JSONRPCMessage": map[string]any{
			"type":                 "object",
			"additionalProperties": true,
			"properties": map[string]any{
				"jsonrpc": map[string]any{"type": "string", "example": "2.0"},
				"id":      map[string]any{"nullable": true},
				"method":  map[string]any{"type": "string"},
				"params":  map[string]any{"type": "object", "additionalProperties": true},
				"result":  map[string]any{},
				"error":   map[string]any{"type": "object", "additionalProperties": true},
			},
		},
	}
	for _, action := range actions {
		schemas[actionSchemaName(action, false)] = schemaForParams(actionParams(action), action.Action, false)
		schemas[actionSchemaName(action, true)] = schemaForParams(actionParams(action), action.Action, true)
	}
	return schemas
}

func schemaForParams(params []Param, action string, includeAction bool) map[string]any {
	properties := map[string]any{}
	requiredSet := map[string]struct{}{}
	if includeAction {
		properties["action"] = map[string]any{
			"type":        "string",
			"enum":        []string{action},
			"description": "Action command dispatched by /api, /packet, or /test.",
		}
		requiredSet["action"] = struct{}{}
	}
	for _, param := range params {
		if param.In != "" && param.In != "form" {
			continue
		}
		properties[param.Name] = propertySchema(param)
		if param.Required {
			requiredSet[param.Name] = struct{}{}
		}
	}
	required := make([]string, 0, len(requiredSet))
	for name := range requiredSet {
		required = append(required, name)
	}
	sort.Strings(required)

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func formBody(name string, schema map[string]any) map[string]any {
	return map[string]any{
		"required": len(schema["properties"].(map[string]any)) > 0,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": schema,
			},
			"application/x-www-form-urlencoded": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/" + name},
			},
		},
	}
}

func jsonBody(name string, schema map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
		"description": name,
	}
}

func propertySchema(param Param) map[string]any {
	if param.Array {
		item := map[string]any{"type": param.Type}
		if param.Format != "" {
			item["format"] = param.Format
		}
		if len(param.Enum) > 0 {
			item["enum"] = param.Enum
		}
		schema := map[string]any{
			"type":        "array",
			"items":       item,
			"description": param.Description,
		}
		if param.Example != nil {
			schema["example"] = param.Example
		}
		return schema
	}

	schema := map[string]any{
		"type":        param.Type,
		"description": param.Description,
	}
	if param.Format != "" {
		schema["format"] = param.Format
	}
	if len(param.Enum) > 0 {
		schema["enum"] = param.Enum
	}
	if param.Default != nil {
		schema["default"] = param.Default
	}
	if param.Example != nil {
		schema["example"] = param.Example
	}
	return schema
}

func openAPIParams(params []Param) []any {
	result := make([]any, 0, len(params))
	for _, param := range params {
		in := param.In
		if in == "" || in == "form" {
			continue
		}
		result = append(result, map[string]any{
			"name":        param.Name,
			"in":          in,
			"required":    param.Required,
			"description": param.Description,
			"schema":      propertySchema(param),
		})
	}
	return result
}

func queryParams(params []Param) []Param {
	result := make([]Param, 0)
	for _, param := range params {
		if param.In == "query" || param.In == "header" {
			result = append(result, param)
		}
	}
	return result
}

func actionParams(action ActionDoc) []Param {
	params := make([]Param, 0, len(action.Params)+len(commonFilterParams())+len(contentablePostParams()))
	if action.UseContent {
		params = append(params, contentablePostParams()...)
	}
	if action.UseFilters {
		params = append(params, commonFilterParams()...)
	}
	params = append(params, action.Params...)
	return dedupeParams(params)
}

func dedupeParams(params []Param) []Param {
	result := make([]Param, 0, len(params))
	index := map[string]int{}
	for _, param := range params {
		key := param.In + ":" + param.Name
		if existing, ok := index[key]; ok {
			if param.Required {
				result[existing].Required = true
			}
			if param.Description != "" {
				result[existing].Description = param.Description
			}
			continue
		}
		index[key] = len(result)
		result = append(result, param)
	}
	return result
}

func actionNames(actions []ActionDoc) []string {
	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, action.Action)
	}
	sort.Strings(names)
	return names
}

func tagObjects(actions []ActionDoc) []any {
	seen := map[string]struct{}{
		"Packet":   {},
		"MCP":      {},
		"Webhooks": {},
		"Sitemaps": {},
		"Swagger":  {},
	}
	names := []string{"Packet", "MCP", "Webhooks", "Sitemaps", "Swagger"}
	for _, action := range actions {
		if _, ok := seen[action.Tag]; ok {
			continue
		}
		seen[action.Tag] = struct{}{}
		names = append(names, action.Tag)
	}
	sort.Strings(names)
	result := make([]any, 0, len(names))
	for _, name := range names {
		result = append(result, map[string]any{"name": name})
	}
	return result
}

func sitemapOperation(summary string) map[string]any {
	return map[string]any{
		"get": operation("Sitemaps", summary, "Returns XML sitemap content.", nil, nil, AuthNone, false, 200),
	}
}

func actionSchemaName(action ActionDoc, packet bool) string {
	suffix := "ActionRequest"
	if packet {
		suffix = "PacketRequest"
	}
	return sanitizeName(action.Action) + suffix
}

func sanitizeName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ':'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func successStatus(status int) int {
	if status == 0 {
		return 200
	}
	return status
}

func stringParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "string", Description: description, Required: required}
}

func uuidParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "string", Format: "uuid", Description: description, Required: required}
}

func intParam(name, description string, required bool, def any) Param {
	return Param{Name: name, In: "form", Type: "integer", Format: "int64", Description: description, Required: required, Default: def}
}

func numberParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "number", Format: "double", Description: description, Required: required}
}

func boolParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "boolean", Description: description, Required: required}
}

func enumParam(name, description string, required bool, values ...string) Param {
	return Param{Name: name, In: "form", Type: "string", Description: description, Required: required, Enum: values}
}

func fileParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "string", Format: "binary", Description: description, Required: required}
}

func fileArrayParam(name, description string, required bool) Param {
	return Param{Name: name, In: "form", Type: "string", Format: "binary", Description: description, Required: required, Array: true}
}

func queryParam(name, description, typ string, required bool) Param {
	return Param{Name: name, In: "query", Type: typ, Description: description, Required: required}
}

func headerParam(name, description string, required bool) Param {
	return Param{Name: name, In: "header", Type: "string", Description: description, Required: required}
}
