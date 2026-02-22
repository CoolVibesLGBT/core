package handlers

import (
	telegramServices "core/services/bot/telegram"

	tele "gopkg.in/telebot.v4"

	"os"

	"github.com/gofiber/fiber/v2"
)

func HandleTelegramUpdates(s *telegramServices.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := c.Get("X-Telegram-Bot-Api-Secret-Token")
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
