package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	if routeCount != 6 { // 2 Extra routes loaded for the probe endpoints automatically
		t.Errorf("Expected 6 routes, got %d", routeCount)
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
	if routeCount != 4 { // 2 extra routes automatically loaded for the probe endpoints
		t.Errorf("Expected 4 routes (skipping invalid JSON), got %d", routeCount)
	}
}

func TestLoadRoutesFromFile_DoesNotIncludeProbeRoutesIfNoOtherRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	routesFile := filepath.Join(tmpDir, "test_no_routes.jsonl")

	content := ``

	if err := os.WriteFile(routesFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{
		"router-probe-backend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)

			if _, err := w.Write([]byte("router-probe-backend")); err != nil {
				fmt.Println("Failed to write to the response", err)
			}
		}),
	}

	err := loadRoutesFromFile(routesFile, mux, backends, logger)
	if err != nil {
		t.Fatalf("Failed to load routes from file: %v", err)
	}

	routeCount := mux.RouteCount()
	if routeCount != 0 {
		t.Errorf("Expected 0 routes, got %d", routeCount)
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/__probe__/gone", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected /__probe__/gone to return HTTP Code 503, got %v", rr.Code)
	}
}

func TestLoadRoutesFromFile_IncludesProbeRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	routesFile := filepath.Join(tmpDir, "test_routes.jsonl")

	content := `{"BackendID":"router-probe-backend","IncomingPath":"/","RouteType":"prefix","RedirectTo":null,"SegmentsMode":null,"SchemaName":null,"Details":null}`

	if err := os.WriteFile(routesFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := zerolog.Nop()
	mux := triemux.NewMux(logger)
	backends := map[string]http.Handler{
		"router-probe-backend": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)

			if _, err := w.Write([]byte("router-probe-backend")); err != nil {
				fmt.Println("Failed to write to the response", err)
			}
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

	type TestCase struct {
		Endpoint           string
		ExpectedStatus     int
		ExpectedBody       string
		ExpectToHitBackend bool
		RedirectLocation   *string
	}

	testCases := []*TestCase{
		{
			Endpoint:           "/__probe__/get",
			ExpectedStatus:     http.StatusOK,
			ExpectedBody:       "router-probe-backend",
			ExpectToHitBackend: true,
			RedirectLocation:   nil,
		},
		{
			Endpoint:           "/__probe__/gone",
			ExpectedStatus:     http.StatusGone,
			ExpectedBody:       "410 Gone\n",
			ExpectToHitBackend: false,
			RedirectLocation:   nil,
		},
		{
			Endpoint:           "/__probe__/router-redirect",
			ExpectedStatus:     http.StatusMovedPermanently,
			ExpectedBody:       "<a href=\"/__probe__/redirected\">Moved Permanently</a>.\n\n",
			ExpectToHitBackend: false,
			RedirectLocation:   new("/__probe__/redirected"),
		},
	}

	for _, testCase := range testCases {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testCase.Endpoint, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != testCase.ExpectedStatus {
			t.Errorf("Expected %s to return HTTP Code %d, got %v", testCase.Endpoint, testCase.ExpectedStatus, rr.Code)
		}

		body := rr.Body.String()
		if body != testCase.ExpectedBody {
			fmt.Printf("\n\nEXPECTED: [%s] - ACTUAL [%s]\n\n", testCase.ExpectedBody, body)
			t.Errorf("Expected %s to be called and return body %s result, got %s", testCase.Endpoint, testCase.ExpectedBody, body)
		}

		switch {
		case testCase.ExpectToHitBackend && body != "router-probe-backend":
			t.Errorf("Expected %s to hit the backend, but it did not", testCase.Endpoint)
		case !testCase.ExpectToHitBackend && body == "router-probe-backend":
			t.Errorf("Expected %s not to hit the backend, but it did", testCase.Endpoint)
		}

		if testCase.RedirectLocation != nil {
			locationHeader := rr.Result().Header.Get("Location")
			if locationHeader != *testCase.RedirectLocation {
				t.Errorf("Expected %s to redirect to %s, but instead it redirected to %s", testCase.Endpoint, *testCase.RedirectLocation, locationHeader)
			}
		}
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

	// Should load 4 routes (the 2 added in this test, and the 2 automatically added probe routes), skipping empty lines
	routeCount := mux.RouteCount()
	if routeCount != 4 {
		t.Errorf("Expected 4 routes (skipping empty lines), got %d", routeCount)
	}
}
