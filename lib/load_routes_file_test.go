package router

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/alphagov/router/triemux"
	"github.com/rs/zerolog"
)

func TestLoadRoutesFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	routesFile := filepath.Join(tmpDir, "test_routes.jsonl")

	content := `{"BackendID":"test-backend","IncomingPath":"/","RouteType":"exact","RedirectTo":null,"SegmentsMode":null,"SchemaName":null,"Details":null}
{"BackendID":"test-backend","IncomingPath":"/prefix","RouteType":"prefix","RedirectTo":null,"SegmentsMode":null,"SchemaName":null,"Details":null}
{"BackendID":null,"IncomingPath":"/redirect","RouteType":"exact","RedirectTo":"/new-location","SegmentsMode":"ignore","SchemaName":"redirect","Details":null}
{"BackendID":null,"IncomingPath":"/gone","RouteType":"exact","RedirectTo":null,"SegmentsMode":null,"SchemaName":"gone","Details":null}
`

	if err := os.WriteFile(routesFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{
		"test-backend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	err := loadRoutesFromFile(routesFile, mux, backends, logger)
	if err != nil {
		t.Fatalf("Failed to load routes from file: %v", err)
	}

	routeCount := mux.RouteCount()
	if routeCount != 4 {
		t.Errorf("Expected 4 routes, got %d", routeCount)
	}
}

func TestLoadRoutesFromFile_MissingFile(t *testing.T) {
	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{}

	err := loadRoutesFromFile("/nonexistent/file.jsonl", mux, backends, logger)
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestLoadRoutesFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	routesFile := filepath.Join(tmpDir, "invalid_routes.jsonl")

	// Write invalid JSON
	content := `{"BackendID":"test-backend","IncomingPath":"/","RouteType":"exact"}
this is not valid JSON
{"BackendID":"test-backend","IncomingPath":"/valid","RouteType":"exact"}
`

	if err := os.WriteFile(routesFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{
		"test-backend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	// Load should succeed but skip invalid lines
	err := loadRoutesFromFile(routesFile, mux, backends, logger)
	if err != nil {
		t.Fatalf("Loading should succeed with warning, got error: %v", err)
	}

	// Should have loaded 2 valid routes (first and last)
	routeCount := mux.RouteCount()
	if routeCount != 2 {
		t.Errorf("Expected 2 routes (skipping invalid JSON), got %d", routeCount)
	}
}

func TestLoadRoutesFromFile_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	routesFile := filepath.Join(tmpDir, "routes_with_empty_lines.jsonl")

	content := `{"BackendID":"test-backend","IncomingPath":"/","RouteType":"exact"}

{"BackendID":"test-backend","IncomingPath":"/second","RouteType":"exact"}

`

	if err := os.WriteFile(routesFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{
		"test-backend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	err := loadRoutesFromFile(routesFile, mux, backends, logger)
	if err != nil {
		t.Fatalf("Failed to load routes from file: %v", err)
	}

	// Should load 2 routes, skipping empty lines
	routeCount := mux.RouteCount()
	if routeCount != 2 {
		t.Errorf("Expected 2 routes (skipping empty lines), got %d", routeCount)
	}
}
