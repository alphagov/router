package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/logger"
	"github.com/alphagov/router/triemux"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a passed mongo database.
type Router struct {
	mux                   *triemux.Mux
	lock                  sync.RWMutex
	mongoURL              string
	mongoDbName           string
	mongoPollInterval     time.Duration
	backendConnectTimeout time.Duration
	backendHeaderTimeout  time.Duration
	mongoReadToOptime     bson.MongoTimestamp
	logger                logger.Logger
	ReloadChan            chan bool
}

type Backend struct {
	BackendID     string `bson:"backend_id"`
	BackendURL    string `bson:"backend_url"`
	SubdomainName string `bson:"subdomain_name"`
}

type MongoReplicaSet struct {
	Members []MongoReplicaSetMember `bson:"members"`
}

type MongoReplicaSetMember struct {
	Name    string              `bson:"name"`
	Optime  bson.MongoTimestamp `bson:"optime"`
	Current bool                `bson:"self"`
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
func NewRouter(mongoURL, mongoDbName, mongoPollInterval, backendConnectTimeout, backendHeaderTimeout, logFileName string) (rt *Router, err error) {
	mgoPollInterval, err := time.ParseDuration(mongoPollInterval)
	if err != nil {
		return nil, err
	}
	beConnTimeout, err := time.ParseDuration(backendConnectTimeout)
	if err != nil {
		return nil, err
	}
	beHeaderTimeout, err := time.ParseDuration(backendHeaderTimeout)
	if err != nil {
		return nil, err
	}
	logInfo("router: using mongo poll interval:", mgoPollInterval)
	logInfo("router: using backend connect timeout:", beConnTimeout)
	logInfo("router: using backend header timeout:", beHeaderTimeout)

	l, err := logger.New(logFileName)
	if err != nil {
		return nil, err
	}

	mongoReadToOptime, err := bson.NewMongoTimestamp(time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC), 1)
	if err != nil {
		return nil, err
	}

	logInfo("router: logging errors as JSON to", logFileName)

	reloadChan := make(chan bool, 1)
	rt = &Router{
		mux:                   triemux.NewMux(),
		mongoURL:              mongoURL,
		mongoPollInterval:     mgoPollInterval,
		mongoDbName:           mongoDbName,
		backendConnectTimeout: beConnTimeout,
		backendHeaderTimeout:  beHeaderTimeout,
		mongoReadToOptime:     mongoReadToOptime,
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
	logInfo(fmt.Sprintf("router: starting self-update process, polling for route changes every %v", rt.mongoPollInterval))

	tick := time.Tick(rt.mongoPollInterval)
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

			logDebug("mgo: connecting to", rt.mongoURL)

			sess, err := mgo.Dial(rt.mongoURL)
			if err != nil {
				logWarn(fmt.Sprintf("mgo: error connecting to MongoDB, skipping update (error: %v)", err))
				return
			}

			defer sess.Close()
			sess.SetMode(mgo.SecondaryPreferred, true)

			currentMongoInstance, err := rt.getCurrentMongoInstance(sess.DB("admin"))
			if err != nil {
				logWarn(err)
				return
			}

			logDebug("mgo: communicating with replica set member", currentMongoInstance.Name)

			logDebug("router: polled mongo instance is ", currentMongoInstance.Name)
			logDebug("router: polled mongo optime is ", currentMongoInstance.Optime)
			logDebug("router: current read-to mongo optime is ", rt.mongoReadToOptime)

			if rt.shouldReload(currentMongoInstance) {
				logDebug("router: updates found")
				rt.reloadRoutes(sess.DB(rt.mongoDbName), currentMongoInstance.Optime)
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
func (rt *Router) reloadRoutes(db *mgo.Database, currentOptime bson.MongoTimestamp) {
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
		} else {
			rt.mongoReadToOptime = currentOptime
		}
	}()

	logInfo("router: reloading routes")
	newmux := triemux.NewMux()

	backends := rt.loadBackends(db.C("backends"))
	loadRoutes(db.C("routes"), newmux, backends)

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes", rt.mux.RouteCount()))

	routesCountMetric.Set(float64(rt.mux.RouteCount()))
}

func (rt *Router) getCurrentMongoInstance(db mongoDatabase) (MongoReplicaSetMember, error) {
	replicaSetStatus := bson.M{}

	if err := db.Run("replSetGetStatus", &replicaSetStatus); err != nil {
		return MongoReplicaSetMember{}, fmt.Errorf("router: couldn't get replica set status from MongoDB, skipping update (error: %w)", err)
	}

	replicaSetStatusBytes, err := bson.Marshal(replicaSetStatus)
	if err != nil {
		return MongoReplicaSetMember{}, fmt.Errorf("router: couldn't marshal replica set status from MongoDB, skipping update (error: %w)", err)
	}

	replicaSet := MongoReplicaSet{}
	err = bson.Unmarshal(replicaSetStatusBytes, &replicaSet)
	if err != nil {
		return MongoReplicaSetMember{}, fmt.Errorf("router: couldn't unmarshal replica set status from MongoDB, skipping update (error: %w)", err)
	}

	currentInstance := make([]MongoReplicaSetMember, 0)
	for _, instance := range replicaSet.Members {
		if instance.Current {
			currentInstance = append(currentInstance, instance)
		}
	}

	logDebug("router: MongoDB instances", currentInstance)

	if len(currentInstance) != 1 {
		return MongoReplicaSetMember{}, fmt.Errorf("router: did not find exactly one current MongoDB instance, skipping update (current instances found: %d)", len(currentInstance))
	}

	return currentInstance[0], nil
}

func (rt *Router) shouldReload(currentMongoInstance MongoReplicaSetMember) bool {
	return currentMongoInstance.Optime > rt.mongoReadToOptime
}

// loadBackends is a helper function which loads backends from the
// passed mongo collection, constructs a Handler for each one, and returns
// them in map keyed on the backend_id
func (rt *Router) loadBackends(c *mgo.Collection) (backends map[string]http.Handler) {
	backend := &Backend{}
	backends = make(map[string]http.Handler)

	iter := c.Find(nil).Iter()

	for iter.Next(&backend) {
		backendURL, err := backend.ParseURL()
		if err != nil {
			logWarn(fmt.Errorf("router: couldn't parse URL %s for backend %s "+
				"(error: %w), skipping", backend.BackendURL, backend.BackendID, err))
			continue
		}

		backends[backend.BackendID] = handlers.NewBackendHandler(
			backend.BackendID,
			backendURL,
			rt.backendConnectTimeout, rt.backendHeaderTimeout,
			rt.logger,
		)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}

	return
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(c *mgo.Collection, mux *triemux.Mux, backends map[string]http.Handler) {
	route := &Route{}

	iter := c.Find(nil).Iter()

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "410 Gone", http.StatusGone)
	})
	unavailableHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "503 Service Unavailable", http.StatusServiceUnavailable)
	})

	for iter.Next(&route) {
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

	if err := iter.Err(); err != nil {
		panic(err)
	}
}

func (be *Backend) ParseURL() (*url.URL, error) {
	backendURL := os.Getenv(fmt.Sprintf("BACKEND_URL_%s", be.BackendID))
	if backendURL == "" {
		backendURL = be.BackendURL
	}
	return url.Parse(backendURL)
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
