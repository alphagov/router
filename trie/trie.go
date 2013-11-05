// Package trie implements a simple trie data structure that maps "paths" (which
// are slices of strings) to arbitrary data values (type interface{}).
package trie

type trieChildren map[string]*Trie

type Trie struct {
	Leaf     bool
	Entry    interface{}
	Children trieChildren
}

// NewTrie makes a new empty Trie
func NewTrie() *Trie {
	return &Trie{
		Children: make(trieChildren),
	}
}

// Get retrieves an element from the Trie
//
// Takes a path (which can be empty, to denote the root element of the Trie),
// and returns the object if the path exists in the Trie, or nil and a status of
// false. Example:
//
//     if res, ok := trie.Get([]string{"foo", "bar"}), ok {
//       fmt.Println("Value at /foo/bar was", res)
//     }
func (t *Trie) Get(path []string) (entry interface{}, ok bool) {
	if len(path) == 0 {
		return t.getentry()
	}

	key := path[0]
	newpath := path[1:]

	res, ok := t.Children[key]
	if !ok {
		// Path doesn't exist: shortcut return value
		return nil, false
	}

	return res.Get(newpath)
}

// GetLongestPrefix retrieves an element from the Trie
//
// Takes a path (which can be empty, to denote the root element of the Trie).
// If a matching object exists, it is returned. Otherwise the object with the
// longest matching prefix is returned. If nothing matches at all, nil and a
// status of false is returned. Example:
//
//     if res, ok := trie.GetLongestPrefix([]string{"foo", "bar"}), ok {
//       fmt.Println("Value at /foo/bar was", res)
//     }
func (t *Trie) GetLongestPrefix(path []string) (entry interface{}, ok bool) {
	if len(path) == 0 {
		return t.getentry()
	}

	key := path[0]
	newpath := path[1:]

	res, ok := t.Children[key]
	if !ok {
		// Path doesn't exist: return this node as possible best match
		return t.getentry()
	}

	entry, ok = res.GetLongestPrefix(newpath)
	if ok {
		return entry, ok
	}
	// We haven't found a match yet, return this node
	return t.getentry()
}

// Set creates an element in the Trie
//
// Takes a path (which can be empty, to denote the root element of the Trie),
// and an arbitrary value (interface{}) to use as the leaf data.
func (t *Trie) Set(path []string, value interface{}) {
	if len(path) == 0 {
		t.setentry(value)
		return
	}

	key := path[0]
	newpath := path[1:]

	res, ok := t.Children[key]
	if !ok {
		// Trie node that should hold entry doesn't already exist, so let's create it
		res = NewTrie()
		t.Children[key] = res
	}

	res.Set(newpath, value)
}

// Del removes an element from the Trie. Returns a boolean indicating whether an
// element was actually deleted.
func (t *Trie) Del(path []string) bool {
	if len(path) == 0 {
		return t.delentry()
	}

	key := path[0]
	newpath := path[1:]

	res, ok := t.Children[key]
	if !ok {
		return false
	}

	return res.Del(newpath)
}

func (t *Trie) setentry(value interface{}) {
	t.Leaf = true
	t.Entry = value
}

func (t *Trie) getentry() (entry interface{}, ok bool) {
	if t.Leaf {
		return t.Entry, true
	}
	return nil, false
}

func (t *Trie) delentry() (ok bool) {
	ok = t.Leaf
	t.Leaf = false
	t.Entry = nil
	return
}
