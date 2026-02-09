package handlers

import (
	"core/constants"
	"core/models/media"
	services "core/services/user"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UploadHandler struct {
	service *services.MediaService
}

func HandleUploadMedia(s *services.MediaService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ownerIDStr := c.FormValue("owner_id")
		ownerTypeStr := c.FormValue("owner_type")
		roleStr := c.FormValue("role")

		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid owner_id",
			})
		}

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid form data or no file uploaded",
			})
		}

		files := form.File["file"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No file uploaded",
			})
		}

		media, err := s.AddMedia(ownerID, media.OwnerType(ownerTypeStr), ownerID, media.MediaRole(roleStr), files[0])
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		return c.Status(fiber.StatusOK).JSON(media)
	}
}
