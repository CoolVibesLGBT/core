package handlers

import (
	telegramServices "coolvibes/services/bot/telegram"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"os"

	"github.com/gofiber/fiber/v2"
)

func HandleTelegramUpdates(s *telegramServices.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := c.Get("X-Telegram-Bot-Api-Secret-Token")
		fmt.Println("SECRET TOKEN", secret)
		if secret != os.Getenv("TELEGRAM_WEBHOOK_SECRET") {
			return c.SendStatus(403)
		}

		var update tele.Update
		if err := c.BodyParser(&update); err != nil {
			return c.SendStatus(400)
		}

		s.Bot.ProcessUpdate(update)

		return c.SendStatus(200)
	}
}
