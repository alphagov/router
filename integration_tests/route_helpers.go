package integration

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

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

func addBackendRoute(path, backendId string, possibleRouteType ...string) {
	route := bson.M{
		"incoming_path": path,
		"route_type":    "exact",
		"handler":       "backend",
		"backend_id":    backendId,
	}
	if len(possibleRouteType) > 0 {
		route["route_type"] = possibleRouteType[0]
	}
	routerDB.C("routes").Insert(route)
}

func addRedirectRoute(path, destination string, extraParams ...string) {
	route := bson.M{
		"incoming_path": path,
		"route_type":    "exact",
		"handler":       "redirect",
		"redirect_to":   destination,
	}
	if len(extraParams) > 0 {
		route["route_type"] = extraParams[0]
	}
	if len(extraParams) > 1 {
		route["redirect_type"] = extraParams[1]
	}
	routerDB.C("routes").Insert(route)
}

func clearRoutes() {
	routerDB.C("routes").DropCollection()
	routerDB.C("backends").DropCollection()
}
