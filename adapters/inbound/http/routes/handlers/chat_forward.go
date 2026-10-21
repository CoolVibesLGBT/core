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

func HandleForwardMessages(service ports.ChatMessageForwarder) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		sourceID, err := uuid.Parse(c.FormValue("source_chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		targetID, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}
		values := form.Value["message_ids[]"]
		if len(values) == 0 {
			values = form.Value["message_ids"]
		}
		if len(values) == 0 || len(values) > 100 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}
		ids := make([]uuid.UUID, 0, len(values))
		seen := map[uuid.UUID]bool{}
		for _, value := range values {
			id, err := uuid.Parse(value)
			if err != nil || id == uuid.Nil || seen[id] {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
			}
			seen[id] = true
			ids = append(ids, id)
		}
		messages, err := service.ForwardMessages(c.Context(), actor, sourceID, targetID, ids, c.FormValue("content"))
		if errors.Is(err, chat.ErrEmptyMessage) {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"messages": messages})
	}
}
