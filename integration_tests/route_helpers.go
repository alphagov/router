package integration

import (
	"fmt"
	"os"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

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
	SegmentsMode string `bson:"segments_mode"`
	Disabled     bool   `bson:"disabled"`
}

func NewBackendRoute(backendID string, extraParams ...string) Route {
	route := Route{
		Handler:   "backend",
		BackendID: backendID,
	}

	if len(extraParams) > 0 {
		route.RouteType = extraParams[0]
	}

	return route
}

func NewRedirectRoute(redirectTo string, extraParams ...string) Route {
	route := Route{
		Handler:      "redirect",
		RedirectTo:   redirectTo,
		RedirectType: "permanent",
		RouteType:    "exact",
	}

	if len(extraParams) > 0 {
		route.RouteType = extraParams[0]
	}
	if len(extraParams) > 1 {
		route.RedirectType = extraParams[1]
	}
	if len(extraParams) > 2 {
		route.SegmentsMode = extraParams[2]
	}

	return route
}

func NewGoneRoute(extraParams ...string) Route {
	route := Route{
		Handler: "gone",
	}

	if len(extraParams) > 0 {
		route.RouteType = extraParams[0]
	}

	return route
}

func initRouteHelper() error {
	databaseUrl := os.Getenv("ROUTER_MONGO_URL")

	if databaseUrl == "" {
		databaseUrl = "127.0.0.1"
	}

	sess, err := mgo.Dial(databaseUrl)
	if err != nil {
		return fmt.Errorf("Failed to connect to mongo: " + err.Error())
	}
	sess.SetSyncTimeout(10 * time.Minute)
	sess.SetSocketTimeout(10 * time.Minute)

	routerDB = sess.DB("router_test")
	return nil
}

func addBackend(id, url string) {
	err := routerDB.C("backends").Insert(bson.M{"backend_id": id, "backend_url": url})
	Expect(err).To(BeNil())
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	err := routerDB.C("routes").Insert(route)
	Expect(err).To(BeNil())
}

func clearRoutes() {
	_ = routerDB.C("routes").DropCollection()
	_ = routerDB.C("backends").DropCollection()
}
