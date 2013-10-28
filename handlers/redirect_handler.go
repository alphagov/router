package handlers

import (
	"net/http"
)

func NewRedirectHandler(path string, temporary bool) http.Handler {
	statusMoved := http.StatusMovedPermanently
	if temporary {
		statusMoved = http.StatusFound
	}
	return http.RedirectHandler(path, statusMoved)
}
