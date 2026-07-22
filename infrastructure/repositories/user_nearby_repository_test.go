package repositories

import (
	"core/application/types"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNearbyUsersLocationQueryIsNarrowIndexableAndStable(t *testing.T) {
	database := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: database}
	actorID := uuid.New()
	cursor := int64(1234)
	distance := 42.25
	domain := "coolvibes"

	query, err := repo.nearbyUsersLocationQuery(types.Filter{
		AuthUser: &types.Actor{ID: actorID, PublicID: 99},
		Domain:   &domain,
		Cursor:   &cursor,
		Distance: &distance,
		Limit:    12,
	}, 41.0082, 28.9784)
	if err != nil {
		t.Fatalf("nearbyUsersLocationQuery() error = %v", err)
	}

	var rows []nearbyUserRow
	result := query.Find(&rows)
	if result.Error != nil {
		t.Fatalf("scan query error = %v", result.Error)
	}

	sql := strings.ToLower(result.Statement.SQL.String())
	for _, fragment := range []string{
		"join locations as nearby_locations",
		"nearby_locations.deleted_at is null",
		"nearby_locations.location_point is not null",
		"nearby_locations.location_point <->",
		"st_setsrid(st_makepoint(",
		"users.id <>",
		"not exists",
		"nearby_blocks.kind in",
		"users.privacy_level",
		"users.user_role not in",
		"users.domain",
		"order by nearby_locations.location_point <->",
		"users.public_id asc",
		"limit",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("nearby query is missing %q: %s", fragment, result.Statement.SQL.String())
		}
	}
	for _, forbidden := range []string{
		"left join locations as nearby_locations",
		"st_distance",
		"coalesce",
		"select *",
		"users.balance",
		"users.preferences_flags",
		"users.subscriptions",
		"users.broadcast_info",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("nearby query contains forbidden %q: %s", forbidden, result.Statement.SQL.String())
		}
	}
}

func TestNearbyUsersLocationCursorRequiresExactDistance(t *testing.T) {
	repo := &UserRepository{db: newDryRunTaxonomyDB(t)}
	cursor := int64(10)
	if _, err := repo.nearbyUsersLocationQuery(types.Filter{Cursor: &cursor}, 1, 2); err == nil {
		t.Fatal("expected a location cursor without its database distance to be rejected")
	}
}

func TestNearbyUsersAllDomainsDoesNotCreateLiteralAllPredicate(t *testing.T) {
	database := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: database}
	all := "all"
	query, err := repo.nearbyUsersLocationQuery(types.Filter{Domain: &all}, 1, 2)
	if err != nil {
		t.Fatalf("nearbyUsersLocationQuery() error = %v", err)
	}
	var rows []nearbyUserRow
	result := query.Find(&rows)
	if result.Error != nil {
		t.Fatalf("scan query error = %v", result.Error)
	}
	if strings.Contains(strings.ToLower(result.Statement.SQL.String()), "users.domain =") {
		t.Fatalf("all domains must not become domain = 'all': %s", result.Statement.SQL.String())
	}
}

func TestNearbyUserJSONIsLosslessAndDoesNotExposePersistenceFields(t *testing.T) {
	payload, err := json.Marshal(types.NearbyUser{
		PublicID:    types.SnowflakeID(9_223_372_036_854_775_000),
		UserName:    "alice",
		DisplayName: "Alice",
		Location:    &types.NearbyLocation{Latitude: 41, Longitude: 29},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(payload)
	if !strings.Contains(text, `"public_id":"9223372036854775000"`) {
		t.Fatalf("public id is not a lossless JSON string: %s", text)
	}
	for _, field := range []string{"balance", "preferences_flags", "storage_path", "user_role", "deleted_at"} {
		if strings.Contains(text, field) {
			t.Fatalf("nearby response leaked %q: %s", field, text)
		}
	}
}
