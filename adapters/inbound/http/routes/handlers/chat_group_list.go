package handlers

import (
	"core/application/ports"
	"core/constants"
	"core/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"strings"
)

func HandleListPublicGroups(s ports.ChatGroupList) fiber.Handler {
	return func(c fiber.Ctx) error {
		limit, err := parseChatPageLimit(c)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		query := ports.PublicGroupQuery{Search: strings.TrimSpace(requestField(c, "query")), Limit: limit}
		if raw := strings.TrimSpace(requestField(c, "cursor")); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
			}
			query.Cursor = &id
		}
		groups, cursor, err := s.ListPublicGroups(c.Context(), query)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{"groups": groups, "cursor": cursor})
	}
}
