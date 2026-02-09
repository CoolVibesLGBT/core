package middleware

import (
	"strings"

	"core/helpers"
	"core/models"
	"core/repositories"

	"github.com/gofiber/fiber/v2"
)

const userContextKey = "authenticatedUser"

type Middleware func(fiber.Handler) fiber.Handler

func AuthMiddleware(userRepo *repositories.UserRepository) func(fiber.Handler) fiber.Handler {
	return func(next fiber.Handler) fiber.Handler {
		return func(c *fiber.Ctx) error {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				return c.Status(fiber.StatusUnauthorized).SendString("Missing Authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid Authorization header")
			}

			tokenString := parts[1]

			claims, err := helpers.DecodeUserJWT(tokenString)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired token")
			}

			u, err := userRepo.GetUserByPublicId(claims.PublicID)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("User not found")
			}

			c.Locals(userContextKey, u)

			return next(c)
		}
	}
}

func AuthMiddlewareWithoutCheck(userRepo *repositories.UserRepository) func(fiber.Handler) fiber.Handler {
	return func(next fiber.Handler) fiber.Handler {
		return func(c *fiber.Ctx) error {
			authHeader := c.Get("Authorization")

			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString := parts[1]

					claims, err := helpers.DecodeUserJWT(tokenString)
					if err == nil {
						u, err := userRepo.GetUserByPublicId(claims.PublicID)
						if err == nil {
							c.Locals(userContextKey, u)
							return next(c)
						}
					}
				}
			}

			// User yok → nil koyuyoruz
			c.Locals(userContextKey, nil)

			return next(c)
		}
	}
}

func GetAuthenticatedUser(c *fiber.Ctx) (*models.User, bool) {
	u := c.Locals(userContextKey)
	if u == nil {
		return nil, false
	}

	user, ok := u.(*models.User)
	return user, ok
}
