package middleware

import (
	"fmt"
	"strings"

	"core/helpers"
	"core/models"
	"core/repositories"

	"github.com/gofiber/fiber/v3"
)

const userContextKey = "authenticatedUser"

type Middleware func(fiber.Handler) fiber.Handler

func GetClientIP(c fiber.Ctx) string {
	headers := []string{
		"CF-Connecting-IP",
		"X-Forwarded-For",
		"X-Real-IP",
	}

	for _, h := range headers {
		ip := c.Get(h)
		if ip != "" {
			if strings.Contains(ip, ",") {
				parts := strings.Split(ip, ",")
				return strings.TrimSpace(parts[0])
			}
			return strings.TrimSpace(ip)
		}
	}

	return c.IP()
}

func AuthMiddleware(userRepo *repositories.UserRepository) func(fiber.Handler) fiber.Handler {
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

			claims, err := helpers.DecodeUserJWT(tokenString)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired token")
			}

			u, err := userRepo.GetUserByPublicId(claims.PublicID)

			userIP := GetClientIP(c)
			fmt.Println("UserIP", userIP)

			updateIPErr := userRepo.UpdateLocation(c.Context(), u, userIP)
			if updateIPErr != nil {
				fmt.Println("UPDATE_IP_FAILED", updateIPErr)
			}

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
		return func(c fiber.Ctx) error {
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

func GetAuthenticatedUser(c fiber.Ctx) (*models.User, bool) {
	u := c.Locals(userContextKey)
	if u == nil {
		return nil, false
	}

	user, ok := u.(*models.User)
	return user, ok
}
