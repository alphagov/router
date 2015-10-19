package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const cacheDuration = 30 * time.Minute

func NewRedirectHandler(source, target string, preserve bool, temporary bool) http.Handler {
	statusMoved := http.StatusMovedPermanently
	if temporary {
		statusMoved = http.StatusFound
	}
	if preserve {
		return &pathPreservingRedirectHandler{source, target, statusMoved}
	}
	return &redirectHandler{target, statusMoved}
}

func addCacheHeaders(writer http.ResponseWriter) {
	writer.Header().Set("Expires", time.Now().Add(cacheDuration).Format(time.RFC1123))
	writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", cacheDuration/time.Second))
}

type redirectHandler struct {
	url  string
	code int
}

func (handler *redirectHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	addCacheHeaders(writer)
	http.Redirect(writer, request, handler.url, handler.code)
}

type pathPreservingRedirectHandler struct {
	sourcePrefix string
	targetPrefix string
	code         int
}

func (handler *pathPreservingRedirectHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	target := handler.targetPrefix + strings.TrimPrefix(request.URL.Path, handler.sourcePrefix)
	if request.URL.RawQuery != "" {
		target = target + "?" + request.URL.RawQuery
	}

	addCacheHeaders(writer)
	http.Redirect(writer, request, target, handler.code)
}
