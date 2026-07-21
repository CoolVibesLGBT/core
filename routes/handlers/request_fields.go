package handlers

import (
	"core/models"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
)

const actionJSONFieldsLocal = "actionJSONFields"

const (
	maxReportKindLength        = 64
	maxReportDescriptionLength = 4000
)

// requestField keeps the action API compatible with query/form clients while
// also allowing native clients to send the same scalar fields as JSON.
func requestField(c fiber.Ctx, name string) string {
	if value := c.FormValue(name); value != "" {
		return value
	}

	fields := requestJSONFields(c)
	raw, ok := fields[name]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return strings.TrimSpace(string(raw))
}

func requestJSONFields(c fiber.Ctx) map[string]json.RawMessage {
	if cached := c.Locals(actionJSONFieldsLocal); cached != nil {
		if fields, ok := cached.(map[string]json.RawMessage); ok {
			return fields
		}
	}

	fields := make(map[string]json.RawMessage)
	if strings.Contains(strings.ToLower(c.Get(fiber.HeaderContentType)), "application/json") {
		_ = json.Unmarshal(c.Body(), &fields)
	}
	c.Locals(actionJSONFieldsLocal, fields)
	return fields
}

func reportKindField(c fiber.Ctx) string {
	for _, name := range []string{"report_kind_key", "kind"} {
		if value := strings.TrimSpace(requestField(c, name)); value != "" {
			return value
		}
	}
	return ""
}

// reportFields preserves the original API contract where reason could be a
// report-kind key or free text. New clients should send report_kind_key and
// description. Unknown legacy reason text is safely categorized as "other".
func reportFields(c fiber.Ctx) (kind, description string) {
	description = strings.TrimSpace(requestField(c, "description"))
	if kind = reportKindField(c); kind != "" {
		return kind, description
	}

	legacyReason := strings.TrimSpace(requestField(c, "reason"))
	if legacyReason == "" {
		return "", description
	}
	if models.IsStandardReportKind(legacyReason) {
		return legacyReason, description
	}
	if description == "" {
		description = legacyReason
	} else {
		description = legacyReason + "\n" + description
	}
	return models.ReportKindOther, description
}

func validateReportFields(kind, description string) error {
	if utf8.RuneCountInString(kind) > maxReportKindLength {
		return errors.New("report kind is too long")
	}
	if utf8.RuneCountInString(description) > maxReportDescriptionLength {
		return errors.New("report description is too long")
	}
	return nil
}
