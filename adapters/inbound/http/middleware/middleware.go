package middleware

import (
	"fmt"
	"strings"

	"core/application/usecases"
	"core/models"

	"github.com/gofiber/fiber/v3"
)

const userContextKey = "authenticatedUser"

type Middleware func(fiber.Handler) fiber.Handler

func GetClientIP(c fiber.Ctx) string {
	// Fiber consults forwarded headers only when the direct peer matches its
	// explicit trusted-proxy configuration. Reading headers here would allow a
	// client to spoof the permanent profile location.
	return c.IP()
}

func AuthMiddleware(session *usecases.SessionService) func(fiber.Handler) fiber.Handler {
	return func(next fiber.Handler) fiber.Handler {
		return func(c fiber.Ctx) error {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				return c.Status(fiber.StatusUnauthorized).SendString("Missing Authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid Authorization header")
			}

			tokenString := parts[1]

			resolved, err := session.ResolveUser(c.Context(), tokenString)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid session")
			}

			userIP := GetClientIP(c)

			updateIPErr := session.TrackLocation(c.Context(), resolved, userIP)
			if updateIPErr != nil {
				fmt.Println("UPDATE_IP_FAILED", updateIPErr)
			}

			c.Locals(userContextKey, resolved.User)

			return next(c)
		}
	}
}

func AuthMiddlewareWithoutCheck(session *usecases.SessionService) func(fiber.Handler) fiber.Handler {
	return func(next fiber.Handler) fiber.Handler {
		return func(c fiber.Ctx) error {
			authHeader := c.Get("Authorization")

			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString := parts[1]

					resolved, err := session.ResolveUser(c.Context(), tokenString)
					if err == nil {
						userIP := GetClientIP(c)

						updateIPErr := session.TrackLocation(c.Context(), resolved, userIP)
						if updateIPErr != nil {
							fmt.Println("UPDATE_IP_FAILED", updateIPErr)
						}
						c.Locals(userContextKey, resolved.User)
						return next(c)
					}
				}
			}

			// User yok → nil koyuyoruz
			c.Locals(userContextKey, nil)

			return next(c)
		}
	}
}

func GetAuthenticatedUser(c fiber.Ctx) (*models.User, bool) {
	u := c.Locals(userContextKey)
	if u == nil {
		return nil, false
	}

	user, ok := u.(*models.User)
	return user, ok
}
