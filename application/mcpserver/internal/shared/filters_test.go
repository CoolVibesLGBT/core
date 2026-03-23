package shared

import (
	"context"
	"testing"
)

func TestBuildFilterPreservesExplicitSmallLimit(t *testing.T) {
	filter, err := BuildFilter(context.Background(), map[string]any{
		"limit": 1,
	})
	if err != nil {
		t.Fatalf("BuildFilter() error = %v", err)
	}

	if filter.Limit != 1 {
		t.Fatalf("expected limit 1, got %d", filter.Limit)
	}
}
