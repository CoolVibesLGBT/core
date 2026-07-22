package repositories

import (
	"context"
	"core/application/types"
	"strings"
	"testing"
)

func TestPublicUserProfileQuerySelectsOnlyPublicProjection(t *testing.T) {
	repo := &UserRepository{db: newDryRunTaxonomyDB(t)}
	var row publicUserProjectionRow
	tx := repo.publicUserProfileProjectionQuery(context.Background(), "public-user").Take(&row)
	if tx.Error != nil {
		t.Fatalf("profile projection query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	selectClause := projectionSelectClause(t, sql)
	for _, fragment := range []string{
		"users.public_id",
		"users.user_name",
		"avatar_file.url AS avatar_url",
		"avatar_file.variants AS avatar_variants",
		"jsonb_build_object(",
		"'follower_count'",
		") AS engagement_counts",
	} {
		if !strings.Contains(selectClause, fragment) {
			t.Fatalf("expected SELECT to contain %q, got %s", fragment, selectClause)
		}
	}
	assertNoPrivateUserProjection(t, selectClause)
	for _, privateCounter := range []string{"report_count", "deposit_amount", "withdraw_amount", "gift_amount"} {
		if strings.Contains(selectClause, privateCounter) {
			t.Fatalf("private engagement counter %q selected: %s", privateCounter, selectClause)
		}
	}
	assertPublicVisibilityPolicy(t, sql, tx.Statement.Vars)
	if strings.Contains(sql, "LOWER(users.email)") {
		t.Fatalf("public profile must not be discoverable by email: %s", sql)
	}
}

func TestPublicUserSearchQuerySelectsOnlySummaryProjection(t *testing.T) {
	repo := &UserRepository{db: newDryRunTaxonomyDB(t)}
	query := "pride"
	var rows []publicUserProjectionRow
	tx := repo.publicUserSearchProjectionQuery(types.Filter{Search: &query, Limit: 7}).Find(&rows)
	if tx.Error != nil {
		t.Fatalf("search projection query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	selectClause := projectionSelectClause(t, sql)
	for _, fragment := range []string{
		"users.public_id",
		"users.user_name",
		"users.display_name",
		"profile_location.city AS location_city",
		"avatar_file.url AS avatar_url",
		"avatar_file.variants AS avatar_variants",
	} {
		if !strings.Contains(selectClause, fragment) {
			t.Fatalf("expected SELECT to contain %q, got %s", fragment, selectClause)
		}
	}
	assertNoPrivateUserProjection(t, selectClause)
	assertPublicVisibilityPolicy(t, sql, tx.Statement.Vars)
	if strings.Contains(selectClause, "cover_file") || strings.Contains(selectClause, "engagement") {
		t.Fatalf("search summary selected profile-only data: %s", selectClause)
	}
}

func assertPublicVisibilityPolicy(t *testing.T, sql string, variables []any) {
	t.Helper()
	for _, fragment := range []string{
		"users.deleted_at IS NULL",
		"users.is_bot =",
		"users.user_role NOT IN",
		"users.privacy_level =",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("public visibility predicate %q missing: %s", fragment, sql)
		}
	}
	values := stringifyVars(variables)
	for _, value := range []string{"banned", "deleted", "pending", "public"} {
		if !strings.Contains(values, value) {
			t.Fatalf("public visibility value %q missing from vars %s", value, values)
		}
	}
}

func projectionSelectClause(t *testing.T, sql string) string {
	t.Helper()
	upper := strings.ToUpper(sql)
	fromIndex := strings.Index(upper, " FROM ")
	if fromIndex < 0 {
		t.Fatalf("query has no FROM clause: %s", sql)
	}
	return sql[:fromIndex]
}

func assertNoPrivateUserProjection(t *testing.T, selectClause string) {
	t.Helper()
	for _, forbidden := range []string{
		"users.id,",
		"users.email",
		"users.password",
		"users.balance",
		"users.user_role",
		"users.preferences_flags",
		"users.broadcast_info",
		"users.subscriptions",
		"users.socket_id",
		"avatar_file.storage_path",
		"cover_file.storage_path",
		"avatar_media.id",
		"cover_media.id",
		"users.*",
	} {
		if strings.Contains(selectClause, forbidden) {
			t.Fatalf("private field %q leaked into SELECT: %s", forbidden, selectClause)
		}
	}
}
