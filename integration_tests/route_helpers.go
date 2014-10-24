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
	routeType := "exact"
	if len(possibleRouteType) > 0 {
		routeType = possibleRouteType[0]
	}
	routerDB.C("routes").Insert(bson.M{
		"incoming_path": path,
		"route_type":    routeType,
		"handler":       "backend",
		"backend_id":    backendId,
	})
}

func clearRoutes() {
	routerDB.C("routes").DropCollection()
	routerDB.C("backends").DropCollection()
}
