package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	"core/constants"
	"core/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleSetChatMuted(service ports.ChatMutePreference) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		chatID, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		value := c.FormValue("muted")
		if value != "true" && value != "false" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if err := service.SetChatMuted(c.Context(), actor, chatID, value == "true"); err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"is_muted": value == "true"})
	}
}
