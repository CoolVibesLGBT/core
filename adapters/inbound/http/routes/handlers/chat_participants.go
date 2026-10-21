package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/models/chat"
	"core/utils"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func sendChatParticipantError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, chat.ErrNotParticipant), errors.Is(err, chat.ErrPermissionDenied), errors.Is(err, domainuser.ErrInteractionBlocked), errors.Is(err, domainuser.ErrInteractionUnavailable):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, chat.ErrChatNotFound):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrNotFound)
	case errors.Is(err, constants.ErrUserNotFound):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
	case errors.Is(err, constants.ErrSelfChatNotAllowed):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrSelfChatNotAllowed)
	case errors.Is(err, constants.ErrInvalidParticipantsLength):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantsLength)
	case errors.Is(err, constants.ErrUnsupportedChatType):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUnsupportedChatType)
	case errors.Is(err, constants.ErrInvalidParticipantID):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantID)
	case errors.Is(err, constants.ErrGroupOwnerCannotLeave):
		return utils.SendError(c, fiber.StatusConflict, constants.ErrGroupOwnerCannotLeave)
	default:
		return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
	}
}

func HandleAddChatParticipant(s ports.ChatUseCase) fiber.Handler {
	return handleChatParticipantCommand(s, constants.CMD_CHAT_PARTICIPANT_ADD)
}
func HandleRemoveChatParticipant(s ports.ChatUseCase) fiber.Handler {
	return handleChatParticipantCommand(s, constants.CMD_CHAT_PARTICIPANT_REMOVE)
}
func HandleGetChatParticipants(s ports.ChatUseCase) fiber.Handler {
	return handleChatParticipantCommand(s, constants.CMD_CHAT_PARTICIPANTS_FETCH)
}

func handleChatParticipantCommand(s ports.ChatUseCase, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		actor, ok := middleware.GetAuthenticatedUser(c)
		if !ok || actor == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		chatID, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil || chatID == uuid.Nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		identifier := c.FormValue("participant_id")
		switch action {
		case constants.CMD_CHAT_PARTICIPANT_ADD:
			err = s.AddChatParticipant(c.Context(), chatID, identifier, actor)
		case constants.CMD_CHAT_PARTICIPANT_REMOVE:
			err = s.RemoveChatParticipant(c.Context(), chatID, identifier, actor)
		case constants.CMD_CHAT_PARTICIPANTS_FETCH:
			participants, err := s.GetChatParticipants(c.Context(), chatID, actor)
			if err != nil {
				return sendChatParticipantError(c, err)
			}
			return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"participants": participants})
		}
		if err != nil {
			return sendChatParticipantError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"success": true})
	}
}
