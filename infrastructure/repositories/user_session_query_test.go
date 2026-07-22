package repositories

import (
	"core/application/ports"
	"strings"
	"testing"
)

func TestSessionUserQueryUsesOneNarrowProjection(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: db}
	query := repo.sessionUserQuery(42)
	var user ports.SessionUser
	result := query.Find(&user)
	if result.Error != nil {
		t.Fatalf("sessionUserQuery() error = %v", result.Error)
	}

	if len(result.Statement.Preloads) != 0 {
		t.Fatalf("session query must not preload associations: %#v", result.Statement.Preloads)
	}

	sql := strings.ToLower(result.Statement.SQL.String())
	for _, required := range []string{
		"users.id",
		"users.public_id",
		"users.user_role as role",
		"exists (",
		"from locations",
		"locations.deleted_at is null",
		"as has_location",
		"users.deleted_at is null",
		"users.is_bot = false",
		"users.user_role not in",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("session projection is missing %q: %s", required, result.Statement.SQL.String())
		}
	}
	for _, forbidden := range []string{
		"users.password",
		"users.email",
		"users.languages",
		"users.hobbies",
		"users.subscriptions",
		"users.broadcast_info",
		"select *",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("session projection includes forbidden %q: %s", forbidden, result.Statement.SQL.String())
		}
	}
}
