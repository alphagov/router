package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/alphagov/router/handlers"
)

type CSRoute struct {
	Path         *string
	MatchType    *string
	Backend      *string
	Destination  *string
	SegmentsMode *string
}

type ContentStoreMux struct{}

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

	if *route.Backend == "redirect" {
		handler = handlers.NewRedirectHandler(path, *route.Destination, shouldPreserveSegments(*route.MatchType, *route.SegmentsMode))
	} else if *route.Backend == "gone" {
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
	requestURL := fmt.Sprintf("http://content-store/routes?path=%s", path)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}

	// Add authorization header with bearer token
	bearerToken := os.Getenv("CONTENT_STORE_BEARER_TOKEN")
	if bearerToken == "" {
		return nil, fmt.Errorf("environment variable CONTENT_STORE_BEARER_TOKEN is not set")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var route CSRoute
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return &route, nil
}
