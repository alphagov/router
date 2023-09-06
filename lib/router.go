package router

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/logger"
	"github.com/alphagov/router/triemux"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
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
// routes from a passed mongo database.
//
// TODO: decouple Router from its database backend. Router should not know
// anything about the database backend. Its representation of the route table
// should be independent of the underlying DBMS. Route should define an
// abstract interface for some other module to be able to bulk-load and
// incrementally update routes. Since Router should not care where its routes
// come from, Route and Backend should not contain bson fields.
// MongoReplicaSet, MongoReplicaSetMember etc. should move out of this module.
type Router struct {
	mux        *triemux.Mux
	lock       sync.RWMutex
	logger     logger.Logger
	opts       Options
	ReloadChan chan bool
}

type Options struct {
	DatabaseURL          string
	DatabaseName         string
	Listener             *pq.Listener
	DatabasePollInterval time.Duration
	BackendConnTimeout   time.Duration
	BackendHeaderTimeout time.Duration
	LogFileName          string
}

type Backend struct {
	BackendID     sql.NullString
	BackendURL    sql.NullString
	SubdomainName sql.NullString
}

type Route struct {
	IncomingPath sql.NullString
	RouteType    sql.NullString
	Handler      sql.NullString
	BackendID    sql.NullString
	RedirectTo   sql.NullString
	RedirectType sql.NullString
	SegmentsMode sql.NullString
	Disabled     bool
}

// RegisterMetrics registers Prometheus metrics from the router module and the
// modules that it directly depends on. To use the default (global) registry,
// pass prometheus.DefaultRegisterer.
func RegisterMetrics(r prometheus.Registerer) {
	registerMetrics(r)
}

// NewRouter returns a new empty router instance. You will need to call
// SelfUpdateRoutes() to initialise the self-update process for routes.
func NewRouter(o Options) (rt *Router, err error) {
	logInfo("router: using database poll interval:", o.DatabasePollInterval)
	logInfo("router: using backend connect timeout:", o.BackendConnTimeout)
	logInfo("router: using backend header timeout:", o.BackendHeaderTimeout)

	l, err := logger.New(o.LogFileName)
	if err != nil {
		return nil, err
	}
	logInfo("router: logging errors as JSON to", o.LogFileName)

	listenerProblemReporter := func(event pq.ListenerEventType, err error) {
		if err != nil {
			logWarn(fmt.Sprintf("pq: error creating listener for PSQL notify channel: %v)", err))
			return
		}
	}

	listener := pq.NewListener(o.DatabaseURL, 10*time.Second, time.Minute, listenerProblemReporter)
	o.Listener = listener

	err = listener.Listen("notify")
	if err != nil {
		panic(err)
	}

	reloadChan := make(chan bool, 1)
	rt = &Router{
		mux:        triemux.NewMux(),
		logger:     l,
		opts:       o,
		ReloadChan: reloadChan,
	}

	go rt.pollAndReload()

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
	rt.lock.RLock()
	mux := rt.mux
	rt.lock.RUnlock()

	mux.ServeHTTP(w, req)
}

func (rt *Router) SelfUpdateRoutes() {
	logInfo(fmt.Sprintf("router: starting self-update process, polling for route changes every %v", rt.opts.DatabasePollInterval))

	tick := time.Tick(rt.opts.DatabasePollInterval)
	for range tick {
		logDebug("router: polling MongoDB for changes")

		rt.ReloadChan <- true
	}
}

// pollAndReload blocks until it receives a message on reloadChan,
// and will immediately reload again if another message was received
// during reload.
func (rt *Router) pollAndReload() {
	for range rt.ReloadChan {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logWarn(r)
				}
			}()

			logDebug("pq: connecting to", rt.opts.DatabaseURL)

			sess, err := sql.Open("postgres", rt.opts.DatabaseURL)
			if err != nil {
				logWarn(fmt.Sprintf("pq: error connecting to PSQL database, skipping update (error: %v)", err))
				return
			}

			defer sess.Close()

			if rt.shouldReload(rt.opts.Listener) {
				logDebug("router: updates found")
				rt.reloadRoutes(sess)
			} else {
				logDebug("router: no updates found - really?")
			}
		}()
	}
}

// reloadRoutes reloads the routes for this Router instance on the fly. It will
// create a new proxy mux, load applications (backends) and routes into it, and
// then flip the "mux" pointer in the Router.
func (rt *Router) reloadRoutes(db *sql.DB) {
	var success bool
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		labels := prometheus.Labels{"success": strconv.FormatBool(success)}
		routeReloadDurationMetric.With(labels).Observe(v)
	}))

	defer func() {
		success = true
		if r := recover(); r != nil {
			success = false
			logWarn("router: recovered from panic in reloadRoutes:", r)
			logInfo("router: original routes have not been modified")
			errorMessage := fmt.Sprintf("panic: %v", r)
			err := logger.RecoveredError{ErrorMessage: errorMessage}
			logger.NotifySentry(logger.ReportableError{Error: err})
		}
		timer.ObserveDuration()
	}()

	logInfo("router: reloading routes")
	newmux := triemux.NewMux()

	backends := rt.loadBackends(db)
	loadRoutes(db, newmux, backends)
	routeCount := newmux.RouteCount()

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes", routeCount))
	routesCountMetric.Set(float64(routeCount))
}

