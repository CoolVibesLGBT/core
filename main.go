package main

import (
	"context"
	"core/constants"
	"core/helpers"
	app "core/infrastructure/bootstrap"
	"core/infrastructure/db"
	"core/infrastructure/repositories"
	"core/infrastructure/socket"
	"core/infrastructure/socket/managers"
	"core/workers"
	mediaworker "core/workers/media"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	appInstance  *app.App
	botFlag      = flag.Bool("bot", false, "Run Bot Mode")
	migrateFlag  = flag.Bool("migrate", false, "Run DB migrations")
	seedFlag     = flag.Bool("seed", false, "Run DB seed")
	installFlag  = flag.Bool("install", false, "Run DB migrate & seed")
	mcpStdioFlag = flag.Bool("mcp-stdio", false, "Run the MCP server over stdio")
)

func ensureFlagsParsed() {
	if !flag.Parsed() {
		flag.Parse()
	}
}

func NewApp() (*app.App, error) {
	if appInstance != nil {
		return appInstance, nil
	}

	ensureFlagsParsed()

	application, err := app.InitializeApp()
	if err != nil {
		return nil, err
	}

	if *botFlag {
		fmt.Println("BOT MODE ON")
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

	/*
		s := "L2dldF9maWxlLzЕvМWI2N2I1МWЕ0МmМ5МjYxNTА5ODU4ZWU1ZTRlNmVmYTЕxZjRiOGU0МWJjLzЕ3МDМwМDАvМTcwМzЕ1Ny8xNzАzМTU3X3NkLm1wNС8,ZD00ODАmYnI9МTQzJnRpPTЕ3NzUyNzc2NTg~"
		decoded := helpers.Base164Decode(s)

		fmt.Println("Decoded bytes length:", len(decoded))
		fmt.Println("Decoded as string:")
		fmt.Println(string(decoded))

		return
	*/
	ensureFlagsParsed()

	err := godotenv.Load()
	if err != nil {
		log.Println(".env not found, using system env")
	}

	if *mcpStdioFlag {
		mcpServer, err := app.InitializeMCPOnly()
		if err != nil {
			log.Fatal(err)
		}

		if err := mcpServer.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	fmt.Printf("%s Started \n", constants.APPLICATION_NAME)

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

	dispatcher := workers.NewDispatcher(100, 1000)
	dispatcher.Run()
	mediaworker.StartProcessor(app.DB, app.SnowFlakeNode)

	if *botFlag {
		fmt.Println("Bot Flag")
		return
	}

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
