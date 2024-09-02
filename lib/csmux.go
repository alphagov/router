package csmux

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

var conn *pgx.Conn

type ContentStoreMux struct {
	backends map[string]http.Handler
}

func NewMux(backends map[string]http.Handler) *ContentStoreMux {
	return &ContentStoreMux{
		backends: backends,
	}
}

func (mux *ContentStoreMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	backend, err := queryContentStore(path)
	if err != nil {
		// Handle error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	backends := rt.loadBackends(db.C("backends"))

	// Use the backend to select the appropriate handler
	handler := backends[backend]

	// Serve the request using the selected handler
	handler.ServeHTTP(w, req)
}

func queryContentStore(path string) (string, error) {
	var err error
	conn, err = pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connection to database: %v\n", err)
		os.Exit(1)
	}

	rows, _ := conn.Query(
		context.Background(),
		`WITH unnested_routes AS (
  SELECT 
	  c.id,
	  c.rendering_app,
	  route->>'path' AS path, 
	  route->>'type' AS type
  FROM 
	  content_items c, 
	  jsonb_array_elements(c.routes) AS route
)
SELECT 
	id,
	rendering_app,
	path, 
	type
FROM 
	unnested_routes
WHERE 
	(type = 'prefix' AND 'your/given/path' LIKE path || '%')  -- Match if type is prefix and path is a prefix
	OR (type = 'exact' AND 'your/given/path' = path)         -- Match if type is exact and paths are identical
ORDER BY 
	CASE WHEN type = 'exact' THEN 1 ELSE 2 END,              -- Prioritize exact matches first
	LENGTH(path) DESC                                         -- Then order by longest prefix match
LIMIT 1;`
)
	
	return "", nil
}
