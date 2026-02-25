package main

import (
	"context"
	app "core/application"
	"core/constants"
	"core/helpers"
	"core/repositories"
	"core/services/db"
	"core/services/socket"
	"core/services/socket/managers"
	"core/workers"
	"core/workers/news"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var appInstance *app.App

func NewApp() (*app.App, error) {
	if appInstance != nil {
		return appInstance, nil
	}

	application, err := app.InitializeApp()
	if err != nil {
		return nil, err
	}

	migrateFlag := flag.Bool("migrate", false, "Run DB migrations")
	seedFlag := flag.Bool("seed", false, "Run DB seed")
	installFlag := flag.Bool("install", false, "Run DB migrate & seed")

	if !flag.Parsed() {
		flag.Parse()
	}

	if *installFlag {
		*seedFlag = true
		*migrateFlag = true
	}

	if *migrateFlag {
		fmt.Println("Migration:BEGIN")
		err = db.Migrate(application.DB)
		if err != nil {
			fmt.Println(err)
		}

		err = db.MigrateIndexes(application.DB)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Migration:END")

		os.Exit(0)
	}

	if *seedFlag {
		err = db.Seed(application.DB, application.SnowFlakeNode)
		if err != nil {
			fmt.Println(err)
		}
		os.Exit(0)
	}

	appInstance = application
	return appInstance, nil
}

func GetApp() (*app.App, error) {
	return NewApp()
}

func main() {
	fmt.Printf("%s Started \n", constants.APPLICATION_NAME)

	err := godotenv.Load()
	if err != nil {
		log.Println(".env not found, using system env")
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	if app.Router.TelegramService != nil {
		err = app.Router.TelegramService.RegisterWebhook()
		if err == nil {
			go app.Router.TelegramService.Start()
		} else {
			fmt.Println("TELEGRAM SERVISI BASLATILAMADI!")
		}
	} else {
		fmt.Println("TELEGRAM SERVISI BASLATILAMADI!")
	}

	fiberApp := app.Router.GetFiber()
	vapidKeys, err := helpers.CreateVapidKeys(app.DB)
	if err != nil {
		log.Fatal("VAPID anahtarı alınamadı:", err)
	} else {
		fmt.Printf("VAPIDKEY : %s \n", vapidKeys.PublicKey)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxWorkers := 10
	queueSize := 100

	dispatcher := workers.NewDispatcher(maxWorkers, queueSize)
	dispatcher.Run()

	go news.FetchAllFeedsSequentiallyAndProcess(dispatcher, app)

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

	// Bu kısım artık InitializeApp içerisinde yönetilebilir veya burada bırakılabilir
	notificationRepo := repositories.NewNotificationRepository(app.DB, app.SnowFlakeNode)
	notificationMgr := managers.NewNotificationManager(app.DB, notificationRepo)
	go func() {
		if _, err := socket.ListenServer(app.DB, notificationMgr); err != nil {
			log.Printf("socket server error: %v", err)
		}
	}()
	log.Println("App running on", os.Getenv("PORT"))
	log.Fatal(fiberApp.Listen(os.Getenv("PORT")))
}
