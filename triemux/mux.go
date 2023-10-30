// Package triemux implements an HTTP multiplexer, or URL router, which can be
// used to serve responses from multiple distinct handlers within a single URL
// hierarchy.
package triemux

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/trie"
)

type Mux struct {
	mu         sync.RWMutex
	defaultHandler http.Handler
	exactTrie  *trie.Trie[http.Handler]
	prefixTrie *trie.Trie[http.Handler]
	count      int
	downcaser  http.Handler
}

// NewMux makes a new empty Mux.
func NewMux(defaultHandler http.Handler) *Mux {
	return &Mux{
		defaultHandler: defaultHandler,
		exactTrie:  trie.NewTrie[http.Handler](),
		prefixTrie: trie.NewTrie[http.Handler](),
		downcaser:  handlers.NewDowncaseRedirectHandler(),
	}
}

// ServeHTTP forwards the request to the backend based on the longest-matching
// URL path prefix, or to the default backend if there is no match. Serves a
// 301 redirect to lowercase path when the URL path is entirely uppercase.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if shouldRedirToLowercasePath(r.URL.Path) {
		mux.downcaser.ServeHTTP(w, r)
		return
	}

	mux.lookup(r.URL.Path).ServeHTTP(w, r)
}

// shouldRedirToLowercasePath takes a URL path string (such as "/government/guidance")
// and returns:
//   - true, if path is in all caps; for example:
//     "/GOVERNMENT/GUIDANCE" -> true (should redirect to "/government/guidance")
//   - false, otherwise; for example:
//     "/GoVeRnMeNt/gUiDaNcE" -> false (should forward "/GoVeRnMeNt/gUiDaNcE" as-is)
func shouldRedirToLowercasePath(path string) (match bool) {
	match, _ = regexp.MatchString(`^\/[A-Z]+[A-Z\W\d]+$`, path)
	return
}

// lookup finds a URL path in the Mux and returns the corresponding handler.
func (mux *Mux) lookup(path string) http.Handler {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	pathSegments := splitPath(path)
	handler, ok := mux.exactTrie.Get(pathSegments)
	if !ok {
		handler, ok = mux.prefixTrie.GetLongestPrefix(pathSegments)
	}
	if !ok {
		return mux.defaultHandler
	}
	return handler
}

// Handle adds a route (either an exact path or a path prefix) to the Mux and
// and associates it with a handler, so that the Mux will pass matching
// requests to that handler.
func (mux *Mux) Handle(path string, prefix bool, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	t := mux.exactTrie
	if prefix {
		t = mux.prefixTrie
	}
	t.Set(splitPath(path), handler)
	mux.count++
}

func (mux *Mux) RouteCount() int {
	return mux.count
}

// splitPath turns a slash-delimited string into a lookup path (a slice
// containing the strings between slashes). splitPath omits empty items
// produced by leading, trailing, or adjacent slashes.
func splitPath(path string) []string {
	partsWithBlanks := strings.Split(path, "/")

	parts := make([]string, 0, len(partsWithBlanks))
	for _, part := range partsWithBlanks {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}
