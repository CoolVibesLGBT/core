package docs

import (
	"core/constants"
	"encoding/json"
	"testing"
)

func TestOpenAPISerializes(t *testing.T) {
	payload, err := OpenAPIJSON()
	if err != nil {
		t.Fatalf("OpenAPIJSON() error = %v", err)
	}
	if !json.Valid(payload) {
		t.Fatal("OpenAPIJSON() returned invalid JSON")
	}

	yamlPayload, err := OpenAPIYAML()
	if err != nil {
		t.Fatalf("OpenAPIYAML() error = %v", err)
	}
	if len(yamlPayload) == 0 {
		t.Fatal("OpenAPIYAML() returned empty payload")
	}
}

func TestReportingActionsAreDocumented(t *testing.T) {
	wanted := map[string]bool{
		constants.CMD_POST_REPORT:               false,
		constants.CMD_USER_REPORT:               false,
		constants.CMD_MODERATION_REPORTS_FETCH:  false,
		constants.CMD_MODERATION_REPORT_RESOLVE: false,
		constants.CMD_MODERATION_POST_HIDE:      false,
		constants.CMD_MODERATION_POST_UNHIDE:    false,
	}
	for _, action := range ActionDocuments() {
		if _, ok := wanted[action.Action]; ok {
			wanted[action.Action] = true
		}
	}
	for action, found := range wanted {
		if !found {
			t.Fatalf("reporting action %q is not documented", action)
		}
		if !actionSupportsJSON(action) {
			t.Fatalf("reporting action %q does not advertise JSON support", action)
		}
	}
}

func TestPrivatePhotoActionsAreDocumented(t *testing.T) {
	wanted := map[string]bool{
		constants.CMD_USER_PRIVATE_PHOTOS_FETCH:           false,
		constants.CMD_USER_PRIVATE_PHOTOS_UPLOAD:          false,
		constants.CMD_USER_PRIVATE_PHOTOS_DELETE:          false,
		constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REQUEST:  false,
		constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REQUESTS: false,
		constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_RESPOND:  false,
		constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REVOKE:   false,
	}
	for _, action := range ActionDocuments() {
		if _, ok := wanted[action.Action]; ok {
			wanted[action.Action] = true
		}
	}
	for action, found := range wanted {
		if !found {
			t.Fatalf("private photo action %q is not documented", action)
		}
	}
}

func TestActionDocumentsAreUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for _, action := range ActionDocuments() {
		if action.Action == "" {
			t.Fatal("action document has empty action")
		}
		if action.Tag == "" {
			t.Fatalf("action %q has empty tag", action.Action)
		}
		if _, ok := seen[action.Action]; ok {
			t.Fatalf("duplicate action document for %q", action.Action)
		}
		seen[action.Action] = struct{}{}
	}
}

func TestPostTipDocumentsHeaderAndOptionalFormIdempotencyFallback(t *testing.T) {
	var tip *ActionDoc
	for _, action := range ActionDocuments() {
		if action.Action == constants.CMD_POST_TIP {
			copy := action
			tip = &copy
			break
		}
	}
	if tip == nil {
		t.Fatal("post tip action is not documented")
	}

	var headerFound, formFound bool
	for _, param := range tip.Params {
		switch {
		case param.In == "header" && param.Name == "Idempotency-Key":
			headerFound = true
			if param.Required {
				t.Fatal("Idempotency-Key header cannot be individually required when the form fallback is valid")
			}
		case param.In == "form" && param.Name == "idempotency_key":
			formFound = true
			if param.Required {
				t.Fatal("idempotency_key form fallback cannot be individually required when the header is valid")
			}
		}
	}
	if !headerFound || !formFound {
		t.Fatalf("post tip idempotency alternatives: header=%v form=%v", headerFound, formFound)
	}
}
