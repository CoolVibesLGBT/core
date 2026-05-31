package handlers

import (
	"core/application/ports"

	tele "gopkg.in/telebot.v4"

	"os"

	"github.com/gofiber/fiber/v3"
)

func HandleTelegramUpdates(processor ports.TelegramUpdateProcessor) fiber.Handler {
	return func(c fiber.Ctx) error {
		secret := c.Get("X-Telegram-Bot-Api-Secret-Token")
		if secret != os.Getenv("TELEGRAM_WEBHOOK_SECRET") {
			return c.SendStatus(403)
		}
		var update tele.Update
		if err := c.Bind().JSON(&update); err != nil {
			return c.SendStatus(400)
		}

		if err := processor.ProcessUpdate(update); err != nil {
			return c.SendStatus(500)
		}

		return c.SendStatus(200)
	}
}
