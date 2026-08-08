package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	domainmedia "core/domain/media"
	"core/utils"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func HandleFetchPrivatePhotos(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		ownerID := principal.PublicID
		if strings.TrimSpace(requestField(c, "owner_id")) != "" {
			var err error
			ownerID, err = privatePhotoPublicIDField(c, "owner_id")
			if err != nil {
				return privatePhotoError(c, err)
			}
		}
		result, err := service.Fetch(c.Context(), principal, ownerID)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Private photos fetched successfully")
	}
}

func HandleUploadPrivatePhotos(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidForm, "invalid private photo upload")
		}

		var headersFound bool
		var files []ports.UploadedFile
		for _, key := range []string{"photos[]", "images[]", "photos", "images", "photo", "image"} {
			headers := form.File[key]
			if len(headers) == 0 {
				continue
			}
			headersFound = true
			files = make([]ports.UploadedFile, 0, len(headers))
			for _, header := range headers {
				files = append(files, uploadedFile(header))
			}
			break
		}
		if !headersFound {
			return privatePhotoError(c, usecases.ErrPrivatePhotoFilesRequired)
		}

		photos, err := service.Upload(c.Context(), principal, files)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, fiber.Map{"photos": photos}, "Private photos uploaded successfully")
	}
}

func HandleDeletePrivatePhoto(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		photoID, err := privatePhotoPublicIDField(c, "photo_id")
		if err != nil {
			return privatePhotoError(c, err)
		}
		if err := service.Delete(c.Context(), principal, photoID); err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"photo_id": strconv.FormatInt(photoID, 10)}, "Private photo deleted successfully")
	}
}

func HandleFetchProfilePhotos(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		ownerID := principal.PublicID
		if strings.TrimSpace(requestField(c, "owner_id")) != "" {
			var err error
			ownerID, err = privatePhotoPublicIDField(c, "owner_id")
			if err != nil {
				return privatePhotoError(c, err)
			}
		}
		result, err := service.FetchProfilePhotos(c.Context(), principal, ownerID)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Profile photos fetched successfully")
	}
}

func HandleMoveProfilePhoto(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		photoID, err := privatePhotoPublicIDField(c, "photo_id")
		if err != nil {
			return privatePhotoError(c, err)
		}
		result, err := service.MoveProfilePhoto(c.Context(), principal, photoID, requestField(c, "destination"))
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Photo moved successfully")
	}
}

func HandleRequestPrivatePhotoAccess(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		ownerID, err := privatePhotoPublicIDField(c, "owner_id")
		if err != nil {
			return privatePhotoError(c, err)
		}
		access, err := service.RequestAccess(c.Context(), principal, ownerID)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"access": access}, "Private photo access request saved")
	}
}

func HandleFetchPrivatePhotoAccessRequests(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		requests, err := service.ListAccessRequests(c.Context(), principal)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"requests": requests}, "Private photo access requests fetched successfully")
	}
}

func HandleRespondPrivatePhotoAccess(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		requestID, err := privatePhotoPublicIDField(c, "request_id")
		if err != nil {
			return privatePhotoError(c, err)
		}
		access, err := service.RespondAccess(c.Context(), principal, requestID, requestField(c, "decision"))
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"access": access}, "Private photo access request updated")
	}
}

func HandleRevokePrivatePhotoAccess(service *usecases.PrivatePhotoService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := privatePhotoPrincipal(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		viewerID, err := privatePhotoPublicIDField(c, "viewer_id")
		if err != nil {
			return privatePhotoError(c, err)
		}
		access, err := service.RevokeAccess(c.Context(), principal, viewerID)
		if err != nil {
			return privatePhotoError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"access": access}, "Private photo access revoked")
	}
}

func privatePhotoPrincipal(c fiber.Ctx) (ports.PrivatePhotoPrincipal, bool) {
	user, ok := middleware.GetAuthenticatedUser(c)
	if !ok || user == nil {
		return ports.PrivatePhotoPrincipal{}, false
	}
	return ports.PrivatePhotoPrincipal{ID: user.ID, PublicID: user.PublicID}, true
}

func privatePhotoPublicIDField(c fiber.Ctx, name string) (int64, error) {
	value := strings.TrimSpace(requestField(c, name))
	publicID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || publicID <= 0 {
		switch name {
		case "owner_id":
			return 0, usecases.ErrPrivatePhotoOwnerRequired
		case "request_id":
			return 0, usecases.ErrPrivatePhotoRequestRequired
		case "viewer_id":
			return 0, usecases.ErrPrivatePhotoViewerRequired
		case "photo_id":
			return 0, usecases.ErrPrivatePhotoIDRequired
		default:
			return 0, usecases.ErrPrivatePhotoIDRequired
		}
	}
	return publicID, nil
}

func privatePhotoError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecases.ErrPrivatePhotoPrincipalRequired):
		return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
	case errors.Is(err, usecases.ErrPrivatePhotoUserNotFound):
		return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrUserNotFound, err.Error())
	case errors.Is(err, usecases.ErrPrivatePhotoNotFound), errors.Is(err, usecases.ErrPrivatePhotoRequestNotFound):
		return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrResourceNotFound, err.Error())
	case errors.Is(err, usecases.ErrPrivatePhotoForbidden), errors.Is(err, usecases.ErrPrivatePhotoSelfRequest):
		return utils.SendErrorWithMessage(c, fiber.StatusForbidden, constants.ErrPermissionDenied, err.Error())
	case errors.Is(err, usecases.ErrPrivatePhotoOwnerLimit):
		return utils.SendErrorWithMessage(c, fiber.StatusConflict, constants.ErrInvalidAction, err.Error())
	case errors.Is(err, usecases.ErrPrivatePhotoOwnerRequired),
		errors.Is(err, usecases.ErrPrivatePhotoRequestRequired),
		errors.Is(err, usecases.ErrPrivatePhotoViewerRequired),
		errors.Is(err, usecases.ErrPrivatePhotoIDRequired),
		errors.Is(err, usecases.ErrPrivatePhotoFilesRequired),
		errors.Is(err, usecases.ErrPrivatePhotoUploadLimit),
		errors.Is(err, usecases.ErrPrivatePhotoImageRequired),
		errors.Is(err, usecases.ErrPrivatePhotoImageDimensions),
		errors.Is(err, usecases.ErrPhotoDestinationRequired),
		errors.Is(err, domainmedia.ErrInvalidPrivatePhotoAccessStatus),
		errors.Is(err, domainmedia.ErrInvalidPrivatePhotoAccessTransition):
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
	default:
		log.Printf("private photo API error action=%q: %v", strings.TrimSpace(c.Get("X-Action")), err)
		return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
	}
}
