package middleware

import (
	"context"
	"net/http"
	"strings"

	"bifrost/helpers"
	"bifrost/models/user"
	"bifrost/repositories"
)

type contextKey string

const userContextKey = contextKey("authenticatedUser")

type Middleware func(http.HandlerFunc) http.HandlerFunc

func AuthMiddleware(userRepo *repositories.UserRepository) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims, err := helpers.DecodeUserJWT(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			u, err := userRepo.GetUserByPublicId(claims.PublicID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, u)
			next(w, r.WithContext(ctx))
		}
	}
}

func AuthMiddlewareWithoutCheck(userRepo *repositories.UserRepository) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString := parts[1]
					claims, err := helpers.DecodeUserJWT(tokenString)
					if err == nil {
						u, err := userRepo.GetUserByPublicId(claims.PublicID)
						if err == nil {
							ctx := context.WithValue(r.Context(), userContextKey, u)
							next(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			ctx := context.WithValue(r.Context(), userContextKey, nil)
			next(w, r.WithContext(ctx))
		}
	}
}
func GetAuthenticatedUser(r *http.Request) (*user.User, bool) {
	u, ok := r.Context().Value(userContextKey).(*user.User)
	return u, ok
}
