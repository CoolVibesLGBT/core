package test

import (
	app "core/application"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func NewTestApp() (*app.App, error) {

	wd, _ := os.Getwd()
	fmt.Println("Working dir:", wd)

	err := godotenv.Load("../.env") // Testler alt klasörde olduğu için root'taki .env'ye bakıyoruz
	if err != nil {
		log.Println("Error loading .env file")
	}

	return app.InitializeApp()
}
