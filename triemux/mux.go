// Package triemux implements an HTTP multiplexer, or URL router, which can be
// used to serve responses from multiple distinct handlers within a single URL
// hierarchy.
package triemux

import (
	"github.com/nickstenning/trie"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Mux struct {
	mu   sync.RWMutex
	trie *trie.Trie
}

type muxEntry struct {
	prefix  bool
	handler http.Handler
}

// NewMux makes a new empty Mux.
func NewMux() *Mux {
	return &Mux{trie: trie.NewTrie()}
}

// ServeHTTP dispatches the request to a backend with a registered route
// matching the request path, or 404s.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	entry, ok := findlongestmatch(mux.trie, path)
	return entry.handler, ok
}

// Handle registers the specified route (either an exact or a prefix route)
// and associates it with the specified handler. Requests through the mux for
// paths matching the route will be passed to that handler.
func (mux *Mux) Handle(path string, prefix bool, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	mux.trie.Set(splitpath(path), muxEntry{prefix, handler})
}

// splitpath turns a slash-delimited string into a lookup path (a slice
// containing the strings between slashes). Any leading slashes are stripped
// before the string is split.
func splitpath(path string) []string {
	for strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

// findlongestmatch will search the passed trie for the longest route matching
// the passed path, taking into account whether or not each muxEntry is a prefix
// route.
//
// The function first attempts an exact match, and if it fails to find one will
// then chop slash-delimited sections off the end of the path in an attempt to
// find a matching exact or prefix route.
func findlongestmatch(t *trie.Trie, path string) (entry muxEntry, ok bool) {
	origpath := splitpath(path)
	copypath := origpath

	// This search algorithm is potentially abusable -- it will take a
	// (relatively) long time to establish that a path with an enormous number of
	// slashes in doesn't have a corresponding route. The obvious fix is for the
	// trie to keep track of how long its longest root-to-leaf path is and
	// shortcut the lookup by chopping the appropriate number of elements off the
	// end of the lookup.
	//
	// Worrying about the above is probably premature optimization, so I leave the
	// mitigation described as an exercise for the reader.
	for len(copypath) >= 0 {
		val, ok := t.Get(copypath)
		if !ok {
			if len(copypath) > 0 {
				copypath = copypath[:len(copypath)-1]
				continue
			}
			break
		}

		ent, ok := val.(muxEntry)
		if !ok {
			log.Printf("findlongestmatch: got value (%v) from trie that wasn't a muxEntry!", val)
			break
		}

		if len(copypath) == len(origpath) {
			return ent, true
		}

		if ent.prefix {
			return ent, true
		}

		if len(copypath) > 0 {
			copypath = copypath[:len(copypath)-1]
			continue
		}

		// Fell through without finding anything or explicitly calling continue, so:
		break
	}
	return muxEntry{}, false
}
