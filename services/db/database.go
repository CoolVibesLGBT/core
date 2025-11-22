package db

import (
	"coolvibes/models"
	"coolvibes/models/chat"
	"coolvibes/models/media"
	"coolvibes/models/notifications"
	"coolvibes/models/post"
	"coolvibes/models/utils"

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

func Migrate(db *gorm.DB) error {
	fmt.Println("Migration:Begin")
	//db.Logger = db.Logger.LogMode(logger.Silent)
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	db.Exec(`CREATE EXTENSION IF NOT EXISTS postgis;`)

	err := db.AutoMigrate(

		&models.VapidKey{},
		&models.ReportKind{},
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

		&models.MatchSeen{},
		&models.Follow{},
		&models.Like{},
		&models.Block{},
		&models.Favorite{},
		&models.Match{},

		&post.Post{},                // Önce parent tablo
		&post_payloads.Poll{},       // Poll önce
		&post_payloads.PollChoice{}, // child tablolar sonra
		&post_payloads.PollVote{},
		&post_payloads.EventKind{},
		&post_payloads.Event{}, // Event tablosu artık Post tablosundan sonra
		&post_payloads.EventAttendee{},

		&utils.Location{},

		// önce Chat tablosu, sonra Message
		&chat.Message{},
		&chat.Chat{},

		&chat.ChatParticipant{},
		&chat.MessageRead{},
	)

	/*
		db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_chats_pinned_msg'
			) THEN
				ALTER TABLE chats
				ADD CONSTRAINT fk_chats_pinned_msg
				FOREIGN KEY (pinned_msg_id) REFERENCES messages(id);
			END IF;
		END
		$$;
		`)
	*/

	return err
}

func Seed(db *gorm.DB) error {
	fmt.Println("Seed Begin")
	seed.Seed(db)
	fmt.Println("Seed End")
	return nil
}
