package integration

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq" // Without which we can't use PSQL calls

	// revive:disable:dot-imports
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	routerDB *sql.DB
)

type Route struct {
	IncomingPath string
	RouteType    string
	Handler      string
	BackendID    string
	RedirectTo   string
	RedirectType string
	SegmentsMode string
	Disabled     bool
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
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		databaseURL = "postgresql://postgres@127.0.0.1:5432/router_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseURL)
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
	Expect(err).ToNot(HaveOccurred())
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	query := `
    INSERT INTO routes (incoming_path, route_type, handler, backend_id, redirect_to, redirect_type, segments_mode, disabled, created_at, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

	_, err := routerDB.Exec(
		query,
		route.IncomingPath,
		route.RouteType,
		route.Handler,
		route.BackendID,
		route.RedirectTo,
		route.RedirectType,
		route.SegmentsMode,
		route.Disabled,
		time.Now(),
		time.Now(),
	)

	Expect(err).ToNot(HaveOccurred())
}

func clearRoutes() {
	_, err := routerDB.Exec("DELETE FROM routes; DELETE FROM backends")
	Expect(err).ToNot(HaveOccurred())
}
