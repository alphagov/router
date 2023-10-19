// Package trie implements a simple trie data structure that maps "paths" (which
// are slices of strings) to values of some type T.
package trie

type trieChildren[T interface{}] map[string]*Trie[T]

type Trie[T interface{}] struct {
	leaf     bool
	entry    T
	children trieChildren[T]
}

// NewTrie makes a new, empty Trie.
func NewTrie[T interface{}]() *Trie[T] {
	return &Trie[T]{children: make(trieChildren[T])}
}

// Get retrieves an entry from the Trie. If there is no fully-matching entry,
// Get returns `(nil, false)`. `path` can be empty, to denote the root node.
//
// Example:
//
//	if res, ok := trie.Get([]string{"foo", "bar"}); ok {
//	  fmt.Println("Value at /foo/bar was", res)
//	}
func (t *Trie[T]) Get(path []string) (entry T, ok bool) {
	if len(path) == 0 {
		return t.getEntry()
	}

	key, newPath := path[0], path[1:]

	res, ok := t.children[key]
	if !ok {
		return
	}
	return res.Get(newPath)
}

// GetLongestPrefix retrieves the longest matching entry from the Trie.
//
// GetLongestPrefix returns a full match if there is one, or the entry with the
// longest matching prefix. If there is no match at all, GetLongestPrefix
// returns `(nil, false)`. `path` can be empty, to denote the root node.
//
// Example:
//
//	if res, ok := trie.GetLongestPrefix([]string{"foo", "bar"}); ok {
//	  fmt.Println("Value at /foo/bar was", res)
//	}
func (t *Trie[T]) GetLongestPrefix(path []string) (entry T, ok bool) {
	if len(path) == 0 {
		return t.getEntry()
	}

	key, newPath := path[0], path[1:]

	res, ok := t.children[key]
	if !ok {
		return t.getEntry() // Full path not found, but this is the longest match.
	}

	entry, ok = res.GetLongestPrefix(newPath)
	if ok {
		return entry, ok
	}
	return t.getEntry() // No match yet, so return this node.
}

// Set adds an entry to the Trie. `path` can be empty, to denote the root node.
func (t *Trie[T]) Set(path []string, value T) {
	if len(path) == 0 {
		t.setEntry(value)
		return
	}

	key, newPath := path[0], path[1:]

	res, ok := t.children[key]
	if !ok {
		res = NewTrie[T]()
		t.children[key] = res
	}

	res.Set(newPath, value)
}

// Del removes an entry from the Trie, returning true if it deleted an entry.
func (t *Trie[T]) Del(path []string) bool {
	if len(path) == 0 {
		return t.delEntry()
	}

	key, newPath := path[0], path[1:]

	res, ok := t.children[key]
	if !ok {
		return false
	}
	return res.Del(newPath)
}

func (t *Trie[T]) setEntry(value T) {
	t.leaf = true
	t.entry = value
}

func (t *Trie[T]) getEntry() (entry T, ok bool) {
	if t.leaf {
		return t.entry, true
	}
	return
}

func (t *Trie[T]) delEntry() (ok bool) {
	ok = t.leaf
	t.leaf = false
	var zero T
	t.entry = zero
	return
}
