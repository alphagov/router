package router

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/logger"
	"github.com/alphagov/router/triemux"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
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
	backends                map[string]http.Handler
	mux                     *triemux.Mux
	csMux                   *triemux.Mux
	lock                    sync.RWMutex
	mongoReadToOptime       bson.MongoTimestamp
	logger                  logger.Logger
	opts                    Options
	ReloadChan              chan bool
	CsReloadChan            chan bool
	csMuxSampleRate         float64
	csLastAttemptReloadTime time.Time
	pool                    *pgxpool.Pool
}

type Options struct {
	MongoURL             string
	MongoDBName          string
	MongoPollInterval    time.Duration
	BackendConnTimeout   time.Duration
	BackendHeaderTimeout time.Duration
	LogFileName          string
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
	SegmentsMode string `bson:"segments_mode"`
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
	logInfo("router: using mongo poll interval:", o.MongoPollInterval)
	logInfo("router: using backend connect timeout:", o.BackendConnTimeout)
	logInfo("router: using backend header timeout:", o.BackendHeaderTimeout)

	l, err := logger.New(o.LogFileName)
	if err != nil {
		return nil, err
	}
	logInfo("router: logging errors as JSON to", o.LogFileName)

	mongoReadToOptime, err := bson.NewMongoTimestamp(time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC), 1)
	if err != nil {
		return nil, err
	}

	backends := loadBackendsFromEnv(o.BackendConnTimeout, o.BackendHeaderTimeout, l)

	csMuxSampleRate, err := strconv.ParseFloat(os.Getenv("CSMUX_SAMPLE_RATE"), 64)

	if err != nil {
		csMuxSampleRate = 0.0
	}

	logInfo("router: content store mux sample rate set at", csMuxSampleRate)

	var pool *pgxpool.Pool

	if csMuxSampleRate != 0.0 {
		pool, err = pgxpool.New(context.Background(), os.Getenv("CONTENT_STORE_DATABASE_URL"))
		if err != nil {
			return nil, err
		}
		logInfo("router: postgres connection pool created")
	} else {
		logInfo("router: not using content store postgres")
	}

	reloadChan := make(chan bool, 1)
	csReloadChan := make(chan bool, 1)
	rt = &Router{
		backends:          backends,
		mux:               triemux.NewMux(),
		csMux:             triemux.NewMux(),
		mongoReadToOptime: mongoReadToOptime,
		logger:            l,
		opts:              o,
		ReloadChan:        reloadChan,
		CsReloadChan:      csReloadChan,
		pool:              pool,
		csMuxSampleRate:   csMuxSampleRate,
	}

	if csMuxSampleRate != 0.0 {
		rt.reloadCsRoutes(pool)

		go func() {
			if err := rt.listenForContentStoreUpdates(context.Background()); err != nil {
				logWarn(fmt.Sprintf("router: error in listenForContentStoreUpdates: %v", err))
			}
		}()

		go rt.waitForReload()
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

	useContentStoreMux := rt.csMuxSampleRate > 0 && rand.Float64() < rt.csMuxSampleRate //nolint:gosec
	var mux *triemux.Mux

	rt.lock.RLock()
	if useContentStoreMux {
		mux = rt.csMux
	} else {
		mux = rt.mux
	}
	rt.lock.RUnlock()

	mux.ServeHTTP(w, req)
}

func (rt *Router) SelfUpdateRoutes() {
	logInfo(fmt.Sprintf("router: starting self-update process, polling for route changes every %v", rt.opts.MongoPollInterval))

	tick := time.Tick(rt.opts.MongoPollInterval)
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

			logDebug("mgo: connecting to", rt.opts.MongoURL)

			sess, err := mgo.Dial(rt.opts.MongoURL)
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
				rt.reloadRoutes(sess.DB(rt.opts.MongoDBName), currentMongoInstance.Optime)
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
	var success bool
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		labels := prometheus.Labels{"success": strconv.FormatBool(success), "source": "mongo"}
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
		} else {
			rt.mongoReadToOptime = currentOptime
		}
		timer.ObserveDuration()
	}()

	logInfo("router: reloading routes")
	newmux := triemux.NewMux()

	loadRoutes(db.C("routes"), newmux, rt.backends)
	routeCount := newmux.RouteCount()

	rt.lock.Lock()
	rt.mux = newmux
	rt.lock.Unlock()

	logInfo(fmt.Sprintf("router: reloaded %d routes", routeCount))
	routesCountMetric.WithLabelValues("mongo").Set(float64(routeCount))
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

	// #nosec G602 -- not actually an out-of-bounds access.
	return currentInstance[0], nil
}

func (rt *Router) shouldReload(currentMongoInstance MongoReplicaSetMember) bool {
	return currentMongoInstance.Optime > rt.mongoReadToOptime
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(c *mgo.Collection, mux *triemux.Mux, backends map[string]http.Handler) {
	route := &Route{}

	iter := c.Find(nil).Iter()

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "410 Gone", http.StatusGone)
	})

	for iter.Next(&route) {
		prefix := (route.RouteType == RouteTypePrefix)

		// the database contains paths with % encoded routes.
		// Unescape them here because the http.Request objects we match against contain the unescaped variants.
		incomingURL, err := url.Parse(route.IncomingPath)
		if err != nil {
			logWarn(fmt.Sprintf("router: found route %+v with invalid incoming path '%s', skipping!", route, route.IncomingPath))
			continue
		}

		switch route.Handler {
		case HandlerTypeBackend:
			handler, ok := backends[route.BackendID]
			if !ok {
				logWarn(fmt.Sprintf("router: found route %+v which references unknown backend "+
					"%s, skipping!", route, route.BackendID))
				continue
			}
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) for %s",
				incomingURL.Path, prefix, route.BackendID))
		case HandlerTypeRedirect:
			handler := handlers.NewRedirectHandler(incomingURL.Path, route.RedirectTo, shouldPreserveSegments(route.RouteType, route.SegmentsMode))
			mux.Handle(incomingURL.Path, prefix, handler)
			logDebug(fmt.Sprintf("router: registered %s (prefix: %v) -> %s",
				incomingURL.Path, prefix, route.RedirectTo))
		case HandlerTypeGone:
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
