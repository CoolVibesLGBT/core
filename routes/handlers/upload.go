package handlers

import (
	"bifrost/constants"
	"bifrost/models/media"
	services "bifrost/services/user"
	"bifrost/utils"
	"net/http"

	"github.com/google/uuid"
)

type UploadHandler struct {
	service *services.MediaService
}

func HandleUploadMedia(s *services.MediaService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(8 << 30); err != nil {
			utils.SendError(w, http.StatusBadRequest, "Invalid form data")
			return
		}

		ownerIDStr := r.FormValue("owner_id")
		ownerTypeStr := r.FormValue("owner_type")
		roleStr := r.FormValue("role")

		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, "Invalid owner_id")
			return
		}

		fileHeader, ok := r.MultipartForm.File["file"]
		if !ok || len(fileHeader) == 0 {
			utils.SendError(w, http.StatusBadRequest, "No file uploaded")
			return
		}

		media, err := s.AddMedia(ownerID, media.OwnerType(ownerTypeStr), ownerID, media.MediaRole(roleStr), fileHeader[0])
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, constants.ErrMediaUploadFailed)
			return
		}

		utils.SendJSON(w, http.StatusOK, media)
	}
}
