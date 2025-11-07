package utils

import (
	"coolvibes/constants"
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Success bool                `json:"success"`
	Code    constants.ErrorCode `json:"code"`
	Message string              `json:"message"`
}

func SendError(w http.ResponseWriter, status int, code constants.ErrorCode) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Code:    code,
		Message: code.String(),
	})
}

func SendJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
