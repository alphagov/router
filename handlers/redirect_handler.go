package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router-postgres/logger"
)

const (
	cacheDuration = 30 * time.Minute

	redirectHandlerType               = "redirect-handler"
	pathPreservingRedirectHandlerType = "path-preserving-redirect-handler"
)

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

func addGAQueryParam(target string, request *http.Request) string {
	if ga := request.URL.Query().Get("_ga"); ga != "" {
		u, err := url.Parse(target)
		if err != nil {
			defer logger.NotifySentry(logger.ReportableError{Error: err, Request: request})
			return target
		}
		values := u.Query()
		values.Set("_ga", ga)
		u.RawQuery = values.Encode()
		return u.String()
	}
	return target
}

type redirectHandler struct {
	url  string
	code int
}

func (handler *redirectHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	addCacheHeaders(writer)

	target := addGAQueryParam(handler.url, request)
	http.Redirect(writer, request, target, handler.code)

	RedirectHandlerRedirectCountMetric.With(prometheus.Labels{
		"redirect_code": fmt.Sprintf("%d", handler.code),
		"redirect_type": redirectHandlerType,
	}).Inc()
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

	RedirectHandlerRedirectCountMetric.With(prometheus.Labels{
		"redirect_code": fmt.Sprintf("%d", handler.code),
		"redirect_type": pathPreservingRedirectHandlerType,
	}).Inc()
}
