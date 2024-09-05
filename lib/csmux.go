package router

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/alphagov/router/handlers"
	"github.com/jackc/pgx/v5"
)

var lookupStatement = `WITH unnested_routes AS (
  SELECT 
    c.id,
    c.rendering_app,
    c.schema_name,
    route->>'path' AS path, 
    route->>'type' AS type,
    route->>'destination' AS destination,
    route->>'segments_mode' AS segments_mode
  FROM 
    content_items c, 
    LATERAL jsonb_array_elements(c.routes || c.redirects) AS route
)
SELECT 
  path, 
  type,
  rendering_app,
  destination,
  segments_mode,
  schema_name
FROM 
  unnested_routes
WHERE 
  (type = 'prefix' AND $1 LIKE path || '%')  -- Match if type is prefix and path is a prefix
  OR (type = 'exact' AND $1 = path)          -- Match if type is exact and paths are identical
ORDER BY 
  CASE WHEN type = 'exact' THEN 1 ELSE 2 END,  -- Prioritize exact matches first
  LENGTH(path) DESC                            -- Then order by longest prefix match
LIMIT 1;`

type CSRoute struct {
	Path         *string
	Type         *string
	Backend      *string `db:"rendering_app"`
	Destination  *string
	SegmentsMode *string
	SchemaName   *string
}

type PgxIface interface {
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
}

type ContentStoreMux struct {
	pool PgxIface
}

func NewCSMux(pool PgxIface) *ContentStoreMux {
	return &ContentStoreMux{
		pool: pool,
	}
}

func (mux *ContentStoreMux) ServeHTTP(w http.ResponseWriter, req *http.Request, backends *map[string]http.Handler) {
	path := req.URL.Path
	route, err := mux.queryContentStore(path)
	if err != nil {
		// Handle error
		fmt.Fprintf(os.Stderr, "Error with quering content store: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var handler http.Handler

	if *route.SchemaName == "redirect" {
		handler = handlers.NewRedirectHandler(path, *route.Destination, shouldPreserveSegments(*route.Type, *route.SegmentsMode))
	} else if *route.SchemaName == "gone" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "410 Gone", http.StatusGone)
		})
	} else if *route.Backend != "" {
		handler = (*backends)[*route.Backend]
	}

	// Serve the request using the selected handler
	handler.ServeHTTP(w, req)
}

func (mux *ContentStoreMux) queryContentStore(path string) (*CSRoute, error) {
	var err error

	rows, err := mux.pool.Query(context.Background(), lookupStatement, path)
	if err != nil {
		return nil, err
	}

	route, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[CSRoute])
	if err != nil {
		return nil, err
	}

	return &route, nil
}
