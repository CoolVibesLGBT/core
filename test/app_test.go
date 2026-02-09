package test

import (
	app "core/application"
	"core/helpers"
	"core/routes"
	"core/services/db"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func NewTestApp() (*app.App, error) {

	wd, _ := os.Getwd()
	fmt.Println("Working dir:", wd)

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	snowFlakeNode, err := helpers.NewNode(1)
	if err != nil {
		return nil, err
	}

	err = db.InitDB()
	if err != nil {
		return nil, err
	}

	instance := &app.App{
		DB:            db.DB,
		Router:        routes.NewRouter(db.DB, snowFlakeNode),
		SnowFlakeNode: snowFlakeNode,
	}

	return instance, nil
}
