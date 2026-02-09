package db

import (
	"context"
	"coolvibes/application"
	"coolvibes/models"
	"coolvibes/models/chat"
	"coolvibes/models/media"
	"coolvibes/models/notifications"
	"coolvibes/models/payment"
	"coolvibes/models/post"
	"coolvibes/models/taxonomy"
	"coolvibes/models/utils"
	"strings"

	post_payloads "coolvibes/models/post/payloads"

	seed "coolvibes/seeders"

	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB // Global değişken olarak veritabanı bağlantısı

func InitDB() error {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("DATABASE_URL is required")
	}

	errorOnlyLogger := logger.New(
		log.New(os.Stderr, "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel:                  logger.Error, // sadece Error
			IgnoreRecordNotFoundError: true,         // record not found'u loglama
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: errorOnlyLogger})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		// Hata işleme
	}

	sqlDB.SetMaxIdleConns(10)           // Boşta bekleyen bağlantıların maksimum sayısı
	sqlDB.SetMaxOpenConns(0)            // Aynı anda açık olabilecek maksimum bağlantı sayısı
	sqlDB.SetConnMaxLifetime(time.Hour) // Bağlantının yeniden kullanılabilir olacağı maksimum süre

	DB = db
	return nil
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
}

func MigrateIndexes(app *application.App) error {

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
	}

	for _, idx := range indexes {

		using := ""
		if idx.Using != "" {
			using = "USING " + idx.Using
		}

		columns := strings.Join(idx.Columns, ", ")

		query := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s
			ON %s
			%s (%s);
		`, idx.Name, idx.Table, using, columns)

		if err := app.DB.Exec(query).Error; err != nil {
			return err
		}
	}

	return nil
}

func Migrate(app *application.App) error {
	fmt.Println("Migration:Begin")
	//db.Logger = db.Logger.LogMode(logger.Silent)

	extensions := map[string]string{
		"uuid-ossp": "public",
		"postgis":   "public",
		"pg_trgm":   "public",
	}

	if err := EnableExtensions(context.Background(), app.DB, extensions); err != nil {
		return err
	}

	err := app.DB.AutoMigrate(

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

		&models.Story{},
		&models.Engagement{},
		&models.EngagementDetail{},

		&models.Preferences{},

		&models.User{},

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
	)

	return err
}

func Seed(app *application.App) error {
	fmt.Println("Seed Begin")
	seed.Seed(app)
	fmt.Println("Seed End")
	return nil
}
