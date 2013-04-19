package main

import (
	"fmt"
	"github.com/nickstenning/proxymux"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"net/url"
)

// Router is a wrapper around a proxy multiplexer (proxymux.Mux) which retrieves
// its routes from a passed mongo database.
type Router struct {
	mux         *proxymux.Mux
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
		mux:         proxymux.NewMux(),
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
	newmux := proxymux.NewMux()

	appMap := loadApplications(db.C("applications"), newmux)
	loadRoutes(db.C("routes"), newmux, appMap)

	rt.mux = newmux
	log.Printf("router: reloaded routes")
}

// loadApplications is a helper function which loads applications from the
// passed mongo collection and registers them as backends with the passed proxy
// mux.
func loadApplications(c *mgo.Collection, mux *proxymux.Mux) (appMap map[string]int) {
	app := &Application{}
	appMap = make(map[string]int)

	iter := c.Find(nil).Iter()

	for iter.Next(&app) {
		backendUrl, err := url.Parse(app.BackendURL)
		if err != nil {
			log.Printf("router: couldn't parse URL %s for backend %s "+
				"(error: %v), skipping!", app.BackendURL, app.ApplicationId, err)
			continue
		}

		appMap[app.ApplicationId] = mux.AddBackend(backendUrl)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}

	return appMap
}

// loadRoutes is a helper function which loads routes from the passed mongo
// collection and registers them with the passed proxy mux.
func loadRoutes(c *mgo.Collection, mux *proxymux.Mux, appMap map[string]int) {
	route := &Route{}

	iter := c.Find(nil).Iter()

	for iter.Next(&route) {
		backendId, ok := appMap[route.ApplicationId]
		if !ok {
			log.Printf("router: found route %+v which references unknown application "+
				"%s, skipping!", route, route.ApplicationId)
			continue
		}

		prefix := (route.RouteType == "prefix")
		mux.Register(route.IncomingPath, prefix, backendId)
		log.Printf("router: registered %s (prefix: %v) for %s (id: %d)",
			route.IncomingPath, prefix, route.ApplicationId, backendId)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}
}
