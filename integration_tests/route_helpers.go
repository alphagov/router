package integration

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	routerDB *sql.DB
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
	databaseUrl := os.Getenv("DATABASE_URL")

	if databaseUrl == "" {
		databaseUrl = "postgresql://postgres@127.0.0.1:5432/router?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		return fmt.Errorf("Failed to connect to Postgres: " + err.Error())
	}

	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(10)

	routerDB = db
	return nil
}

func addBackend(id, url string) {
	query := `
		INSERT INTO backends (backend_id, backend_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := routerDB.Exec(query, id, url, time.Now(), time.Now())
	Expect(err).To(BeNil())
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	query := `
		INSERT INTO routes (incoming_path, created_at, updated_at)
		VALUES ($1, $2, $3)
	`

	_, err := routerDB.Exec(query, route.IncomingPath, time.Now(), time.Now())
	Expect(err).To(BeNil())
}

func clearRoutes() {
	_, err := routerDB.Exec("DELETE FROM routes; DELETE FROM backends")
	Expect(err).To(BeNil())
}
