package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog"

	"github.com/alphagov/router/triemux"
)

// loadRoutesFromFile loads routes from a JSONL file
// Each line in the file should be a JSON object representing a Route
func loadRoutesFromFile(filePath string, mux *triemux.Mux, backends map[string]http.Handler, logger zerolog.Logger) error {
	file, err := os.Open(filePath) //nolint:gosec // filePath is from ROUTER_ROUTES_FILE env var, controlled by user
	if err != nil {
		return fmt.Errorf("failed to open routes file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn().Err(err).Msg("failed to close routes file")
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		route := &Route{}
		if err := json.Unmarshal([]byte(line), route); err != nil {
			logger.Warn().
				Int("line", lineNum).
				Str("content", line).
				Err(err).
				Msg("failed to parse route from file, skipping")
			continue
		}

		if err := addHandler(mux, route, backends, logger); err != nil {
			return fmt.Errorf("failed to add handler at line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading routes file: %w", err)
	}

	return nil
}
