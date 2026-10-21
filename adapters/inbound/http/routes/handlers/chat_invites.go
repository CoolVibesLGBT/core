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

func HandleExportGroupInvite(s ports.ChatGroupInvites) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		id, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		invitation, err := s.ExportGroupInvite(c.Context(), actor, id, c.FormValue("rotate") == "true")
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"invite": invitation})
	}
}
func HandlePreviewGroup(s ports.ChatGroupInvites) fiber.Handler {
	return func(c fiber.Ctx) error {
		view, err := s.PreviewGroup(c.Context(), c.FormValue("invite_token"), c.FormValue("public_username"))
		if err != nil {
			return utils.SendError(c, fiber.StatusNotFound, constants.ErrInvalidInput)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"group": view})
	}
}
func HandleJoinGroup(s ports.ChatGroupInvites) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		entity, err := s.JoinGroup(c.Context(), actor, c.FormValue("invite_token"), c.FormValue("public_username"))
		if errors.Is(err, chat.ErrGroupLinkInvalid) {
			return utils.SendError(c, fiber.StatusNotFound, constants.ErrInvalidInput)
		}
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"chat": entity})
	}
}
