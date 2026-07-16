package shared

import "core/mcp"

func BaseFilterSchema(required ...string) mcp.JSONSchema {
	return mcp.JSONSchema{
		Type: "object",
		Properties: map[string]any{
			"user_id":   SchemaString("User UUID or numeric id."),
			"post_id":   SchemaString("Post UUID or numeric id."),
			"limit":     SchemaInteger("Pagination limit."),
			"cursor":    SchemaInteger("Pagination cursor."),
			"search":    SchemaString("Search term."),
			"category":  SchemaString("Category slug or name."),
			"name":      SchemaString("Name filter."),
			"city":      SchemaString("City filter."),
			"country":   SchemaString("Country filter."),
			"latitude":  SchemaNumber("Latitude."),
			"longitude": SchemaNumber("Longitude."),
			"distance":  SchemaNumber("Distance in kilometers."),
		},
		Required:             required,
		AdditionalProperties: false,
	}
}

func ReadOnlyToolAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:          title,
		ReadOnlyHint:   true,
		IdempotentHint: true,
	}
}

func SchemaString(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func SchemaNumber(description string) map[string]any {
	return map[string]any{
		"type":        "number",
		"description": description,
	}
}

func SchemaInteger(description string) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
	}
}
