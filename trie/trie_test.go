package trie

import (
	"testing"
)

type Meth int

const (
	Set Meth = iota
	Del
)

type Pair struct {
	meth Meth
	path []string
	val  interface{}
}

type Check struct {
	path []string
	val  interface{}
	ok   bool
}

type Example struct {
	pairs  []Pair
	checks []Check
}

// Data driven testing. Each Example consists of a slice of Pairs which are
// "set" in the Trie, and a set of Checks, which consist of a Trie path and
// expected return values and "ok" checks.
//
// TestExamples iterates through the examples slice and runs all the Checks
// having set up a Trie with the appropriate Pairs.
var examples = []Example{
	{ // Simple setting and getting
		[]Pair{
			{Set, []string{"foo"}, "bar"},
		},
		[]Check{
			{[]string{"foo"}, "bar", true},
			{[]string{"baz"}, nil, false},
			{[]string{"foo", "bar"}, nil, false},
		},
	},
	{ // Multiple path components
		[]Pair{
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Check{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"baz"}, nil, false},
			{[]string{"foo", "baz"}, nil, false},
			{[]string{"foo", "bat", "bax"}, nil, false},
			{[]string{"foo", "bar", "bax"}, nil, false},
		},
	},
	{ // Multiple values on the same path
		[]Pair{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Check{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, nil, false},
			{[]string{"foo", "bat", "bax"}, nil, false},
			{[]string{"foo", "bar", "bax"}, nil, false},
		},
	},
	{ // Deleting
		[]Pair{
			{Set, []string{"foo"}, "hello"},
			{Del, []string{"foo"}, nil},
		},
		[]Check{
			{[]string{"foo"}, nil, false},
		},
	},
	{ // Setting at the root
		[]Pair{
			{Set, []string{}, "hello"},
		},
		[]Check{
			{[]string{}, "hello", true},
		},
	},
	{ // Setting nil
		[]Pair{
			{Set, []string{"foo"}, nil},
		},
		[]Check{
			{[]string{"foo"}, nil, true},
		},
	},
}

var prefixExamples = []Example{
	{ // Multiple values on the same path
		[]Pair{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Check{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, "hello", true},
			{[]string{"foo", "bar", "bax"}, 123, true},
			{[]string{"bar"}, nil, false},
			{[]string{}, nil, false},
		},
	},
	{ // Multiple values on the same path with gaps
		[]Pair{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar", "baz"}, 123},
		},
		[]Check{
			{[]string{"foo", "bar", "baz"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, "hello", true},
			{[]string{"foo", "bar"}, "hello", true},
			{[]string{"foo", "bar", "baz", "bax"}, 123, true},
			{[]string{"bar"}, nil, false},
		},
	},
	{ // Deleting
		[]Pair{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar", "baz"}, 123},
			{Del, []string{"foo", "bar", "baz"}, nil},
		},
		[]Check{
			{[]string{"foo", "bar", "baz"}, "hello", true},
		},
	},
}

func TestNew(t *testing.T) {
	path := []string{"hello", "world"}
	trie := NewTrie()
	_, ok := trie.Get(path)
	if ok {
		t.Error("An empty Trie should not contain any entries")
	}
}

func TestExamples(t *testing.T) {
	for i, ex := range examples {
		testExample(t, i, ex)
	}
}

func testExample(t *testing.T, i int, ex Example) {
	trie := buildExampleTrie(t, ex.pairs)
	for _, c := range ex.checks {
		val, ok := trie.Get(c.path)
		t.Logf("trie.Get(path:%v) -> val:%v, ok:%v", c.path, val, ok)
		if ok != c.ok {
			t.Errorf("Example %d check %+v: trie.Get ok was %v (expected %v)", i, c, ok, c.ok)
		}
		if val != c.val {
			t.Errorf("Example %d check %+v: trie.Get val was %v (expected %v)", i, c, val, c.val)
		}
	}
}

func TestPrefixExamples(t *testing.T) {
	for i, ex := range prefixExamples {
		testPrefixExample(t, i, ex)
	}
}

func testPrefixExample(t *testing.T, i int, ex Example) {
	trie := buildExampleTrie(t, ex.pairs)
	for _, c := range ex.checks {
		val, ok := trie.GetLongestPrefix(c.path)
		t.Logf("trie.GetLongestPrefix(path:%v) -> val:%v, ok:%v", c.path, val, ok)
		if ok != c.ok {
			t.Errorf("Example %d check %+v: trie.Get ok was %v (expected %v)", i, c, ok, c.ok)
		}
		if val != c.val {
			t.Errorf("Example %d check %+v: trie.Get val was %v (expected %v)", i, c, val, c.val)
		}
	}
}

func TestDelReturnsStatus(t *testing.T) {
	trie := NewTrie()
	path := []string{"foo"}
	trie.Set(path, "bar")
	ok := trie.Del(path)
	if !ok {
		t.Error("trie.Del didn't return true when deleting a key that exists!")
	}
	ok = trie.Del(path)
	if ok {
		t.Error("trie.Del didn't return false when deleting a key that didn't exist!")
	}
}

func buildExampleTrie(t *testing.T, pairs []Pair) *Trie {
	trie := NewTrie()
	for _, p := range pairs {
		switch p.meth {
		case Set:
			trie.Set(p.path, p.val)
			t.Logf("trie.Set(path:%v, val:%v)", p.path, p.val)
		case Del:
			trie.Del(p.path)
			t.Logf("trie.Del(path:%v)", p.path)
		default:
			t.Errorf("Unrecognised method %v in Pair %v", p.meth, p)
		}
	}
	return trie
}
