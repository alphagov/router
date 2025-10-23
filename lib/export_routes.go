package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ExportRoutes queries the database and writes routes to the provided writer in JSONL format
func ExportRoutes(writer io.Writer, logger zerolog.Logger) error {
	databaseURL := os.Getenv("CONTENT_STORE_DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("CONTENT_STORE_DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	logger.Info().Msg("querying routes from content store")

	rows, err := pool.Query(context.Background(), loadRoutesQuery)
	if err != nil {
		return fmt.Errorf("failed to query routes: %w", err)
	}
	defer rows.Close()

	routeCount := 0
	for rows.Next() {
		route := &Route{}
		scans := []any{
			&route.BackendID,
			&route.IncomingPath,
			&route.RouteType,
			&route.RedirectTo,
			&route.SegmentsMode,
			&route.SchemaName,
			&route.Details,
		}

		err := rows.Scan(scans...)
		if err != nil {
			return fmt.Errorf("failed to scan route: %w", err)
		}

		// Serialize route to JSON
		jsonBytes, err := json.Marshal(route)
		if err != nil {
			logger.Warn().Interface("route", route).Err(err).Msg("failed to marshal route, skipping")
			continue
		}

		// Write JSON line
		if _, err := fmt.Fprintf(writer, "%s\n", jsonBytes); err != nil {
			return fmt.Errorf("failed to write route: %w", err)
		}

		routeCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating routes: %w", err)
	}

	logger.Info().Int("route_count", routeCount).Msg("exported routes")
	return nil
}
