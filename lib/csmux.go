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

type ContentStoreMux struct {
	BearerToken     string
	ContentStoreURL string
}

func NewContentStoreMux() (*ContentStoreMux, error) {
	contentStoreToken := os.Getenv("CONTENT_STORE_BEARER_TOKEN")
	if contentStoreToken == "" {
		return nil, fmt.Errorf("environment variable CONTENT_STORE_BEARER_TOKEN is not set")
	}

	contentStoreURL := os.Getenv("BACKEND_URL_content-store")
	if contentStoreURL == "" {
		return nil, fmt.Errorf("environment variable BACKEND_URL_content-store is not set")
	}

	return &ContentStoreMux{
		BearerToken:     fmt.Sprintf("Bearer %s", contentStoreToken),
		ContentStoreURL: contentStoreURL,
	}, nil
}

func (mux *ContentStoreMux) ServeHTTP(w http.ResponseWriter, req *http.Request, backends map[string]http.Handler) {
	path := req.URL.Path
	fmt.Printf("Debug: Request path: %s\n", path)

	route, err := mux.queryContentStore(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error querying content store: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if route == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	handler, err := mux.getHandler(w, req, route, backends)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding request handler: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, req)
}

func (mux *ContentStoreMux) getHandler(w http.ResponseWriter, req *http.Request, route *CSRoute, backends map[string]http.Handler) (http.Handler, error) {
	switch *route.Backend {
	case "redirect":
		return handlers.NewRedirectHandler(req.URL.Path, *route.Destination, shouldPreserveSegments(*route.MatchType, *route.SegmentsMode)), nil
	case "gone":
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "410 Gone", http.StatusGone) }), nil
	default:
		handler, exists := backends[*route.Backend]
		if !exists {
			return nil, fmt.Errorf("no handler available for: %s", *route.Backend)
		}
		return handler, nil
	}
}

func (mux *ContentStoreMux) queryContentStore(path string) (*CSRoute, error) {
	requestURL := fmt.Sprintf("%s/routes?path=%s", mux.ContentStoreURL, path)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", mux.BearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request to content store: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from content store: %d", resp.StatusCode)
	}

	var route CSRoute
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode response body from content store: %w", err)
	}

	return &route, nil
}
