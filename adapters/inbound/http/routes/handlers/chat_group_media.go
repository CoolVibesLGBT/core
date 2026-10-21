package handlers

import (
	"core/adapters/inbound/http/middleware"
	upload "core/adapters/media"
	"core/application/ports"
	"core/constants"
	domainmedia "core/domain/media"
	"core/models/chat"
	"core/utils"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleUpdateChatGroup(s ports.ChatGroupMediaUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		id, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		var avatar *domainmedia.UploadedFile
		if header, err := c.FormFile("avatar"); err == nil {
			avatar = upload.FromMultipart(header)
		}
		entity, err := s.UpdateGroup(c.Context(), actor, id, c.FormValue("title"), c.FormValue("description"), avatar, c.FormValue("remove_avatar") == "true", chat.GroupOptions{Visibility: c.FormValue("visibility"), PublicUsername: c.FormValue("public_username")})
		if errors.Is(err, chat.ErrInvalidGroupAvatar) || errors.Is(err, chat.ErrEmptyMessage) || errors.Is(err, chat.ErrInvalidGroupSettings) {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"chat": entity})
	}
}
