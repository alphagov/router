package integration

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	// revive:disable:dot-imports
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	postgresContainer *postgres.PostgresContainer
	pgConn            *pgx.Conn
)

type Route struct {
	IncomingPath string `bson:"incoming_path"`
	RouteType    string `bson:"route_type"`
	Handler      string `bson:"handler"`
	BackendID    string `bson:"backend_id"`
	RedirectTo   string `bson:"redirect_to"`
	SegmentsMode string `bson:"segments_mode"`
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
		Handler:    "redirect",
		RedirectTo: redirectTo,
		RouteType:  "exact",
	}

	if len(extraParams) > 0 {
		route.RouteType = extraParams[0]
	}
	if len(extraParams) > 1 {
		route.SegmentsMode = extraParams[1]
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
	ctx := context.Background()
	databaseURL := postgresContainer.MustConnectionString(ctx)

	var err error

	pgConn, err = pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return nil
}

func addRoute(path string, route Route) {
	route.IncomingPath = path

	// basePath has to be unique, even though routes can be the same
	basePath := path + "-" + route.RouteType

	query := `
        INSERT INTO content_items (base_path, details, schema_name, rendering_app, routes, redirects) VALUES (@base_path, @details, @schema_name, @rendering_app, @routes, @redirects)
    `
	// Define the named arguments for the query.
	var args pgx.NamedArgs
	if route.Handler == "redirect" {
		args = pgx.NamedArgs{
			"base_path":     basePath,
			"details":       "{}",
			"schema_name":   route.Handler,
			"rendering_app": route.BackendID,
			"routes":        "[]",
			"redirects":     "[{\"path\":\"" + route.IncomingPath + "\",\"type\":\"" + route.RouteType + "\",\"destination\":\"" + route.RedirectTo + "\",\"segments_mode\":\"" + route.SegmentsMode + "\"}]",
		}
	} else {
		args = pgx.NamedArgs{
			"base_path":     basePath,
			"details":       "{}",
			"schema_name":   route.Handler,
			"rendering_app": route.BackendID,
			"routes":        "[{\"path\":\"" + route.IncomingPath + "\",\"type\":\"" + route.RouteType + "\"}]",
			"redirects":     "[]",
		}
	}
	// Execute the query with named arguments to insert the book details into the database.
	_, err := pgConn.Exec(context.Background(), query, args)
	Expect(err).NotTo(HaveOccurred())
}

func clearRoutes() {
	_, err := pgConn.Exec(context.Background(), "TRUNCATE content_items")
	Expect(err).NotTo(HaveOccurred())

	_, err = pgConn.Exec(context.Background(), "TRUNCATE publish_intents")
	Expect(err).NotTo(HaveOccurred())
}
