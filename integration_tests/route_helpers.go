package integration

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	routerDB *mongo.Database
	testContext context.Context = context.Background()
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

	uri := "mongodb://" + databaseUrl
	client, err := mongo.Connect(testContext, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("Failed to connect to mongo: " + err.Error())
	}
	// sess.SetSyncTimeout(10 * time.Minute)
	// sess.SetSocketTimeout(10 * time.Minute)

	routerDB = client.Database("router_test")
	return nil
}

func addBackend(id, url string) {
	_, err := routerDB.Collection("backends").InsertOne(testContext, bson.M{"backend_id": id, "backend_url": url})
	Expect(err).To(BeNil())
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	_, err := routerDB.Collection("routes").InsertOne(testContext, route)
	Expect(err).To(BeNil())
}

func clearRoutes() {
	routerDB.Collection("routes").Drop(testContext)
	routerDB.Collection("backends").Drop(testContext)
}

func clearRoutesWithOpcounterBump() {
	clearRoutes()
	addBackend("backend-1", "https://www.example.com/")
	clearRoutes()
}
