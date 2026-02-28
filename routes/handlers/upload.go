package handlers

import (
	"core/constants"
	"core/models/media"
	services "core/services/user"
	"core/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type UploadHandler struct {
	service *services.MediaService
}

func NewUploadHandler(service *services.MediaService) *UploadHandler {
	return &UploadHandler{service: service}
}

func (h *UploadHandler) HandleUploadMedia(s *services.MediaService) fiber.Handler {
	return func(c fiber.Ctx) error {
		ownerIDStr := c.FormValue("owner_id")
		ownerTypeStr := c.FormValue("owner_type")
		roleStr := c.FormValue("role")

		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid owner ID")
		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid form data or no file uploaded")
		}

		files := form.File["file"]
		if len(files) == 0 {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "No file uploaded")
		}

		media, err := s.AddMedia(ownerID, media.OwnerType(ownerTypeStr), ownerID, media.MediaRole(roleStr), files[0])
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed, "Media upload failed")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"media": media,
		}, "Media uploaded successfully")
	}
}
