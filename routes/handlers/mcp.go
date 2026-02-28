package handlers

import (
	"core/constants"
	"core/mcp"
	services "core/services/user"
	"core/utils"

	"github.com/gofiber/fiber/v3"
)

func HandleMCP(s *services.AIService) fiber.Handler {
	return func(c fiber.Ctx) error {

		var payload any

		if c.Method() != fiber.MethodGet {
			if err := c.Bind().Body(&payload); err != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid input: "+err.Error())
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
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInternalServer, "Failed to route message: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, response.Payload, "Message routed successfully")
	}
}
