package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const cacheDuration = 30 * time.Minute

func NewRedirectHandler(source, target string, prefix, temporary bool) http.Handler {
	statusMoved := http.StatusMovedPermanently
	if temporary {
		statusMoved = http.StatusFound
	}
	if prefix {
		return &pathPreservingRedirectHandler{source, target, statusMoved}
	}
	return &redirectHandler{target, statusMoved}
}

func addCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Now().Add(cacheDuration).Format(time.RFC1123))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", cacheDuration/time.Second))
}

type redirectHandler struct {
	url  string
	code int
}

func (rh *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCacheHeaders(w)
	http.Redirect(w, r, rh.url, rh.code)
}

type pathPreservingRedirectHandler struct {
	sourcePrefix string
	targetPrefix string
	code         int
}

func (rh *pathPreservingRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := rh.targetPrefix + strings.TrimPrefix(r.URL.Path, rh.sourcePrefix)
	if r.URL.RawQuery != "" {
		target = target + "?" + r.URL.RawQuery
	}

	addCacheHeaders(w)
	http.Redirect(w, r, target, rh.code)
}
