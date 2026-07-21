package routes

import (
	"core/application/ports"
	"core/application/usecases"
	"errors"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var mediaVariantSuffix = regexp.MustCompile(`_(?:(?:square|landscape|portrait)_(?:icon|thumb|sm|md|lg)|(?:poster|low|medium|high|preview))$`)

func handleMediaFile(service *usecases.MediaAccessService, tokenDecoder ports.UserTokenDecoder) fiber.Handler {
	return func(c fiber.Ctx) error {
		requestedPath, originalPrefix, ok := mediaStoragePath(c.Path())
		if !ok || service == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		decision, err := service.Authorize(c.Context(), originalPrefix, mediaViewerPublicID(c, tokenDecoder))
		if errors.Is(err, ports.ErrNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		if !decision.Allowed {
			return c.SendStatus(fiber.StatusNotFound)
		}

		if decision.Public {
			c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
		} else {
			c.Set(fiber.HeaderCacheControl, "private, no-store")
		}
		return c.SendFile(requestedPath)
	}
}

func mediaViewerPublicID(c fiber.Ctx, decoder ports.UserTokenDecoder) *int64 {
	if decoder == nil {
		return nil
	}
	authHeader := c.Get(fiber.HeaderAuthorization)
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return nil
	}
	publicID, err := decoder.DecodeUserPublicID(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil
	}
	return &publicID
}

func mediaStoragePath(requestPath string) (requestedPath, originalPrefix string, ok bool) {
	cleanURL := pathpkg.Clean("/" + strings.TrimSpace(requestPath))
	const uploadPrefix = "/static/uploads/"
	if !strings.HasPrefix(cleanURL, uploadPrefix) {
		return "", "", false
	}

	relative := strings.TrimPrefix(cleanURL, uploadPrefix)
	if relative == "" || relative == "." || strings.HasPrefix(relative, "../") {
		return "", "", false
	}
	requestedPath = filepath.Clean(filepath.FromSlash("." + uploadPrefix + relative))

	extension := filepath.Ext(requestedPath)
	stem := strings.TrimSuffix(filepath.Base(requestedPath), extension)
	stem = mediaVariantSuffix.ReplaceAllString(stem, "")
	if _, err := uuid.Parse(stem); err != nil {
		return "", "", false
	}
	originalPrefix = filepath.ToSlash(filepath.Join(filepath.Dir(requestedPath), stem))
	if !strings.HasPrefix(originalPrefix, "static/uploads/") {
		return "", "", false
	}
	originalPrefix = "./" + originalPrefix
	return requestedPath, originalPrefix, true
}