func (rt *Router) shouldReload(listener *pq.Listener) bool {
	select {
	case n := <-listener.Notify:
		// n.Extra contains the payload from the notification
		logInfo("notification:", n.Channel)
		return true
	default:
		if err := listener.Ping(); err != nil {
			panic(err)
		}
		return false
	}
}

// loadBackends is a helper function which loads backends from the
// passed mongo collection, constructs a Handler for each one, and returns
// them in map keyed on the backend_id
func (rt *Router) loadBackends(db *sql.DB) (backends map[string]http.Handler) {
	backend := &Backend{}
	backends = make(map[string]http.Handler)

	rows, err := db.Query("SELECT backend_id, backend_url FROM backends")
	if err != nil {
		logWarn(fmt.Sprintf("pq: error retrieving row information from table, skipping update. (error: %v)", err))
		return
	}

	for rows.Next() {
		err := rows.Scan(&backend.BackendID, &backend.BackendURL)
		if err != nil {
			logWarn(fmt.Sprintf("pq: error retrieving row information from table, skipping update. (error: %v)", err))
			return
		}

		backendURL, err := backend.ParseURL()
		if err != nil {
			logWarn(fmt.Sprintf("router: couldn't parse URL %s for backends %s "+
				"(error: %v), skipping!", backend.BackendURL.String, backend.BackendID.String, err))
			continue
		}

		backends[backend.BackendID.String] = handlers.NewBackendHandler(
			backend.BackendID.String,
			backendURL,
			rt.opts.BackendConnTimeout,
			rt.opts.BackendHeaderTimeout,
			rt.logger,
		)
	}

	return
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(db *sql.DB, mux *triemux.Mux, backends map[string]http.Handler) {
	route := &Route{}

	rows, err := db.Query("SELECT incoming_path, route_type, handler, disabled, backend_id, redirect_to, redirect_type, segments_mode FROM routes")
	if err != nil {
		logWarn(fmt.Sprintf("pq: error retrieving row information from table, skipping update. (error: %v)", err))
		return
	}

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "410 Gone", http.StatusGone)
	})
	unavailableHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "503 Service Unavailable", http.StatusServiceUnavailable)
	})

	for rows.Next() {
		err := rows.Scan(&route.IncomingPath, &route.RouteType, &route.Handler, &route.Disabled, &route.BackendID, &route.RedirectTo, &route.RedirectType, &route.SegmentsMode)
		if err != nil {
			logWarn(fmt.Sprintf("pq: error retrieving row information from table, skipping update. (error: %v)", err))
			return
		}
		prefix := (route.RouteType.String == RouteTypePrefix)

		// the database contains paths with % encoded routes.
		// Unescape them here because the http.Request objects we match against contain the unescaped variants.
		incomingURL, err := url.Parse(route.IncomingPath.String)
		if err != nil {
			logWarn(fmt.Sprintf("router: found route %+v with invalid incoming path '%s', skipping!", route, route.IncomingPath.String))
			continue
		}

		if route.Disabled {
			mux.Handle(incomingURL.Path, prefix, unavailableHandler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v)(disabled) -> Unavailable", incomingURL.Path, prefix))
			continue
		}

		switch route.Handler.String {
		case "backend":
			handler, ok := backends[route.BackendID.String]
			if !ok {
				logWarn(fmt.Sprintf("router: found route %+v which references unknown backend "+
					"%s, skipping!", route, route.BackendID.String))
				continue
			}
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) for %s",
				incomingURL.Path, prefix, route.BackendID.String))
		case "redirect":
			redirectTemporarily := (route.RedirectType.String == "temporary")
			handler := handlers.NewRedirectHandler(incomingURL.Path, route.RedirectTo.String, shouldPreserveSegments(route), redirectTemporarily)
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) -> %s",
				incomingURL.Path, prefix, route.RedirectTo.String))
		case "gone":
			mux.Handle(incomingURL.Path, prefix, goneHandler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) -> Gone", incomingURL.Path, prefix))
		case "boom":
			// Special handler so that we can test failure behaviour.
			mux.Handle(incomingURL.Path, prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("Boom!!!")
			}))
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) -> Boom!!!", incomingURL.Path, prefix))
		default:
			logWarn(fmt.Sprintf("router: found route %+v with unknown handler type "+
				"%s, skipping!", route, route.Handler.String))
			continue
		}
	}
}

func (be *Backend) ParseURL() (*url.URL, error) {
	backendURL := os.Getenv(fmt.Sprintf("BACKEND_URL_%s", be.BackendID.String))
	if backendURL == "" {
		backendURL = be.BackendURL.String
	}
	return url.Parse(backendURL)
}

func shouldPreserveSegments(route *Route) bool {
	switch route.RouteType.String {
	case RouteTypeExact:
		return route.SegmentsMode.String == SegmentsModePreserve
	case RouteTypePrefix:
		return route.SegmentsMode.String != SegmentsModeIgnore
	default:
		return false
	}
}
