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
	"github.com/rs/zerolog"

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
	backends              map[string]http.Handler
	mux                   *triemux.Mux
	lock                  sync.RWMutex
	opts                  Options
	ReloadChan            chan bool
	pool                  *pgxpool.Pool
	lastAttemptReloadTime time.Time
	Logger                zerolog.Logger
}

type Options struct {
	BackendConnTimeout   time.Duration
	BackendHeaderTimeout time.Duration
	Logger               zerolog.Logger
	RouteReloadInterval  time.Duration
}

// RegisterMetrics registers Prometheus metrics from the router module and the
// modules that it directly depends on. To use the default (global) registry,
// pass prometheus.DefaultRegisterer.
func RegisterMetrics(r prometheus.Registerer) {
	registerMetrics(r)
}

func NewRouter(o Options) (rt *Router, err error) {
	backends := loadBackendsFromEnv(o.BackendConnTimeout, o.BackendHeaderTimeout, o.Logger)

	var pool *pgxpool.Pool

	pool, err = pgxpool.New(context.Background(), os.Getenv("CONTENT_STORE_DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	o.Logger.Info().Msg("postgres connection pool created")

	reloadChan := make(chan bool, 1)
	rt = &Router{
		backends:   backends,
		mux:        triemux.NewMux(o.Logger),
		Logger:     o.Logger,
		opts:       o,
		ReloadChan: reloadChan,
		pool:       pool,
	}

	rt.reloadRoutes(pool)

	go func() {
		if err := rt.listenForContentStoreUpdates(context.Background()); err != nil {
			rt.Logger.Error().Err(err).Msg("failed to listen for content store updates")
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
			rt.Logger.Err(fmt.Errorf("%v", r)).Msgf("recovered from panic in ServeHTTP")

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
