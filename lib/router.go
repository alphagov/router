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
// routes from a postgres database (content-store)
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

// Additional configurable options
type Options struct {
	BackendConnTimeout        time.Duration
	BackendHeaderTimeout      time.Duration
	Logger                    zerolog.Logger
	RouteReloadInterval       time.Duration
	EnableContentStoreUpdates bool
}

// RegisterMetrics registers Prometheus metrics from the router module and the
// modules that it directly depends on. To use the default (global) registry,
// pass prometheus.DefaultRegisterer.
func RegisterMetrics(r prometheus.Registerer) {
	registerMetrics(r)
}

/*
Creates an instance of Router struct which:
1. Loads routes from file or content-store database
2. Sets up a channel that will be used to send reload requests
3. If enabled starts goroutine to listen for content-store updates
4. Starts goroutine to listen for reload requests
*/
func NewRouter(o Options) (rt *Router, err error) {
	// Generate a map of backend handlers for configured backends
	backends := loadBackendsFromEnv(o.BackendConnTimeout, o.BackendHeaderTimeout, o.Logger)

	// Load routes from a flat file
	routesFile := os.Getenv("ROUTER_ROUTES_FILE")
	if routesFile != "" {
		o.Logger.Info().Str("file", routesFile).Msg("loading routes from flat file")

		mux := triemux.NewMux(o.Logger)
		err = loadRoutesFromFile(routesFile, mux, backends, o.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to load routes from file: %w", err)
		}

		routeCount := mux.RouteCount()
		o.Logger.Info().Int("route_count", routeCount).Msg("loaded routes from file")
		routesCountMetric.WithLabelValues("file").Set(float64(routeCount))

		// No ReloadChan or pool when using flat file
		rt = &Router{
			backends: backends,
			mux:      mux,
			Logger:   o.Logger,
			opts:     o,
		}

		return rt, nil
	}

	// Load routes from PostgreSQL
	var pool *pgxpool.Pool

	pool, err = pgxpool.New(context.Background(), os.Getenv("CONTENT_STORE_DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	o.Logger.Info().Msg("postgres connection pool created")

	/*
		Setup channel which Router's API server, content-store LISTEN/NOTIFY,and periodic route updates will use
		to send messages which will trigger reload of routes.
	*/
	reloadChan := make(chan bool, 1)

	// Create instance of Router
	rt = &Router{
		backends:   backends,
		mux:        triemux.NewMux(o.Logger),
		Logger:     o.Logger,
		opts:       o,
		ReloadChan: reloadChan,
		pool:       pool,
	}

	// Trigger a reload of routes from content-store
	rt.reloadRoutes(pool)

	// Start goroutine to listen for content-store changes
	if o.EnableContentStoreUpdates {
		rt.Logger.Info().Msg("content store updates enabled")
		go func() {
			if err := rt.listenForContentStoreUpdates(context.Background()); err != nil {
				rt.Logger.Error().Err(err).Msg("failed to listen for content store updates")
			}
		}()
	} else {
		rt.Logger.Info().Msg("content store updates are disabled")
	}

	// Start goroutine to listen on Router's channel
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
	// Retrieve the mux object from router
	mux = rt.mux
	rt.lock.RUnlock()

	mux.ServeHTTP(w, req)
}

// Determines whether the URL path in a redirect route should be preserved
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
