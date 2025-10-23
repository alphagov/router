package router

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ExportRoutes queries the database and writes routes to the specified file in JSONL format
func ExportRoutes(filePath string, logger zerolog.Logger) error {
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

	// Create the output file
	file, err := os.Create(filePath) // #nosec G304 - filePath is provided via environment variable
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Warn().Err(closeErr).Msg("failed to close output file")
		}
	}()

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
		if _, err := fmt.Fprintf(file, "%s\n", jsonBytes); err != nil {
			return fmt.Errorf("failed to write route: %w", err)
		}

		routeCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating routes: %w", err)
	}

	logger.Info().Int("route_count", routeCount).Str("file", filePath).Msg("exported routes")
	return nil
}
