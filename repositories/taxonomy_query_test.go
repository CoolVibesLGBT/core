package repositories

import (
	"core/models/post"
	"core/models/taxonomy"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestExactClusterLookupQueryUsesExactSlugAndParent(t *testing.T) {
	db := newDryRunTaxonomyDB(t)

	var cluster taxonomy.Cluster
	pillarID := uuid.New()
	parentID := uuid.New()

	tx := exactClusterLookupQuery(db.Model(&taxonomy.Cluster{}), pillarID, &parentID, "Gay Bar").
		First(&cluster)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	if strings.Contains(strings.ToUpper(sql), "ILIKE") {
		t.Fatalf("expected exact lookup without ILIKE, got %s", sql)
	}
	if !strings.Contains(sql, "slug =") {
		t.Fatalf("expected slug equality in SQL, got %s", sql)
	}
	if !strings.Contains(sql, "parent_id =") {
		t.Fatalf("expected parent_id equality in SQL, got %s", sql)
	}
	if !strings.Contains(sql, "deleted_at IS NULL") {
		t.Fatalf("expected deleted_at guard in SQL, got %s", sql)
	}
}

func TestTaxonomyPillarMatchQueryUsesExistsSubquery(t *testing.T) {
	db := newDryRunTaxonomyDB(t)

	var pillars []taxonomy.Pillar
	tx := taxonomyPillarTreeQuery(
		taxonomyPillarMatchQuery(db.Model(&taxonomy.Pillar{}), "Gay Bar"),
	).Find(&pillars)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"FROM \"pillars\"",
		"EXISTS (",
		"FROM clusters",
		"FROM synonyms",
		"pillars.slug =",
		"clusters.search_vector ILIKE",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestApplyTaxonomyCategoryFilterUsesTaxonomyRelations(t *testing.T) {
	db := newDryRunTaxonomyDB(t)

	category := "Gay Bar"
	var posts []post.Post
	tx := applyTaxonomyCategoryFilter(db.Model(&post.Post{}), &category).Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"FROM post_clusters",
		"JOIN clusters",
		"JOIN pillars",
		"LEFT JOIN synonyms",
		"post_clusters.post_id = posts.id",
		"clusters.search_vector ILIKE",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func newDryRunTaxonomyDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=test password=test dbname=test sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	return db
}
