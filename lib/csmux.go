package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	fmt.Printf("Debug: Request path: %s\n", path)
	route, err := mux.queryContentStore(path)
	if err != nil {
		// Handle error
		fmt.Fprintf(os.Stderr, "Error with querying content store: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var handler http.Handler

	if *route.Backend == "redirect" {
		fmt.Printf("Debug: Route backend is redirect. Destination: %s\n", *route.Destination)
		handler = handlers.NewRedirectHandler(path, *route.Destination, shouldPreserveSegments(*route.MatchType, *route.SegmentsMode))
	} else if *route.Backend == "gone" {
		fmt.Println("Debug: Route backend is gone.")
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "410 Gone", http.StatusGone)
		})
	} else if *route.Backend != "" {
		fmt.Printf("Debug: Route backend is %s\n", *route.Backend)
		handler = (*backends)[*route.Backend]
	} else {
		fmt.Println("Debug: No valid backend found.")
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Serve the request using the selected handler
	fmt.Println("Debug: Serving request using the selected handler.")
	handler.ServeHTTP(w, req)
}

func (mux *ContentStoreMux) queryContentStore(path string) (*CSRoute, error) {
	requestURL := fmt.Sprintf("http://content-store/routes?path=%s", path)
	fmt.Printf("Debug: Request URL: %s\n", requestURL)
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
	fmt.Println("Debug: Added Authorization header")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()
	fmt.Printf("Debug: Response status code: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		fmt.Println("Debug: Route not found (404).")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var route CSRoute
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	fmt.Printf("Debug: Response body: %s\n", string(bodyBytes))

	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	fmt.Printf("Debug: Decoded route: %+v\n", route)
	//if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
	//	return nil, fmt.Errorf("failed to decode response body: %w", err)
	//}
	//fmt.Printf("Debug: Decoded route: %+v\n", route)

	return &route, nil
}
