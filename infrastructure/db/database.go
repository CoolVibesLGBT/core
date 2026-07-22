package db

import (
	"context"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/notifications"
	"core/models/payment"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"encoding/json"
	"errors"
	"strings"

	post_payloads "core/models/post/payloads"

	seed "core/seeders"
	reportkinds "core/seeders/reportkinds"

	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB   *gorm.DB // Global değişken olarak veritabanı bağlantısı
	dbMu sync.Mutex
)

const (
	defaultConnectTimeout = 3 * time.Second
	defaultMaxIdleConns   = 10
	defaultMaxOpenConns   = 50
)

func NewDatabase() (*gorm.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if DB == nil {
		database, err := openDatabase()
		if err != nil {
			return nil, err
		}
		DB = database
	}
	return DB, nil
}

func InitDB() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if DB != nil {
		return nil
	}
	database, err := openDatabase()
	if err != nil {
		return err
	}
	DB = database
	return nil
}

func openDatabase() (*gorm.DB, error) {
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" {
		if err := godotenv.Load(); err != nil {
			log.Println(".env not found, using system env")
		}
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	errorOnlyLogger := logger.New(
		log.New(os.Stderr, "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel:                  logger.Error, // sadece Error
			IgnoreRecordNotFoundError: true,         // record not found'u loglama
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:               errorOnlyLogger,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", defaultMaxIdleConns))
	sqlDB.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", defaultMaxOpenConns))
	sqlDB.SetConnMaxLifetime(time.Hour) // Bağlantının yeniden kullanılabilir olacağı maksimum süre

	pingTimeout := envDuration("DB_CONNECT_TIMEOUT", defaultConnectTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to connect database within %s: %w", pingTimeout, err)
	}

	return db, nil
}

func Close(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	dbMu.Lock()
	if DB == database {
		DB = nil
	}
	dbMu.Unlock()

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql db for close: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func EnableExtension(db *gorm.DB, name string) error {
	return db.Exec(fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s";`, name)).Error
}

func EnableUUID(db *gorm.DB) error {
	return EnableExtension(db, "uuid-ossp")
}

func EnablePostGIS(db *gorm.DB) error {
	return EnableExtension(db, "postgis")
}

func EnableTrigram(db *gorm.DB) error {
	return EnableExtension(db, "pg_trgm")
}

func EnableExtensions(ctx context.Context, db *gorm.DB, extensions map[string]string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		for name, schema := range extensions {

			query := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, name)

			if schema != "" {
				query += fmt.Sprintf(` WITH SCHEMA "%s"`, schema)
			}

			query += ";"

			if err := tx.Exec(query).Error; err != nil {
				return fmt.Errorf("failed to enable extension %s: %w", name, err)
			}
		}

		return nil
	})
}

type IndexDefinition struct {
	Name      string
	Table     string
	Using     string // gin, btree vs
	Columns   []string
	Condition string // optional (partial index)
	Unique    bool
}

func MigrateIndexes(db *gorm.DB) error {
	dropQueries := []string{
		"DROP INDEX IF EXISTS idx_synonyms_slug;",
		"DROP INDEX IF EXISTS idx_posts_active_chat_message_expiry;",
		"DROP INDEX IF EXISTS idx_reports_moderation_queue;",
	}

	for _, query := range dropQueries {
		if err := db.Exec(query).Error; err != nil {
			return err
		}
	}

	indexes := []IndexDefinition{
		{
			Name:    "idx_clusters_search_vector_trgm",
			Table:   "clusters",
			Using:   "gin",
			Columns: []string{"search_vector gin_trgm_ops"},
		},
		{
			Name:    "idx_clusters_slug",
			Table:   "clusters",
			Using:   "btree",
			Columns: []string{"slug"},
		},
		{
			Name:    "idx_synonyms_slug",
			Table:   "synonyms",
			Using:   "btree",
			Columns: []string{"slug"},
		},
		{
			Name:      "uidx_clusters_root_identity",
			Table:     "clusters",
			Using:     "btree",
			Columns:   []string{"pillar_id", "slug"},
			Condition: "parent_id IS NULL AND deleted_at IS NULL",
			Unique:    true,
		},
		{
			Name:      "uidx_clusters_child_identity",
			Table:     "clusters",
			Using:     "btree",
			Columns:   []string{"pillar_id", "parent_id", "slug"},
			Condition: "parent_id IS NOT NULL AND deleted_at IS NULL",
			Unique:    true,
		},
		{
			Name:    "uidx_synonyms_cluster_slug",
			Table:   "synonyms",
			Using:   "btree",
			Columns: []string{"cluster_id", "slug"},
			Unique:  true,
		},
		{
			Name:      "uidx_synonyms_primary_per_cluster",
			Table:     "synonyms",
			Using:     "btree",
			Columns:   []string{"cluster_id"},
			Condition: "is_primary = true",
			Unique:    true,
		},
		{
			Name:      "idx_posts_active_chat_message_expiry",
			Table:     "posts",
			Using:     "btree",
			Columns:   []string{"expires_at"},
			Condition: "opened_at IS NOT NULL AND expires_at IS NOT NULL AND deleted_at IS NULL AND post_kind = 'message' AND contentable_type = 'chat'",
		},
	}
	indexes = append(indexes, reportIndexDefinitions()...)
	indexes = append(indexes, discoveryIndexDefinitions()...)
	indexes = append(indexes, chatPaginationIndexDefinitions()...)
	indexes = append(indexes, matchIndexDefinitions()...)
	if err := createIndexDefinitions(db, indexes); err != nil {
		return err
	}
	if err := MigrateEngagementAggregateUniqueness(db); err != nil {
		return err
	}
	if err := MigrateLocationOwnerUniqueness(db); err != nil {
		return err
	}

	return MigrateUserIdentityIndexes(db)
}

// MigrateEngagementAggregateUniqueness makes the polymorphic content owner a
// database-enforced aggregate identity. Historical duplicates are merged
// while the tables are locked: details move to the oldest aggregate and known
// count/amount values are summed before the duplicate rows are removed.
func MigrateEngagementAggregateUniqueness(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate engagement aggregate uniqueness: database is nil")
	}
	validIndex, err := hasValidEngagementAggregateIndex(db)
	if err != nil {
		return err
	}
	if validIndex {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// This is an explicit maintenance migration. Lock both tables in the
		// same order used by writers and block new row lockers before touching
		// details, otherwise a live tip could form a row/table-lock cycle.
		if err := tx.Exec("LOCK TABLE engagements, engagement_details IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return fmt.Errorf("lock engagement aggregate tables for deduplication: %w", err)
		}
		validIndex, err := hasValidEngagementAggregateIndex(tx)
		if err != nil {
			return err
		}
		if validIndex {
			return nil
		}

		type duplicateOwner struct {
			ContentableID   uuid.UUID
			ContentableType models.EngagementContentableType
		}
		var owners []duplicateOwner
		if err := tx.Model(&models.Engagement{}).
			Select("contentable_id, contentable_type").
			Group("contentable_id, contentable_type").
			Having("COUNT(*) > 1").
			Scan(&owners).Error; err != nil {
			return fmt.Errorf("list duplicate engagement aggregates: %w", err)
		}

		for _, owner := range owners {
			var aggregates []models.Engagement
			if err := tx.
				Where("contentable_id = ? AND contentable_type = ?", owner.ContentableID, owner.ContentableType).
				Order("created_at ASC, id ASC").
				Find(&aggregates).Error; err != nil {
				return fmt.Errorf("load duplicate engagement aggregates: %w", err)
			}
			if len(aggregates) < 2 {
				continue
			}

			mergedCounts, err := mergeEngagementAggregateCounts(aggregates)
			if err != nil {
				return fmt.Errorf("merge engagement counts for %s/%s: %w", owner.ContentableType, owner.ContentableID, err)
			}
			canonicalID := aggregates[0].ID
			duplicateIDs := make([]uuid.UUID, 0, len(aggregates)-1)
			for _, aggregate := range aggregates[1:] {
				duplicateIDs = append(duplicateIDs, aggregate.ID)
			}

			if err := tx.Model(&models.EngagementDetail{}).
				Where("engagement_id IN ?", duplicateIDs).
				Update("engagement_id", canonicalID).Error; err != nil {
				return fmt.Errorf("move duplicate engagement details: %w", err)
			}
			if err := tx.Model(&models.Engagement{}).
				Where("id = ?", canonicalID).
				Updates(map[string]interface{}{
					"counts":     datatypes.JSON(mergedCounts),
					"updated_at": time.Now().UTC(),
				}).Error; err != nil {
				return fmt.Errorf("update canonical engagement counts: %w", err)
			}
			if err := tx.Delete(&models.Engagement{}, "id IN ?", duplicateIDs).Error; err != nil {
				return fmt.Errorf("delete duplicate engagement aggregates: %w", err)
			}
		}

		if err := tx.Exec("DROP INDEX IF EXISTS uidx_engagements_contentable").Error; err != nil {
			return fmt.Errorf("drop malformed engagement aggregate index: %w", err)
		}
		return createIndexDefinitions(tx, engagementAggregateIndexDefinitions())
	})
}

