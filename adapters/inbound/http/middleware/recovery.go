package middleware

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

func Recovery() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC recovered: %v\n", err)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "unknown error",
				})
			}
		}()
		return c.Next()
	}
}
