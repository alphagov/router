package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const CachePeriod = 24 * time.Hour

func NewRedirectHandler(sourcePath, targetPath string, prefix, temporary bool) http.Handler {
	statusMoved := http.StatusMovedPermanently
	if temporary {
		statusMoved = http.StatusFound
	}
	if prefix {
		return &pathPreservingRedirectHandler{sourcePath, targetPath, statusMoved}
	}
	return http.RedirectHandler(targetPath, statusMoved)
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

	w.Header().Set("Expires", time.Now().Add(CachePeriod).Format(time.RFC1123))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", CachePeriod / time.Second))

	http.Redirect(w, r, target, rh.code)
}
