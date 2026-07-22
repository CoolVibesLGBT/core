package handlers

import (
	"context"
	"core/application/ports"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestProfileUpdateHTTPErrorDoesNotExposeInfrastructureFailures(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    constants.ErrorCode
		wantMessage bool
	}{
		{name: "username collision", err: usecases.ErrUsernameAlreadyExists, wantStatus: fiber.StatusConflict, wantCode: constants.ErrUsernameTaken, wantMessage: true},
		{name: "email collision", err: usecases.ErrEmailAlreadyExists, wantStatus: fiber.StatusConflict, wantCode: constants.ErrInvalidInput, wantMessage: true},
		{name: "invalid current password", err: usecases.ErrInvalidCurrentPassword, wantStatus: fiber.StatusUnauthorized, wantCode: constants.ErrInvalidPassword},
		{name: "missing user", err: ports.ErrNotFound, wantStatus: fiber.StatusNotFound, wantCode: constants.ErrUserNotFound},
		{name: "deadline", err: context.DeadlineExceeded, wantStatus: fiber.StatusGatewayTimeout, wantCode: constants.ErrRequestTimeout},
		{name: "validation", err: domainuser.ErrInvalidWebsite, wantStatus: fiber.StatusBadRequest, wantCode: constants.ErrInvalidInput, wantMessage: true},
		{name: "database", err: errors.New("secret database connection failed"), wantStatus: fiber.StatusInternalServerError, wantCode: constants.ErrDatabaseError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, code, message := profileUpdateHTTPError(test.err)
			if status != test.wantStatus || code != test.wantCode || (message != "") != test.wantMessage {
				t.Fatalf("profileUpdateHTTPError() = (%d, %s, %q); want (%d, %s, message=%v)", status, code, message, test.wantStatus, test.wantCode, test.wantMessage)
			}
			if test.name == "database" && message != "" {
				t.Fatalf("database error leaked to client: %q", message)
			}
		})
	}
}
