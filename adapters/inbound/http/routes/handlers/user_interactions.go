package handlers

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/utils"
	"errors"

	"github.com/gofiber/fiber/v3"
)

func userInteractionError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domainuser.ErrInteractionBlocked), errors.Is(err, domainuser.ErrActorMismatch):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, domainuser.ErrInteractionUnavailable), errors.Is(err, ports.ErrNotFound):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
	case errors.Is(err, domainuser.ErrSelfInteraction):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
	default:
		return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
	}
}
