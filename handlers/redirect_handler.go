package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router/logger"
)

const (
	cacheDuration = 30 * time.Minute

	redirectHandlerType               = "redirect-handler"
	pathPreservingRedirectHandlerType = "path-preserving-redirect-handler"
	downcaseRedirectHandlerType       = "downcase-redirect-handler"
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

func addCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Now().Add(cacheDuration).Format(time.RFC1123))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", cacheDuration/time.Second))
}

func addGAQueryParam(target string, r *http.Request) string {
	if ga := r.URL.Query().Get("_ga"); ga != "" {
		u, err := url.Parse(target)
		if err != nil {
			defer logger.NotifySentry(logger.ReportableError{Error: err, Request: r})
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

func (handler *redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCacheHeaders(w)

	target := addGAQueryParam(handler.url, r)
	http.Redirect(w, r, target, handler.code)

	redirectCountMetric.With(prometheus.Labels{
		"redirect_code": fmt.Sprintf("%d", handler.code),
		"redirect_type": redirectHandlerType,
	}).Inc()
}

type pathPreservingRedirectHandler struct {
	sourcePrefix string
	targetPrefix string
	code         int
}

func (handler *pathPreservingRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := handler.targetPrefix + strings.TrimPrefix(r.URL.Path, handler.sourcePrefix)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	addCacheHeaders(w)
	http.Redirect(w, r, target, handler.code)

	redirectCountMetric.With(prometheus.Labels{
		"redirect_code": fmt.Sprintf("%d", handler.code),
		"redirect_type": pathPreservingRedirectHandlerType,
	}).Inc()
}

type downcaseRedirectHandler struct{}

func NewDowncaseRedirectHandler() http.Handler {
	return &downcaseRedirectHandler{}
}

func (handler *downcaseRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const status = http.StatusMovedPermanently

	target := strings.ToLower(r.URL.Path)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	addCacheHeaders(w)
	http.Redirect(w, r, target, status)

	redirectCountMetric.With(prometheus.Labels{
		"redirect_code": fmt.Sprintf("%d", status),
		"redirect_type": downcaseRedirectHandlerType,
	}).Inc()
}
