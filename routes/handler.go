package routes

import (
	"net/http"
)

type AppHandler interface {
	// Burada app paketinin kullanabileceği methodlar yer alacak
	// Örnek:
	ServeHTTP(http.ResponseWriter, *http.Request)
}
