package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router-postgres/handlers"
	"github.com/alphagov/router-postgres/logger"
	"github.com/alphagov/router-postgres/triemux"
	"github.com/lib/pq"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a passed mongo database.
type Router struct {
	mux                   *triemux.Mux
	lock                  sync.RWMutex
	postgresURL           string
	postgresDbName        string
	listener              *pq.Listener
	dbPollInterval        time.Duration
	backendConnectTimeout time.Duration
	backendHeaderTimeout  time.Duration
	logger                logger.Logger
	ReloadChan            chan bool
}

type Backend struct {
	BackendID     string `bson:"backend_id"`
	BackendURL    string `bson:"backend_url"`
	SubdomainName string `bson:"subdomain_name"`
}

type Route struct {
	IncomingPath string `bson:"incoming_path"`
	RouteType    string `bson:"route_type"`
	Handler      string `bson:"handler"`
	BackendID    string `bson:"backend_id"`
	RedirectTo   string `bson:"redirect_to"`
	RedirectType string `bson:"redirect_type"`
	SegmentsMode string `bson:"segments_mode"`
	Disabled     bool   `bson:"disabled"`
}

// NewRouter returns a new empty router instance. You will need to call
// SelfUpdateRoutes() to initialise the self-update process for routes.
func NewRouter(postgresURL, postgresDbName, dbPollInterval, backendConnectTimeout, backendHeaderTimeout, logFileName string) (rt *Router, err error) {
	pgPollInterval, err := time.ParseDuration(dbPollInterval)
	if err != nil {
		return nil, err
	}

	listenerProblemReporter := func(event pq.ListenerEventType, err error) {
		if err != nil {
			logWarn(fmt.Sprintf("pq: error creating listener for PSQL notify channel: %v)", err))
			return
		}
	}

	listener := pq.NewListener(postgresURL, 10*time.Second, time.Minute, listenerProblemReporter)

	err = listener.Listen("events")
	if err != nil {
		panic(err)
	}

	beConnTimeout, err := time.ParseDuration(backendConnectTimeout)
	if err != nil {
		return nil, err
	}
	beHeaderTimeout, err := time.ParseDuration(backendHeaderTimeout)
	if err != nil {
		return nil, err
	}
	logInfo("router: using postgres poll interval:", pgPollInterval)
	logInfo("router: using backend connect timeout:", beConnTimeout)
	logInfo("router: using backend header timeout:", beHeaderTimeout)

	l, err := logger.New(logFileName)
	if err != nil {
		return nil, err
	}

	logInfo("router: logging errors as JSON to", logFileName)

	reloadChan := make(chan bool, 1)
	rt = &Router{
		mux:                   triemux.NewMux(),
		postgresURL:           postgresURL,
		postgresDbName:        postgresDbName,
		listener:              listener,
		dbPollInterval:        pgPollInterval,
		backendConnectTimeout: beConnTimeout,
		backendHeaderTimeout:  beHeaderTimeout,
		logger:                l,
		ReloadChan:            reloadChan,
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
	logInfo(fmt.Sprintf("router: starting self-update process, polling for route changes every %v", rt.dbPollInterval))

	tick := time.Tick(rt.dbPollInterval)
	for range tick {
		logDebug("router: polling postgres for changes")

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

			logDebug("pq: connecting to", rt.postgresURL)

			sess, err := sql.Open("postgres", rt.postgresURL)
			if err != nil {
				logWarn(fmt.Sprintf("pq: error connecting to PSQL database, skipping update (error: %v)", err))
				return
			}

			defer sess.Close()

			if rt.shouldReload(rt.listener) {
				logDebug("router: updates found")
				rt.reloadRoutes(sess)
			} else {
				logDebug("router: no updates found")
			}
		}()
	}
}

type mongoDatabase interface {
	Run(command interface{}, result interface{}) error
}

// reloadRoutes reloads the routes for this Router instance on the fly. It will
// create a new proxy mux, load applications (backends) and routes into it, and
// then flip the "mux" pointer in the Router.
func (rt *Router) reloadRoutes(db *sql.DB) {
	defer func() {
		// increment this metric regardless of whether the route reload succeeded
		routeReloadCountMetric.Inc()

		if r := recover(); r != nil {
			logWarn("router: recovered from panic in reloadRoutes:", r)
			logInfo("router: original routes have not been modified")
			errorMessage := fmt.Sprintf("panic: %v", r)
			err := logger.RecoveredError{ErrorMessage: errorMessage}
			logger.NotifySentry(logger.ReportableError{Error: err})

			routeReloadErrorCountMetric.Inc()
		}
	}()

	logInfo("router: reloading routes")
	newmux := triemux.NewMux()

	backends := rt.loadBackends(db)
	loadRoutes(db, newmux, backends)

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes", rt.mux.RouteCount()))

	routesCountMetric.Set(float64(rt.mux.RouteCount()))
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

	rows, err := db.Query("SELECT * FROM backends")
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
				"(error: %v), skipping!", backend.BackendURL, backend.BackendID, err))
			continue
		}

		backends[backend.BackendID] = handlers.NewBackendHandler(
			backend.BackendID,
			backendURL,
			rt.backendConnectTimeout, rt.backendHeaderTimeout,
			rt.logger,
		)
	}

	return
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(db *sql.DB, mux *triemux.Mux, backends map[string]http.Handler) {
	route := &Route{}

	rows, err := db.Query("SELECT * FROM routes")
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

		prefix := (route.RouteType == "prefix")

		// the database contains paths with % encoded routes.
		// Unescape them here because the http.Request objects we match against contain the unescaped variants.
		incomingURL, err := url.Parse(route.IncomingPath)
		if err != nil {
			logWarn(fmt.Sprintf("router: found route %+v with invalid incoming path '%s', skipping!", route, route.IncomingPath))
			continue
		}

		if route.Disabled {
			mux.Handle(incomingURL.Path, prefix, unavailableHandler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v)(disabled) -> Unavailable", incomingURL.Path, prefix))
			continue
		}

		switch route.Handler {
		case "backend":
			handler, ok := backends[route.BackendID]
			if !ok {
				logWarn(fmt.Sprintf("router: found route %+v which references unknown backend "+
					"%s, skipping!", route, route.BackendID))
				continue
			}
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) for %s",
				incomingURL.Path, prefix, route.BackendID))
		case "redirect":
			redirectTemporarily := (route.RedirectType == "temporary")
			handler := handlers.NewRedirectHandler(incomingURL.Path, route.RedirectTo, shouldPreserveSegments(route), redirectTemporarily)
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) -> %s",
				incomingURL.Path, prefix, route.RedirectTo))
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
				"%s, skipping!", route, route.Handler))
			continue
		}
	}
}

func (be *Backend) ParseURL() (*url.URL, error) {
	backend_url := os.Getenv(fmt.Sprintf("BACKEND_URL_%s", be.BackendID))
	if backend_url == "" {
		return url.Parse(be.BackendURL)
	}
	return url.Parse(backend_url)
}

func (rt *Router) RouteStats() (stats map[string]interface{}) {
	rt.lock.RLock()
	mux := rt.mux
	rt.lock.RUnlock()

	stats = make(map[string]interface{})
	stats["count"] = mux.RouteCount()
	return
}

func shouldPreserveSegments(route *Route) bool {
	switch {
	case route.RouteType == "exact" && route.SegmentsMode == "preserve":
		return true
	case route.RouteType == "exact":
		return false
	case route.RouteType == "prefix" && route.SegmentsMode == "ignore":
		return false
	case route.RouteType == "prefix":
		return true
	}
	return false
}
