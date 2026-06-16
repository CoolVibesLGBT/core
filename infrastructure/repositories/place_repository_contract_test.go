package repositories

import (
	"core/models/post"
	"core/types"
	"strings"
	"testing"
)

func TestNearByPlacesQueryWithoutCoordinatesUsesPublicIDCursor(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PlaceRepository{db: db}
	cursor := int64(1001)

	var posts []post.Post
	tx := repo.nearByPlacesQuery(types.Filter{Cursor: &cursor, Limit: 12}, 12).Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"posts.contentable_type",
		"parent_id IS NULL",
		"posts.public_id <",
		"ORDER BY posts.public_id DESC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	if strings.Contains(sql, "ST_Distance") {
		t.Fatalf("expected non-location query without ST_Distance, got %s", sql)
	}
}

func TestNearByPlacesQueryWithCoordinatesUsesDistanceCursor(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PlaceRepository{db: db}
	cursor := int64(1001)
	distance := 250.5
	lat := 41.0082
	lon := 28.9784

	var posts []post.Post
	tx := repo.nearByPlacesQuery(types.Filter{
		Cursor:    &cursor,
		Distance:  &distance,
		Latitude:  &lat,
		Longitude: &lon,
		Limit:     12,
	}, 12).Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"LEFT JOIN locations",
		"ST_Distance",
		"locations.contentable_id = posts.id",
		"posts.public_id <",
		"ORDER BY",
		"posts.public_id DESC",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	vars := stringifyVars(tx.Statement.Vars)
	for _, value := range []string{"post", "250.5", "1001"} {
		if !strings.Contains(vars, value) {
			t.Fatalf("expected query vars to contain %q, got %s", value, vars)
		}
	}
}
