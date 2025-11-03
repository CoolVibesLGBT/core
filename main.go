package main

import (
	"bifrost/faker"
	"bifrost/helpers"
	"bifrost/routes"
	"bifrost/services/db"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	socketcli "github.com/zhouhui8915/go-socket.io-client"
	"gorm.io/gorm"
)

// App struct'u, tüm uygulama bileşenlerini içerir
type App struct {
	DB             *gorm.DB
	Router         routes.AppHandler
	SocketIOClient *socketcli.Client
	SnowFlakeNode  *helpers.Node

	//RedisClient         *_redis.Client
}

var instance *App // Singleton App instance

// NewApp, yeni bir App instance'ı oluşturur
func NewApp() (*App, error) {
	if instance == nil {
		snowFlakeNode, err := helpers.NewNode(1) // Node ID, genelde 0-1023 arası
		if err != nil {
			log.Fatalf("Failed to initialize snowflake node: %v", err)
		}
		// Database başlatma ve bağlantı
		err = db.InitDB()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		instance = &App{
			DB:            db.DB,
			Router:        routes.NewRouter(db.DB, snowFlakeNode),
			SnowFlakeNode: snowFlakeNode,
		}

		migrateFlag := flag.Bool("migrate", false, "Run DB migrations")
		seedFlag := flag.Bool("seed", false, "Run DB seed")

		flag.Parse()

		if *migrateFlag {
			err = db.Migrate(instance.DB)
			if err != nil {
				fmt.Println(err)
			}
			os.Exit(0) // migration sonrası programdan çık

		}

		if *seedFlag {
			err = db.Seed(instance.DB)
			if err != nil {
				fmt.Println(err)
			}
			os.Exit(0) // seed sonrası programdan çık

		}

		err = db.Migrate(instance.DB)
		if err != nil {
			fmt.Println(err)
		}

		err = db.Seed(instance.DB)
		if err != nil {
			fmt.Println(err)
		}

		faker.FakeUser(instance.DB, snowFlakeNode)

	}

	return instance, nil
}

func GetApp() (*App, error) {
	return NewApp()
}

// Close, uygulamayı kapatır ve kaynakları temizler
func (a *App) Close() {
	// Database bağlantısını kapatma ve diğer bileşenleri temizleme

	// Örneğin:
	//a.DB.Close()

	// Diğer bileşenler için de kapatma işlemleri yapılabilir.
}

func main() {
	fmt.Println("Merhaba, Dünya!")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	r := app.Router

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "authorization", "Content-Type", "Content-Length", "X-CSRF-Token", "Token", "session", "Origin", "Host", "Connection", "Accept-Encoding", "Accept-Language", "X-Requested-With"},
	})

	handler := c.Handler(r)
	log.Println("App running on", os.Getenv("PORT"))
	http.ListenAndServe(os.Getenv("PORT"), handler)

}
