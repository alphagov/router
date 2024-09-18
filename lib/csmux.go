package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/alphagov/router/handlers"
)

type CSRoute struct {
	Path         *string `json:"path"`
	MatchType    *string `json:"match_type"`
	Backend      *string `json:"backend"`
	Destination  *string `json:"destination"`
	SegmentsMode *string `json:"segments_mode"`
}

type ContentStoreMux struct{}

func (mux *ContentStoreMux) ServeHTTP(w http.ResponseWriter, req *http.Request, backends *map[string]http.Handler) {
	path := req.URL.Path
	fmt.Printf("Debug: Request path: %s\n", path)
	route, err := mux.queryContentStore(path)
	if err != nil {
		// Handle error
		fmt.Fprintf(os.Stderr, "Error with querying content store: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var handler http.Handler
	if route == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	} else if *route.Backend == "redirect" {
		handler = handlers.NewRedirectHandler(path, *route.Destination, shouldPreserveSegments(*route.MatchType, *route.SegmentsMode))
	} else if *route.Backend == "gone" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "410 Gone", http.StatusGone)
		})
	} else if *route.Backend != "" {
		handler = (*backends)[*route.Backend]
	}

	// Serve the request using the selected handler
	fmt.Println("Debug: Serving request using the selected handler.")
	handler.ServeHTTP(w, req)
}

func (mux *ContentStoreMux) queryContentStore(path string) (*CSRoute, error) {
	requestURL := fmt.Sprintf("http://content-store/routes?path=%s", path)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request object: %w", err)
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var route CSRoute
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return &route, nil
}
