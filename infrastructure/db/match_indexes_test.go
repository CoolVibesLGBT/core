package db

import (
	"strings"
	"testing"
)

func TestMatchIndexesCoverExactCursorAndPairLookup(t *testing.T) {
	definitions := matchIndexDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("match index count = %d; want 3", len(definitions))
	}
	byName := make(map[string]IndexDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}
	if columns := strings.Join(byName["idx_engagement_details_actor_kind_last_seen"].Columns, ","); columns != "engager_id,kind,updated_at DESC" {
		t.Fatalf("last seen columns = %q", columns)
	}
	if columns := strings.Join(byName["idx_engagement_details_actor_kind_cursor"].Columns, ","); columns != "engager_id,kind,created_at DESC,id DESC" {
		t.Fatalf("cursor columns = %q", columns)
	}
	if columns := strings.Join(byName["idx_engagement_details_pair_kind"].Columns, ","); columns != "engager_id,engagee_id,kind" {
		t.Fatalf("pair columns = %q", columns)
	}
}
