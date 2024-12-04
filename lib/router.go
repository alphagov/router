package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router/logger"
	"github.com/alphagov/router/triemux"
)

const (
	RouteTypePrefix = "prefix"
	RouteTypeExact  = "exact"
)

const (
	SegmentsModePreserve = "preserve"
	SegmentsModeIgnore   = "ignore"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a postgres database.
type Router struct {
	backends   map[string]http.Handler
	mux        *triemux.Mux
	lock       sync.RWMutex
	logger     logger.Logger
	opts       Options
	ReloadChan chan bool
	pool       *pgxpool.Pool
}

type Options struct {
	BackendConnTimeout   time.Duration
	BackendHeaderTimeout time.Duration
	LogFileName          string
}

// RegisterMetrics registers Prometheus metrics from the router module and the
// modules that it directly depends on. To use the default (global) registry,
// pass prometheus.DefaultRegisterer.
func RegisterMetrics(r prometheus.Registerer) {
	registerMetrics(r)
}

func NewRouter(o Options) (rt *Router, err error) {
	l, err := logger.New(o.LogFileName)
	if err != nil {
		return nil, err
	}
	logInfo("router: logging errors as JSON to", o.LogFileName)

	backends := loadBackendsFromEnv(o.BackendConnTimeout, o.BackendHeaderTimeout, l)

	var pool *pgxpool.Pool

	pool, err = pgxpool.New(context.Background(), os.Getenv("CONTENT_STORE_DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	logInfo("router: postgres connection pool created")

	ReloadChan := make(chan bool, 1)
	rt = &Router{
		backends:   backends,
		mux:        triemux.NewMux(),
		logger:     l,
		opts:       o,
		ReloadChan: ReloadChan,
		pool:       pool,
	}

	rt.reloadRoutes(pool)

	go func() {
		if err := rt.listenForContentStoreUpdates(context.Background()); err != nil {
			logWarn(fmt.Sprintf("router: error in listenForContentStoreUpdates: %v", err))
		}
	}()

	go rt.waitForReload()

	return rt, nil
}

// ServeHTTP delegates responsibility for serving requests to the proxy mux
// instance for this router.
func (rt *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			logWarn("router: recovered from panic in ServeHTTP:", r)

			errorMessage := fmt.Sprintf("panic: %v", r)
			err := logger.RecoveredError{ErrorMessage: errorMessage}

			logger.NotifySentry(logger.ReportableError{Error: err, Request: req})
			rt.logger.LogFromClientRequest(map[string]interface{}{
				"error":  errorMessage,
				"status": http.StatusInternalServerError,
			}, req)

			w.WriteHeader(http.StatusInternalServerError)

			internalServerErrorCountMetric.With(prometheus.Labels{"host": req.Host}).Inc()
		}
	}()

	var mux *triemux.Mux

	rt.lock.RLock()
	mux = rt.mux
	rt.lock.RUnlock()

	mux.ServeHTTP(w, req)
}

func shouldPreserveSegments(routeType, segmentsMode string) bool {
	switch routeType {
	case RouteTypeExact:
		return segmentsMode == SegmentsModePreserve
	case RouteTypePrefix:
		return segmentsMode != SegmentsModeIgnore
	default:
		return false
	}
}
