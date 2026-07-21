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
	"strings"

	post_payloads "core/models/post/payloads"

	seed "core/seeders"

	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
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
			return err
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
	return MigrateLegacyChatMediaOwnership(db)
}

func Seed(db *gorm.DB, node *helpers.Node) error {
	return seed.Seed(db, node)
}
