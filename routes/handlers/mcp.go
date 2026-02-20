package handlers

import (
	"core/mcp"
	services "core/services/user"

	"github.com/gofiber/fiber/v2"
)

func HandleMCP(s *services.AIService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var payload any

		if c.Method() != fiber.MethodGet {
			if err := c.BodyParser(&payload); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid body",
				})
			}
		}

		target := c.Params("target")
		action := c.Params("action")

		msg := mcp.NewMessage(
			"http",
			target,
			action,
			payload,
			mcp.TypeCommand,
		)

		response, err := s.MCPServer().Router().Route(msg)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(response.Payload)
	}
}
