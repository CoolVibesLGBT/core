package middleware

import (
	"coolvibes/constants"
	"coolvibes/utils"
	"log"
	"net/http"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC recovered: %v\n", err)
				utils.SendError(w, http.StatusInternalServerError, constants.ErrUnknown)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
