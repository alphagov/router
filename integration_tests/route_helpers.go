package integration

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	routerDB *mgo.Database
)

type Route struct {
	IncomingPath string `bson:"incoming_path"`
	RouteType    string `bson:"route_type"`
	Handler      string `bson:"handler"`
	BackendID    string `bson:"backend_id"`
	RedirectTo   string `bson:"redirect_to"`
	RedirectType string `bson:"redirect_type"`
	Disabled     bool   `bson:"disabled"`
}

func NewBackendRoute(backendID string) Route {
	route := Route {
		Handler: "backend",
		BackendID: backendID,
	}

	return route
}

func NewRedirectRoute(redirectTo string) Route {
	route := Route {
		Handler: "redirect",
		RedirectTo: redirectTo,
		RedirectType: "permanent",
		RouteType: "exact",
	}

	return route
}

func NewGoneRoute() Route {
	route := Route {
		Handler: "gone",
	}

	return route
}

func init() {
	sess, err := mgo.Dial("localhost")
	if err != nil {
		panic("Failed to connect to mongo: " + err.Error())
	}
	routerDB = sess.DB("router_test")
}

func addBackend(id, url string) {
	err := routerDB.C("backends").Insert(bson.M{"backend_id": id, "backend_url": url})
	Expect(err).To(BeNil())
}

func addBackendRoute(path, backendID string, possibleRouteType ...string) {
	route := NewBackendRoute(backendID)

	if len(possibleRouteType) > 0 {
		route.RouteType = possibleRouteType[0]
	}

	addRoute(path, route)
}

func addRedirectRoute(path, redirectTo string, extraParams ...string) {
	route := NewRedirectRoute(redirectTo)

	if len(extraParams) > 0 {
		route.RouteType = extraParams[0]
	}
	if len(extraParams) > 1 {
		route.RedirectType = extraParams[1]
	}

	addRoute(path, route)
}

func addGoneRoute(path string, possibleRouteType ...string) {
	route := NewGoneRoute()

	if len(possibleRouteType) > 0 {
		route.RouteType = possibleRouteType[0]
	}

	addRoute(path, route)
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	err := routerDB.C("routes").Insert(route)
	Expect(err).To(BeNil())
}

func clearRoutes() {
	routerDB.C("routes").DropCollection()
	routerDB.C("backends").DropCollection()
}
