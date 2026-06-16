package docs

import (
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
