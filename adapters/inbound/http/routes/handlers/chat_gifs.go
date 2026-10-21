package handlers

import (
	"core/application/ports"
	"core/constants"
	"core/utils"
	"github.com/gofiber/fiber/v3"
	"os"
	"path/filepath"
	"regexp"
)

var telegramGIFFilename = regexp.MustCompile(`^[a-f0-9]{64}\.(mp4|gif)$`)

func HandleTelegramGIFMedia(media ports.ChatGIFMedia) fiber.Handler {
	return func(c fiber.Ctx) error {
		name := c.Params("file")
		if !telegramGIFFilename.MatchString(name) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		path := filepath.Join("static", "telegram", "gifs", name)
		if _, err := os.Stat(path); err != nil {
			if media == nil || media.PrepareGIF(c.Context(), name) != nil {
				return c.SendStatus(fiber.StatusNotFound)
			}
		}
		c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
		return c.SendFile(path)
	}
}

func HandleSearchChatGIFs(search ports.ChatGIFSearch) fiber.Handler {
	return func(c fiber.Ctx) error {
		query, offset := c.FormValue("query"), c.FormValue("offset")
		if len(query) > 200 || len(offset) > 1024 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if search == nil {
			return utils.SendError(c, fiber.StatusServiceUnavailable, constants.ErrChatGIFUnavailable)
		}
		page, err := search.SearchGIFs(c.Context(), query, offset)
		if err != nil {
			return utils.SendError(c, fiber.StatusServiceUnavailable, constants.ErrChatGIFUnavailable)
		}
		return utils.SendSuccess(c, fiber.StatusOK, page)
	}
}
