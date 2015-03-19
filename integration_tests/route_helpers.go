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
	route := bson.M{
		"handler":    "backend",
		"backend_id": backendID,
	}
	addRoute(path, route, possibleRouteType...)
}

func addRedirectRoute(path, destination string, extraParams ...string) {
	route := bson.M{
		"handler":     "redirect",
		"redirect_to": destination,
	}
	// extraParams[0] handled by addRoute
	if len(extraParams) > 1 {
		route["redirect_type"] = extraParams[1]
	}
	addRoute(path, route, extraParams...)
}

func addGoneRoute(path string, possibleRouteType ...string) {
	addRoute(path, bson.M{"handler": "gone"}, possibleRouteType...)
}

func addRoute(path string, details bson.M, possibleRouteType ...string) {
	details["incoming_path"] = path
	details["route_type"] = "exact"
	if len(possibleRouteType) > 0 {
		details["route_type"] = possibleRouteType[0]
	}
	err := routerDB.C("routes").Insert(details)
	Expect(err).To(BeNil())
}

func clearRoutes() {
	routerDB.C("routes").DropCollection()
	routerDB.C("backends").DropCollection()
}
