package main

import (
	"context"
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

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a passed mongo database.
type Router struct {
	mux                   *triemux.Mux
	lock                  sync.RWMutex
	mongoURL              string
	mongoDbName           string
	backendConnectTimeout time.Duration
	backendHeaderTimeout  time.Duration
	mongoContext					context.Context
	logger                logger.Logger
	ReloadChan            chan bool
	changeStreams         [2]ChangeStreamInfo
}

type ChangeStreamInfo struct {
	collectionName  string
	stream 					*mongo.ChangeStream
	isValid					bool
	invalidChan			chan bool
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
func NewRouter(mongoURL, mongoDbName, backendConnectTimeout, backendHeaderTimeout, logFileName string) (rt *Router, err error) {
	beConnTimeout, err := time.ParseDuration(backendConnectTimeout)
	if err != nil {
		return nil, err
	}
	beHeaderTimeout, err := time.ParseDuration(backendHeaderTimeout)
	if err != nil {
		return nil, err
	}
	logInfo("router: using backend connect timeout:", beConnTimeout)
	logInfo("router: using backend header timeout:", beHeaderTimeout)

	l, err := logger.New(logFileName)
	if err != nil {
		return nil, err
	}

	logInfo("router: logging errors as JSON to", logFileName)

	reloadChan := make(chan bool, 1)
	invalidCSChan := make(chan bool, 1)
	rt = &Router{
		mux:                   triemux.NewMux(),
		mongoURL:              mongoURL,
		mongoDbName:           mongoDbName,
		backendConnectTimeout: beConnTimeout,
		backendHeaderTimeout:  beHeaderTimeout,
		mongoContext:					 context.Background(),
		logger:                l,
		ReloadChan:            reloadChan,
		changeStreams:				 [2]ChangeStreamInfo{
			ChangeStreamInfo {
				collectionName: "routes",
				isValid: false,
				invalidChan: invalidCSChan,
			},
			ChangeStreamInfo {
				collectionName: "backendas",
				isValid: false,
				invalidChan: invalidCSChan,
			},
		},
	}

	go rt.pollAndReload()
	go rt.manageChangeStreams()

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

func (rt *Router) manageChangeStreams() {
	logInfo("mgo: setting up management of change streams for ", rt.mongoURL)

	uri := "mongodb://" + rt.mongoURL
	client, err := mongo.Connect(rt.mongoContext, options.Client().ApplyURI(uri))
	if err != nil {
		logWarn(fmt.Sprintf("mongo: error connecting to MongoDB, skipping change stream checking (error: %v)", err))
		return
	}

	// defer client.Disconnect(rt.mongoContext)

	// Streams, so simple but so horrific - we need to poll their setup at first because we
	// can't assume the database is ready for them.

	for (rt.changeStreams[0].isValid == false || rt.changeStreams[1].isValid == false) {
		for i := range rt.changeStreams {
			if !rt.changeStreams[i].isValid {
				rt.tryStreamWatch(client, rt.mongoDbName, &rt.changeStreams[i])
				if rt.changeStreams[i].isValid {
					go iterateStreamCursor(rt.mongoContext, rt.changeStreams[i].stream, rt.ReloadChan, rt.changeStreams[i].invalidChan)
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func iterateStreamCursor(context context.Context, cs *mongo.ChangeStream, reloadChan chan bool, invalidateChan chan bool) {
	logInfo("Listening on stream!")
	for cs.Next(context) {
		logInfo("Detected updates in collection, signalling reload channel")
		reloadChan <- true;
	}

	invalidateChan <- true;
}

func (rt *Router) tryStreamWatch(client *mongo.Client, dbName string, changeStream *ChangeStreamInfo) {
	db := client.Database(dbName)
	collection := db.Collection(changeStream.collectionName)

	logInfo(fmt.Sprintf("Connecting to change stream on collection: %s", changeStream.collectionName))
	cs, err := collection.Watch(rt.mongoContext, mongo.Pipeline{})
	if err != nil {
		changeStream.isValid = false
		logWarn("Unable to listen to change stream")
		logWarn(err)
		return
	}

	changeStream.isValid = true
	changeStream.stream = cs
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

			uri := "mongodb://" + rt.mongoURL
			client, err := mongo.Connect(rt.mongoContext, options.Client().ApplyURI(uri))
			if err != nil {
				logWarn(fmt.Sprintf("mongo: error connecting to MongoDB, skipping update (error: %v)", err))
				return
			}

			defer client.Disconnect(rt.mongoContext)

			rt.reloadRoutes(client.Database(rt.mongoDbName))
		}()
	}
}

// reloadRoutes reloads the routes for this Router instance on the fly. It will
// create a new proxy mux, load applications (backends) and routes into it, and
// then flip the "mux" pointer in the Router.
func (rt *Router) reloadRoutes(db *mongo.Database) {
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

	backends := rt.loadBackends(db.Collection("backends"))
	rt.loadRoutes(db.Collection("routes"), newmux, backends)

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes", rt.mux.RouteCount()))

	routesCountMetric.Set(float64(rt.mux.RouteCount()))
}

// loadBackends is a helper function which loads backends from the
// passed mongo collection, constructs a Handler for each one, and returns
// them in map keyed on the backend_id
func (rt *Router) loadBackends(c *mongo.Collection) (backends map[string]http.Handler) {
	backend := &Backend{}
	backends = make(map[string]http.Handler)

	iter, _ := c.Find(rt.mongoContext, bson.D{})

	for iter.Next(rt.mongoContext) {
		iter.Decode(&backend);
		backendURL, err := backend.ParseURL()
		if err != nil {
			logWarn(fmt.Sprintf("router: couldn't parse URL %s for backend %s "+
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

	if err := iter.Err(); err != nil {
		panic(err)
	}

	return
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func (rt *Router) loadRoutes(c *mongo.Collection, mux *triemux.Mux, backends map[string]http.Handler) {
	route := &Route{}

	iter, _ := c.Find(rt.mongoContext, bson.D{})

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "410 Gone", http.StatusGone)
	})
	unavailableHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "503 Service Unavailable", http.StatusServiceUnavailable)
	})

	for iter.Next(rt.mongoContext) {
		iter.Decode(&route);
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
