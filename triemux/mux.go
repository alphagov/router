// Package triemux implements an HTTP multiplexer, or URL router, which can be
// used to serve responses from multiple distinct handlers within a single URL
// hierarchy.
package triemux

import (
	"crypto/sha1"
	"github.com/alphagov/router/trie"
	"hash"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type Mux struct {
	mu         sync.RWMutex
	exactTrie  *trie.Trie
	prefixTrie *trie.Trie
	count      int
	checksum   hash.Hash
}

type muxEntry struct {
	prefix  bool
	handler http.Handler
	methods []string
}

type muxError struct {
	code    int
	message string
}

// NewMux makes a new empty Mux.
func NewMux() *Mux {
	return &Mux{exactTrie: trie.NewTrie(), prefixTrie: trie.NewTrie(), checksum: sha1.New()}
}

// ServeHTTP dispatches the request to a backend with a registered route
// matching the request path, or 404s.
//
// If the routing table is empty, return a 503.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.count == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	handler, err := mux.lookup(r.Method, r.URL.Path)
	if err != nil {
		http.Error(w, err.message, err.code)
		return
	}

	handler.ServeHTTP(w, r)
}

// lookup takes a method and path and looks up its registered entry in the mux trie,
// returning the handler for that path, if any matches.
func (mux *Mux) lookup(method string, path string) (http.Handler, *muxError) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	pathSegments := splitpath(path)
	val, ok := mux.exactTrie.Get(pathSegments)
	if !ok {
		val, ok = mux.prefixTrie.GetLongestPrefix(pathSegments)
	}

	if !ok {
		return nil, &muxError{ code: http.StatusNotFound, message: "404 Not Found" }
	}

	entry, ok := val.(muxEntry)
	if !ok {
		log.Printf("lookup: got value (%v) from trie that wasn't a muxEntry!", val)
		return nil, &muxError{ code: http.StatusInternalServerError, message: "500 Internal Server Error" }
	}

	if !entry.methodSupported(method) {
		return nil, &muxError{ code: http.StatusMethodNotAllowed, message: "405 Method Not Allowed" }
	}

	return entry.handler, nil
}

// Handle registers the specified route (either an exact or a prefix route)
// and associates it with the specified handler. Requests through the mux for
// paths matching the route will be passed to that handler.
func (mux *Mux) Handle(path string, methods []string, prefix bool, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	// For faster lookups and consistent checksums
	sort.Strings(methods)

	mux.addToStats(path, prefix, methods)
	if prefix {
		mux.prefixTrie.Set(splitpath(path), muxEntry{prefix, handler, methods})
	} else {
		mux.exactTrie.Set(splitpath(path), muxEntry{prefix, handler, methods})
	}
}

func (mux *Mux) addToStats(path string, prefix bool, methods []string) {
	mux.count++
	mux.checksum.Write([]byte(path))
	if prefix {
		mux.checksum.Write([]byte("(true)"))
	} else {
		mux.checksum.Write([]byte("(false)"))
	}
	mux.checksum.Write([]byte(strings.Join(methods,",")))
}

func (mux *Mux) RouteCount() int {
	return mux.count
}

func (mux *Mux) RouteChecksum() []byte {
	return mux.checksum.Sum(nil)
}

func (e muxEntry) methodSupported(method string) bool {
	if len(e.methods) == 0 {
		// If routes don't specify the methods they allow, we assume they
		// allow all methods.
		return true
	}
	return contains(e.methods, method)
}

// contains searches a sorted list efficiently using the sort package
func contains(sortedStrings []string, s string) bool {
	i := sort.SearchStrings(sortedStrings, s)
	return i < len(sortedStrings) && sortedStrings[i] == s
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
