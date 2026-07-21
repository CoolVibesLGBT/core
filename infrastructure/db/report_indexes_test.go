package db

import (
	"strings"
	"testing"
)

func TestReportIndexDefinitionsCoverQueueAndTargetUpdates(t *testing.T) {
	definitions := reportIndexDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("report index count = %d, want 3", len(definitions))
	}
	byName := make(map[string]IndexDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}
	queue := byName["idx_reports_status_queue"]
	if got := strings.Join(queue.Columns, ","); got != "status,created_at DESC,id DESC" {
		t.Fatalf("queue index columns = %q", got)
	}
	typedQueue := byName["idx_reports_type_status_queue"]
	if got := strings.Join(typedQueue.Columns, ","); got != "contentable_type,status,created_at DESC,id DESC" {
		t.Fatalf("typed queue index columns = %q", got)
	}
	target := byName["idx_reports_target_status"]
	if got := strings.Join(target.Columns, ","); got != "contentable_type,contentable_id,status" {
		t.Fatalf("target index columns = %q", got)
	}
}