func hasValidEngagementAggregateIndex(db *gorm.DB) (bool, error) {
	var definition string
	result := db.Raw(`
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND tablename = 'engagements'
		  AND indexname = 'uidx_engagements_contentable'
	`).Scan(&definition)
	if result.Error != nil {
		return false, fmt.Errorf("inspect engagement aggregate index: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(definition, `"`, "")), " "))
	return strings.HasPrefix(normalized, "create unique index uidx_engagements_contentable ") &&
		strings.Contains(normalized, "(contentable_type, contentable_id)") &&
		!strings.Contains(normalized, " where "), nil
}

func engagementAggregateIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:    "uidx_engagements_contentable",
			Table:   "engagements",
			Using:   "btree",
			Columns: []string{"contentable_type", "contentable_id"},
			Unique:  true,
		},
	}
}

func mergeEngagementAggregateCounts(aggregates []models.Engagement) ([]byte, error) {
	knownCounts := make(map[string]bool)
	knownAmounts := make(map[string]bool)
	for _, keys := range models.EngagementCountKeys {
		knownCounts[keys.CountKey] = true
		if keys.AmountKey != "" {
			knownAmounts[keys.AmountKey] = true
		}
	}

	totals := make(map[string]decimal.Decimal)
	seenKnown := make(map[string]bool)
	unknown := make(map[string]json.RawMessage)
	for _, aggregate := range aggregates {
		if len(aggregate.Counts) == 0 || string(aggregate.Counts) == "null" {
			continue
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(aggregate.Counts, &values); err != nil {
			return nil, err
		}
		for key, raw := range values {
			if !knownCounts[key] && !knownAmounts[key] {
				if _, exists := unknown[key]; !exists {
					unknown[key] = append(json.RawMessage(nil), raw...)
				}
				continue
			}
			value, err := engagementJSONDecimal(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid %s value: %w", key, err)
			}
			if value.IsNegative() {
				return nil, fmt.Errorf("invalid negative %s value", key)
			}
			totals[key] = totals[key].Add(value)
			seenKnown[key] = true
		}
	}

	result := make(map[string]interface{}, len(unknown)+len(seenKnown))
	for key, raw := range unknown {
		var value interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	for key := range seenKnown {
		value := totals[key]
		if knownAmounts[key] {
			result[key] = value.String()
			continue
		}
		if !value.Equal(value.Truncate(0)) || value.IsNegative() {
			return nil, fmt.Errorf("count %s is not a non-negative integer", key)
		}
		parsed, err := strconv.ParseInt(value.StringFixed(0), 10, 64)
		if err != nil {
			return nil, err
		}
		result[key] = parsed
	}
	return json.Marshal(result)
}

func engagementJSONDecimal(raw json.RawMessage) (decimal.Decimal, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return decimal.Zero, nil
	}
	if strings.HasPrefix(text, "\"") {
		if err := json.Unmarshal(raw, &text); err != nil {
			return decimal.Zero, err
		}
	}
	if len(text) > 128 {
		return decimal.Zero, errors.New("numeric JSON value is too long")
	}
	if strings.ContainsAny(text, "eE") {
		return decimal.Zero, errors.New("scientific notation is not supported in aggregate counts")
	}
	return decimal.NewFromString(text)
}

// MigrateLocationOwnerUniqueness makes the polymorphic owner the identity of
// an active location. The table lock closes the write window between cleaning
// historical duplicates and installing the partial unique index. Older rows
// are soft-deleted so no location data is irreversibly discarded.
func MigrateLocationOwnerUniqueness(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate location owner uniqueness: database is nil")
	}
	if db.Migrator().HasIndex(&utils.Location{}, "uidx_locations_active_owner") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE locations IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return fmt.Errorf("lock locations for owner deduplication: %w", err)
		}
		if err := tx.Exec(`
			WITH ranked_locations AS (
				SELECT id,
					ROW_NUMBER() OVER (
						PARTITION BY contentable_type, contentable_id
						ORDER BY updated_at DESC NULLS LAST,
							created_at DESC NULLS LAST,
							id DESC
					) AS owner_rank
				FROM locations
				WHERE deleted_at IS NULL
			)
			UPDATE locations AS location
			SET deleted_at = NOW(), updated_at = NOW()
			FROM ranked_locations
			WHERE location.id = ranked_locations.id
			  AND ranked_locations.owner_rank > 1
		`).Error; err != nil {
			return fmt.Errorf("deduplicate active location owners: %w", err)
		}

		return createIndexDefinitions(tx, locationOwnerIndexDefinitions())
	})
}

func locationOwnerIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:      "uidx_locations_active_owner",
			Table:     "locations",
			Using:     "btree",
			Columns:   []string{"contentable_type", "contentable_id"},
			Condition: "deleted_at IS NULL",
			Unique:    true,
		},
	}
}

// discoveryIndexDefinitions support keyset/KNN discovery without sorting the
// complete users or locations tables. The spatial index is deliberately
// shared by user and place discovery because location is polymorphic.
func discoveryIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:      "idx_locations_active_point_gist",
			Table:     "locations",
			Using:     "gist",
			Columns:   []string{"location_point"},
			Condition: "deleted_at IS NULL AND location_point IS NOT NULL",
		},
		{
			Name:      "idx_users_active_domain_public_id",
			Table:     "users",
			Using:     "btree",
			Columns:   []string{"domain", "public_id"},
			Condition: "deleted_at IS NULL",
		},
	}
}

func MigrateUserIdentityIndexes(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate user identity indexes: database is nil")
	}

	// Build both constraints as one atomic migration. A duplicate in either
	// identity field rolls the whole migration back instead of leaving only one
	// of the two uniqueness guarantees installed.
	return db.Transaction(func(tx *gorm.DB) error {
		return createIndexDefinitions(tx, userIdentityIndexDefinitions())
	})
}

func userIdentityIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:      "uidx_users_active_user_name_ci",
			Table:     "users",
			Using:     "btree",
			Columns:   []string{"LOWER(user_name)"},
			Condition: "deleted_at IS NULL",
			Unique:    true,
		},
		{
			Name:    "uidx_users_active_email_ci",
			Table:   "users",
			Using:   "btree",
			Columns: []string{"LOWER(email)"},
			// Email is optional; blank values are not identities and therefore
			// must not prevent multiple email-less active accounts.
			Condition: "deleted_at IS NULL AND NULLIF(BTRIM(email), '') IS NOT NULL",
			Unique:    true,
		},
	}
}

func createIndexDefinitions(db *gorm.DB, indexes []IndexDefinition) error {

	for _, idx := range indexes {
		createIndex := "CREATE INDEX IF NOT EXISTS"
		if idx.Unique {
			createIndex = "CREATE UNIQUE INDEX IF NOT EXISTS"
		}

		using := ""
		if idx.Using != "" {
			using = "USING " + idx.Using
		}

		columns := strings.Join(idx.Columns, ", ")

		query := fmt.Sprintf(`
			%s %s
			ON %s
			%s (%s)%s;
		`, createIndex, idx.Name, idx.Table, using, columns, func() string {
			if idx.Condition == "" {
				return ""
			}
			return " WHERE " + idx.Condition
		}())

		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("create index %s: %w", idx.Name, err)
		}
	}

	return nil
}

func reportIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:    "idx_reports_status_queue",
			Table:   "reports",
			Using:   "btree",
			Columns: []string{"status", "created_at DESC", "id DESC"},
		},
		{
			Name:    "idx_reports_type_status_queue",
			Table:   "reports",
			Using:   "btree",
			Columns: []string{"contentable_type", "status", "created_at DESC", "id DESC"},
		},
		{
			Name:    "idx_reports_target_status",
			Table:   "reports",
			Using:   "btree",
			Columns: []string{"contentable_type", "contentable_id", "status"},
		},
	}
}

func MigrateLegacyChatMediaOwnership(db *gorm.DB) error {
	// Older chat uploads used owner_type=chat even though owner_id was the
	// message post UUID. Correct only rows proven to belong to chat-message
	// posts; file metadata and physical storage are intentionally untouched.
	return db.Exec(`
		UPDATE medias AS m
		SET owner_type = 'post'
		FROM posts AS p
		WHERE m.owner_type = 'chat'
		  AND m.role = 'chat_media'
		  AND m.owner_id = p.id
		  AND p.post_kind = 'message'
		  AND p.contentable_type = 'chat'
	`).Error
}

// MigrateProtectedMediaVisibility repairs rows created by older media writers
// whose ORM default could override an explicit false IsPublic value, then
// enforces the invariant at the database boundary for every protected role.
func MigrateProtectedMediaVisibility(db *gorm.DB) error {
	if db == nil {
		return errors.New("migrate protected media visibility: database is nil")
	}
	protectedRoles := []media.MediaRole{
		media.RolePrivatePhoto,
		media.RoleChatImage,
		media.RoleChatMedia,
		media.RoleChatVideo,
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&media.Media{}).
			Where("role IN ? AND is_public IS DISTINCT FROM FALSE", protectedRoles).
			UpdateColumn("is_public", false).Error; err != nil {
			return err
		}
		if tx.Name() != "postgres" {
			return nil
		}
		if err := tx.Exec("UPDATE medias SET is_public = FALSE WHERE is_public IS NULL").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE medias ALTER COLUMN is_public SET DEFAULT FALSE").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE medias ALTER COLUMN is_public SET NOT NULL").Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conname = 'chk_medias_protected_visibility'
					  AND conrelid = 'medias'::regclass
				) THEN
					ALTER TABLE medias ADD CONSTRAINT chk_medias_protected_visibility
					CHECK (role NOT IN ('private_photo', 'chat_image', 'chat_media', 'chat_video') OR is_public = FALSE)
					NOT VALID;
				END IF;
			END $$;
		`).Error; err != nil {
			return err
		}
		return tx.Exec("ALTER TABLE medias VALIDATE CONSTRAINT chk_medias_protected_visibility").Error
	})
}

