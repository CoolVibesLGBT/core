package handlers

import (
	"core/application/ports"
	"crypto/sha256"
	"crypto/subtle"

	tele "gopkg.in/telebot.v4"

	"os"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func validTelegramWebhookSecret(provided, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" || provided == "" {
		return false
	}

	providedHash := sha256.Sum256([]byte(provided))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}

func telegramProcessorIsNil(processor ports.TelegramUpdateProcessor) bool {
	if processor == nil {
		return true
	}

	value := reflect.ValueOf(processor)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func HandleTelegramUpdates(processor ports.TelegramUpdateProcessor) fiber.Handler {
	return func(c fiber.Ctx) error {
		secret := c.Get("X-Telegram-Bot-Api-Secret-Token")
		if !validTelegramWebhookSecret(secret, os.Getenv("TELEGRAM_WEBHOOK_SECRET")) {
			return c.SendStatus(403)
		}
		if telegramProcessorIsNil(processor) {
			return c.SendStatus(fiber.StatusServiceUnavailable)
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
