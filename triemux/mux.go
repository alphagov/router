// Package triemux implements an HTTP multiplexer, or URL router, which can be
// used to serve responses from multiple distinct handlers within a single URL
// hierarchy.
package triemux

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/alphagov/router-postgres/logger"
	"github.com/alphagov/router-postgres/trie"

	"github.com/prometheus/client_golang/prometheus"
)

type Mux struct {
	mu         sync.RWMutex
	exactTrie  *trie.Trie
	prefixTrie *trie.Trie
	count      int
}

type muxEntry struct {
	prefix  bool
	handler http.Handler
}

// NewMux makes a new empty Mux.
func NewMux() *Mux {
	return &Mux{exactTrie: trie.NewTrie(), prefixTrie: trie.NewTrie()}
}

// ServeHTTP dispatches the request to a backend with a registered route
// matching the request path, or 404s.
//
// If the routing table is empty, return a 503.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.count == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		logger.NotifySentry(logger.ReportableError{
			Error:   logger.RecoveredError{ErrorMessage: "Route table is empty!"},
			Request: r,
		})
		tempChild, isParent := os.LookupEnv("TEMPORARY_CHILD")
		if !isParent {
			tempChild = "0"
		}
		InternalServiceUnavailableCountMetric.With(prometheus.Labels{
			"temporary_child": tempChild,
		}).Inc()
		return
	}

	handler, ok := mux.lookup(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	handler.ServeHTTP(w, r)
}

// lookup takes a path and looks up its registered entry in the mux trie,
// returning the handler for that path, if any matches.
func (mux *Mux) lookup(path string) (handler http.Handler, ok bool) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	pathSegments := splitpath(path)
	val, ok := mux.exactTrie.Get(pathSegments)
	if !ok {
		val, ok = mux.prefixTrie.GetLongestPrefix(pathSegments)
	}
	if !ok {
		EntryNotFoundCountMetric.Inc()
		return nil, false
	}

	entry, ok := val.(muxEntry)
	if !ok {
		log.Printf("lookup: got value (%v) from trie that wasn't a muxEntry!", val)
		EntryNotFoundCountMetric.Inc()
		return nil, false
	}

	return entry.handler, ok
}

// Handle registers the specified route (either an exact or a prefix route)
// and associates it with the specified handler. Requests through the mux for
// paths matching the route will be passed to that handler.
func (mux *Mux) Handle(path string, prefix bool, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	mux.count++
	if prefix {
		mux.prefixTrie.Set(splitpath(path), muxEntry{prefix, handler})
	} else {
		mux.exactTrie.Set(splitpath(path), muxEntry{prefix, handler})
	}
}

func (mux *Mux) RouteCount() int {
	return mux.count
}

// splitpath turns a slash-delimited string into a lookup path (a slice
// containing the strings between slashes). Empty items produced by
// leading, trailing, or adjacent slashes are removed.
func splitpath(path string) []string {
	partsWithBlanks := strings.Split(path, "/")

	parts := make([]string, 0, len(partsWithBlanks))
	for _, part := range partsWithBlanks {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}
