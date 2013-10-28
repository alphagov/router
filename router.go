package main

import (
	"fmt"
	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/triemux"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a passed mongo database.
type Router struct {
	mux                   *triemux.Mux
	mongoUrl              string
	mongoDbName           string
	backendConnectTimeout time.Duration
	backendHeaderTimeout  time.Duration
}

type Backend struct {
	BackendId  string `bson:"backend_id"`
	BackendURL string `bson:"backend_url"`
}

type Route struct {
	IncomingPath string `bson:"incoming_path"`
	RouteType    string `bson:"route_type"`
	Handler      string `bson:"handler"`
	BackendId    string `bson:"backend_id"`
	RedirectTo   string `bson:"redirect_to"`
	RedirectType string `bson:"redirect_type"`
}

// NewRouter returns a new empty router instance. You will still need to call
// ReloadRoutes() to do the initial route load.
func NewRouter(mongoUrl, mongoDbName, backendConnectTimeout, backendHeaderTimeout string) (rt *Router, err error) {
	beConnTimeout, err := time.ParseDuration(backendConnectTimeout)
	if err != nil {
		return nil, err
	}
	beHeaderTimeout, err := time.ParseDuration(backendHeaderTimeout)
	if err != nil {
		return nil, err
	}
	log.Printf("router: using backend connect timeout: %v", beConnTimeout)
	log.Printf("router: using backend header timeout: %v", beHeaderTimeout)

	rt = &Router{
		mux:                   triemux.NewMux(),
		mongoUrl:              mongoUrl,
		mongoDbName:           mongoDbName,
		backendConnectTimeout: beConnTimeout,
		backendHeaderTimeout:  beHeaderTimeout,
	}
	return rt, nil
}

// ServeHTTP delegates responsibility for serving requests to the proxy mux
// instance for this router.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

// ReloadRoutes reloads the routes for this Router instance on the fly. It will
// create a new proxy mux, load applications (backends) and routes into it, and
// then flip the "mux" pointer in the Router.
func (rt *Router) ReloadRoutes() {
	// save a reference to the previous mux in case we have to restore it
	oldmux := rt.mux
	defer func() {
		if r := recover(); r != nil {
			log.Println("router: recovered from panic in ReloadRoutes:", r)
			rt.mux = oldmux
			log.Println("router: original routes have been restored")
		}
	}()

	log.Println("mgo: connecting to", rt.mongoUrl)
	sess, err := mgo.Dial(rt.mongoUrl)
	if err != nil {
		panic(fmt.Sprintln("mgo:", err))
	}
	defer sess.Close()
	sess.SetMode(mgo.Monotonic, true)

	db := sess.DB(rt.mongoDbName)

	log.Printf("router: reloading routes")
	newmux := triemux.NewMux()

	backends := rt.loadBackends(db.C("backends"))
	loadRoutes(db.C("routes"), newmux, backends)

	rt.mux = newmux
	log.Printf("router: reloaded routes")
}

// loadBackends is a helper function which loads backends from the
// passed mongo collection, constructs a Handler for each one, and returns
// them in map keyed on the backend_id
func (rt *Router) loadBackends(c *mgo.Collection) (backends map[string]http.Handler) {
	backend := &Backend{}
	backends = make(map[string]http.Handler)

	iter := c.Find(nil).Iter()

	for iter.Next(&backend) {
		backendUrl, err := url.Parse(backend.BackendURL)
		if err != nil {
			log.Printf("router: couldn't parse URL %s for backend %s "+
				"(error: %v), skipping!", backend.BackendURL, backend.BackendId, err)
			continue
		}

		backends[backend.BackendId] = handlers.NewBackendHandler(backendUrl, rt.backendConnectTimeout, rt.backendHeaderTimeout)
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

	for iter.Next(&route) {
		prefix := (route.RouteType == "prefix")
		switch route.Handler {
		case "backend":
			handler, ok := backends[route.BackendId]
			if !ok {
				log.Printf("router: found route %+v which references unknown application "+
					"%s, skipping!", route, route.BackendId)
				continue
			}
			mux.Handle(route.IncomingPath, prefix, handler)
			log.Printf("router: registered %s (prefix: %v) for %s",
				route.IncomingPath, prefix, route.BackendId)
		case "redirect":
			if prefix {
				log.Printf("router: found redirect route %+v which is a prefix route, skipping!", route)
				continue
			}
			redirectTemporarily := (route.RedirectType == "temporary")
			handler := handlers.NewRedirectHandler(route.RedirectTo, redirectTemporarily)
			mux.Handle(route.IncomingPath, false, handler)
			log.Printf("router: registered %s -> %s",
				route.IncomingPath, route.RedirectTo)
		default:
			log.Printf("router: found route %+v with unknown handler type "+
				"%s, skipping!", route, route.Handler)
			continue
		}

	}

	if err := iter.Err(); err != nil {
		panic(err)
	}
}