func Migrate(db *gorm.DB) error {
	fmt.Println("Migration:Begin")
	//db.Logger = db.Logger.LogMode(logger.Silent)

	extensions := map[string]string{
		"uuid-ossp": "public",
		"postgis":   "public",
		"pg_trgm":   "public",
	}

	if err := EnableExtensions(context.Background(), db, extensions); err != nil {
		return err
	}
	// Existing installations may still have the legacy nullable/default-true
	// visibility column. Repair it before AutoMigrate tightens the model to
	// NOT NULL so old NULL rows cannot make the schema migration fail.
	if db.Migrator().HasTable(&media.Media{}) {
		if err := MigrateProtectedMediaVisibility(db); err != nil {
			return err
		}
	}

	err := db.AutoMigrate(

		&taxonomy.Entity{},
		&taxonomy.Intent{},
		&taxonomy.ClusterEntity{},
		&taxonomy.ClusterIntent{},
		&taxonomy.Pillar{},
		&taxonomy.Cluster{},
		&taxonomy.Synonym{},

		&models.VapidKey{},
		&payment.PaymentMethod{},
		&models.ReportKind{},
		&models.Report{},
		&notifications.Notification{},
		&utils.FileMetadata{},

		&media.Media{},

		&models.Engagement{},
		&models.EngagementDetail{},

		&models.Preferences{},

		&models.User{},
		&models.PrivatePhotoAccessRequest{},
		&models.Wallet{},

		&models.Mention{},
		&models.Hashtag{},

		&post.Post{},                // Önce parent tablo
		&post_payloads.Poll{},       // Poll önce
		&post_payloads.PollChoice{}, // child tablolar sonra
		&post_payloads.PollVote{},
		&post_payloads.EventKind{},
		&post_payloads.Event{}, // Event tablosu artık Post tablosundan sonra
		&post_payloads.EventAttendee{},

		&utils.Location{},

		// önce Chat tablosu, sonra Message
		&chat.Chat{},

		&chat.ChatParticipant{},
		&chat.MessageView{},
	)
	if err != nil {
		return err
	}
	if err := MigrateLegacyChatMediaOwnership(db); err != nil {
		return err
	}
	if err := MigrateProtectedMediaVisibility(db); err != nil {
		return err
	}
	// Report kinds are required reference data, not optional demo content.
	// Keeping them in the migration path makes post/user reporting usable even
	// on deployments that intentionally run -migrate without the full seeder.
	return reportkinds.SeedReportKinds(db)
}

func Seed(db *gorm.DB, node *helpers.Node) error {
	return seed.Seed(db, node)
}
