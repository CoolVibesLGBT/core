package handlers

import (
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"core/utils"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func HandleFriendshipMutation(service *usecases.FriendshipService, action string) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		field := "request_id"
		if action == constants.CMD_USER_FRIEND_REQUEST {
			field = "user_id"
		}
		id, err := strconv.ParseInt(requestField(c, field), 10, 64)
		if err != nil || id <= 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		var result usecases.FriendshipView
		if action == constants.CMD_USER_FRIEND_REQUEST {
			result, err = service.Request(c.Context(), principal, id)
		} else {
			var next domainuser.FriendshipStatus
			switch action {
			case constants.CMD_USER_FRIEND_RESPOND:
				next = domainuser.FriendshipStatus(requestField(c, "decision"))
				if next != domainuser.FriendshipAccepted && next != domainuser.FriendshipDenied {
					return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
				}
			case constants.CMD_USER_FRIEND_CANCEL:
				next = domainuser.FriendshipCancelled
			case constants.CMD_USER_FRIEND_REMOVE:
				next = domainuser.FriendshipRemoved
			default:
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
			}
			result, err = service.Change(c.Context(), principal, id, next)
		}
		if err != nil {
			return friendshipError(c, err)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"friendship": result})
	}
}

func HandleFriendshipList(service *usecases.FriendshipService, requests bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		scope := domainuser.FriendshipScopeFriends
		if requests {
			scope = domainuser.FriendshipScope(requestField(c, "scope"))
			if scope == "" {
				scope = domainuser.FriendshipScopeIncoming
			}
			if scope != domainuser.FriendshipScopeIncoming && scope != domainuser.FriendshipScopeOutgoing {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
			}
		}
		cursor, err := parseMatchListCursor(requestField(c, "cursor"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		limit := 10
		if raw := requestField(c, "limit"); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil || limit <= 0 {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
			}
		}
		rows, next, err := service.List(c.Context(), principal, scope, cursor, limit)
		if err != nil {
			return friendshipError(c, err)
		}
		_, token, err := encodeSocialCursorPair(nil, next)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"friendships": rows, "cursor": token})
	}
}
func friendshipError(c fiber.Ctx, err error) error {
	if errors.Is(err, domainuser.ErrFriendshipForbidden) {
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	}
	if errors.Is(err, domainuser.ErrFriendshipTransition) {
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
	}
	return userInteractionError(c, err)
}
