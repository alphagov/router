package main

import (
	"fmt"
	"github.com/alphagov/router/triemux"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Router is a wrapper around an HTTP multiplexer (trie.Mux) which retrieves its
// routes from a passed mongo database.
type Router struct {
	mux         *triemux.Mux
	mongoUrl    string
	mongoDbName string
}

type Application struct {
	ApplicationId string `bson:"application_id"`
	BackendURL    string `bson:"backend_url"`
}

type Route struct {
	IncomingPath  string `bson:"incoming_path"`
	ApplicationId string `bson:"application_id"`
	RouteType     string `bson:"route_type"`
}

// NewRouter returns a new empty router instance. You will still need to call
// ReloadRoutes() to do the initial route load.
func NewRouter(mongoUrl, mongoDbName string) *Router {
	return &Router{
		mux:         triemux.NewMux(),
		mongoUrl:    mongoUrl,
		mongoDbName: mongoDbName,
	}
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

	apps := loadApplications(db.C("applications"))
	loadRoutes(db.C("routes"), newmux, apps)

	rt.mux = newmux
	log.Printf("router: reloaded routes")
}

// loadApplications is a helper function which loads applications from the
// passed mongo collection, constructs a Handler for each one, and returns
// them in map keyed on the backend-id
func loadApplications(c *mgo.Collection) (apps map[string]http.Handler) {
	app := &Application{}
	apps = make(map[string]http.Handler)

	iter := c.Find(nil).Iter()

	for iter.Next(&app) {
		backendUrl, err := url.Parse(app.BackendURL)
		if err != nil {
			log.Printf("router: couldn't parse URL %s for backend %s "+
				"(error: %v), skipping!", app.BackendURL, app.ApplicationId, err)
			continue
		}

		apps[app.ApplicationId] = newBackendReverseProxy(backendUrl)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}

	return
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(c *mgo.Collection, mux *triemux.Mux, apps map[string]http.Handler) {
	route := &Route{}

	iter := c.Find(nil).Iter()

	for iter.Next(&route) {
		handler, ok := apps[route.ApplicationId]
		if !ok {
			log.Printf("router: found route %+v which references unknown application "+
				"%s, skipping!", route, route.ApplicationId)
			continue
		}

		prefix := (route.RouteType == "prefix")
		mux.Handle(route.IncomingPath, prefix, handler)
		log.Printf("router: registered %s (prefix: %v) for %s",
			route.IncomingPath, prefix, route.ApplicationId)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}
}

func newBackendReverseProxy(backendUrl *url.URL) (proxy *httputil.ReverseProxy) {
	proxy = httputil.NewSingleHostReverseProxy(backendUrl)
	// Allow the proxy to keep more than the default (2) keepalive connections
	// per upstream.
	proxy.Transport = &http.Transport{MaxIdleConnsPerHost: 20}

	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)

		// Set the Host header to match the backend hostname instead of the one from the incoming request.
		req.Host = backendUrl.Host

		// Setting a blank User-Agent causes the http lib not to output one, whereas if there
		// is no header, it will output a default one.
		// See: http://code.google.com/p/go/source/browse/src/pkg/net/http/request.go?name=go1.1.2#349
		if _, present := req.Header["User-Agent"]; !present {
			req.Header.Set("User-Agent", "")
		}

		populateViaHeader(req.Header, fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor))
	}

	return proxy
}

func populateViaHeader(header http.Header, httpVersion string) {
	via := httpVersion + " router"
	if prior, ok := header["Via"]; ok {
		via = strings.Join(prior, ", ") + ", " + via
	}
	header.Set("Via", via)
}
