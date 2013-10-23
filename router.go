package main

import (
	"fmt"
	"github.com/alphagov/router/triemux"
	"io/ioutil"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

		backends[backend.BackendId] = newBackendReverseProxy(backendUrl, rt.backendHeaderTimeout)
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

func newBackendReverseProxy(backendUrl *url.URL, headerTimeout time.Duration) (proxy *httputil.ReverseProxy) {
	proxy = httputil.NewSingleHostReverseProxy(backendUrl)
	proxy.Transport = newBackendTransport(headerTimeout)

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

// Construct a backendTransport that wraps an http.Transport and implements http.RoundTripper.
// This allows us to intercept the response from the backend and modify it before it's copied
// back to the client.
func newBackendTransport(headerTimeout time.Duration) (transport *backendTransport) {
	transport = &backendTransport{&http.Transport{}}
	// Allow the proxy to keep more than the default (2) keepalive connections
	// per upstream.
	transport.wrapped.MaxIdleConnsPerHost = 20
	transport.wrapped.ResponseHeaderTimeout = headerTimeout
	return
}

type backendTransport struct {
	wrapped *http.Transport
}

func (bt *backendTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = bt.wrapped.RoundTrip(req)
	if err == nil {
		populateViaHeader(resp.Header, fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor))
	} else {
		switch err.Error() {
		case "net/http: timeout awaiting response headers":
			// Intercept timeout errors and generate an HTTP error response
			return newErrorResponse(504), nil
		}
	}
	return
}

func newErrorResponse(status int) (resp *http.Response) {
	resp = &http.Response{StatusCode: 504}
	resp.Body = ioutil.NopCloser(strings.NewReader(""))
	return
}
