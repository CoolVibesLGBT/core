package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	"core/constants"
	"core/models/chat"
	"core/utils"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleEditMessage(service ports.ChatMessageEditor) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		chatID, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		messageID, err := uuid.Parse(c.FormValue("message_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}
		message, err := service.EditMessage(c.Context(), actor, chatID, messageID, c.FormValue("content"))
		if errors.Is(err, chat.ErrEmptyMessage) {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"message": message})
	}
}
