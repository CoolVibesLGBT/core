package main

import (
	"context"
	app "coolvibes/application"
	"coolvibes/helpers"
	"coolvibes/repositories"
	"coolvibes/routes"
	"coolvibes/services/db"
	"coolvibes/services/socket"
	"coolvibes/services/socket/managers"
	"coolvibes/test"
	"coolvibes/workers"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var instance *app.App // Singleton App instance

// NewApp, yeni bir App instance'ı oluşturur
func NewApp() (*app.App, error) {
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

		instance = &app.App{
			DB:            db.DB,
			Router:        routes.NewRouter(db.DB, snowFlakeNode),
			SnowFlakeNode: snowFlakeNode,
		}

		migrateFlag := flag.Bool("migrate", false, "Run DB migrations")
		seedFlag := flag.Bool("seed", false, "Run DB seed")
		installFlag := flag.Bool("install", false, "Run DB migrate & seed")
		testFlag := flag.Bool("test", false, "Test")

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
			err = db.Seed(instance)
			if err != nil {
				fmt.Println(err)
			}
		}

		if *testFlag {
			test.StartImageTest(db.DB, snowFlakeNode)
			//test.StartTest(db.DB, snowFlakeNode)
			os.Exit(0)
		}

		//faker.FakeUser(instance.DB, snowFlakeNode)

	}

	return instance, nil
}

func GetApp() (*app.App, error) {
	return NewApp()
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

	err = app.Router.TelegramService.RegisterWebhook()

	if err == nil {
		go app.Router.TelegramService.Start()
	} else {
		fmt.Println("TELEGRAM SERVISI BASLATILAMADI!")
	}

	fiberApp := app.Router.GetFiber()
	vapidKeys, err := helpers.CreateVapidKeys(app.DB)
	if err != nil {
		log.Fatal("VAPID anahtarı alınamadı:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxWorkers := 10
	queueSize := 100

	dispatcher := workers.NewDispatcher(maxWorkers, queueSize)
	dispatcher.Run()

	//go news.SubmitRSSFetchTasks(dispatcher, app)

	ticker := time.NewTicker(5000 * time.Hour)

	fmt.Println("TICKEr", ticker, ctx)

	/*
		go func() {
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					news.SubmitRSSFetchTasks(dispatcher, app)

				case <-ctx.Done():
					return
				}
			}
		}()*/

	fmt.Println("PublicKey:", vapidKeys.PublicKey)
	fmt.Println("PrivateKey:", vapidKeys.PrivateKey)

	notificationRepo := repositories.NewNotificationRepository(app.DB, nil)
	notificationMgr := managers.NewNotificationManager(app.DB, notificationRepo)
	go socket.ListenServer(app.DB, notificationMgr)

	//httpHandler := httpCors.Handler(applicationRouter)
	log.Println("App running on", os.Getenv("PORT"))
	log.Fatal(fiberApp.Listen(os.Getenv("PORT")))
}
