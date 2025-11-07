package main

import (
	"coolvibes/helpers"
	"coolvibes/routes"
	"coolvibes/services/db"
	"coolvibes/services/socket"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	socketio "github.com/vchitai/go-socket.io/v4"
	"gorm.io/gorm"
)

// App struct'u, tüm uygulama bileşenlerini içerir
type App struct {
	DB            *gorm.DB
	Router        routes.AppHandler
	SnowFlakeNode *helpers.Node
	SocketServer  *socketio.Server
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
		installFlag := flag.Bool("install", false, "Run DB migrate & seed")

		flag.Parse()

		if *installFlag {
			*seedFlag = true
			*migrateFlag = true
		}

		if *migrateFlag {
			fmt.Println("Migration:BEGIN")
			err = db.Migrate(instance.DB)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Migration:END")
		}

		if *seedFlag {
			err = db.Seed(instance.DB)
			if err != nil {
				fmt.Println(err)
			}
		}

		//	faker.FakeUser(instance.DB, snowFlakeNode)

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

	fmt.Println("ROUTER")

	applicationRouter := app.Router

	httpCors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "authorization", "Content-Type", "Content-Length", "X-CSRF-Token", "Token", "session", "Origin", "Host", "Connection", "Accept-Encoding", "Accept-Language", "X-Requested-With"},
	})

	go socket.ListenServer(app.DB)
	httpHandler := httpCors.Handler(applicationRouter)
	log.Println("App running on", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(os.Getenv("PORT"), httpHandler))
}
