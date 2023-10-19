package trie

import (
	"testing"
)

type method int

const (
	Set method = iota
	Del
)

type FixtureRow struct {
	method method
	path   []string
	val    interface{}
}

type Expectation struct {
	path []string
	val  interface{}
	ok   bool
}

type Example struct {
	fixtures     []FixtureRow
	expectations []Expectation
}

// In these table-driven tests, each Example consists of:
// 
//   - a slice of FixtureRows which are inserted into the Trie
//   - a set of Expectations, which consist of a Trie path and expected return
//     values and "ok" checks
//
// TestExamples iterates over each test case in `examples`, setting up the
// fixture and asserting the corresponding Expectations.
var examples = []Example{
	{ // Simple setting and getting
		[]FixtureRow{
			{Set, []string{"foo"}, "bar"},
		},
		[]Expectation{
			{[]string{"foo"}, "bar", true},
			{[]string{"baz"}, nil, false},
			{[]string{"foo", "bar"}, nil, false},
		},
	},
	{ // Multiple path components
		[]FixtureRow{
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Expectation{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"baz"}, nil, false},
			{[]string{"foo", "baz"}, nil, false},
			{[]string{"foo", "bat", "bax"}, nil, false},
			{[]string{"foo", "bar", "bax"}, nil, false},
		},
	},
	{ // Multiple values on the same path
		[]FixtureRow{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Expectation{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, nil, false},
			{[]string{"foo", "bat", "bax"}, nil, false},
			{[]string{"foo", "bar", "bax"}, nil, false},
		},
	},
	{ // Deleting
		[]FixtureRow{
			{Set, []string{"foo"}, "hello"},
			{Del, []string{"foo"}, nil},
		},
		[]Expectation{
			{[]string{"foo"}, nil, false},
		},
	},
	{ // Setting at the root
		[]FixtureRow{
			{Set, []string{}, "hello"},
		},
		[]Expectation{
			{[]string{}, "hello", true},
		},
	},
	{ // Setting nil
		[]FixtureRow{
			{Set, []string{"foo"}, nil},
		},
		[]Expectation{
			{[]string{"foo"}, nil, true},
		},
	},
}

var prefixExamples = []Example{
	{ // Multiple values on the same path
		[]FixtureRow{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar"}, 123},
		},
		[]Expectation{
			{[]string{"foo", "bar"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, "hello", true},
			{[]string{"foo", "bar", "bax"}, 123, true},
			{[]string{"bar"}, nil, false},
			{[]string{}, nil, false},
		},
	},
	{ // Multiple values on the same path with gaps
		[]FixtureRow{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar", "baz"}, 123},
		},
		[]Expectation{
			{[]string{"foo", "bar", "baz"}, 123, true},
			{[]string{"foo"}, "hello", true},
			{[]string{"foo", "baz"}, "hello", true},
			{[]string{"foo", "bar"}, "hello", true},
			{[]string{"foo", "bar", "baz", "bax"}, 123, true},
			{[]string{"bar"}, nil, false},
		},
	},
	{ // Deleting
		[]FixtureRow{
			{Set, []string{"foo"}, "hello"},
			{Set, []string{"foo", "bar", "baz"}, 123},
			{Del, []string{"foo", "bar", "baz"}, nil},
		},
		[]Expectation{
			{[]string{"foo", "bar", "baz"}, "hello", true},
		},
	},
}

func TestNew(t *testing.T) {
	path := []string{"hello", "world"}
	trie := NewTrie[interface{}]()
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
	trie := buildExampleTrie(t, ex.fixtures)
	for _, e := range ex.expectations {
		val, ok := trie.Get(e.path)
		t.Logf("trie.Get(path:%v) -> val:%v, ok:%v", e.path, val, ok)
		if ok != e.ok {
			t.Errorf("Example %d check %+v: trie.Get ok was %v (expected %v)", i, e, ok, e.ok)
		}
		if val != e.val {
			t.Errorf("Example %d check %+v: trie.Get val was %v (expected %v)", i, e, val, e.val)
		}
	}
}

func TestPrefixExamples(t *testing.T) {
	for i, ex := range prefixExamples {
		testPrefixExample(t, i, ex)
	}
}

func testPrefixExample(t *testing.T, i int, ex Example) {
	trie := buildExampleTrie(t, ex.fixtures)
	for _, e := range ex.expectations {
		val, ok := trie.GetLongestPrefix(e.path)
		t.Logf("trie.GetLongestPrefix(path:%v) -> val:%v, ok:%v", e.path, val, ok)
		if ok != e.ok {
			t.Errorf("Example %d check %+v: trie.Get ok was %v (expected %v)", i, e, ok, e.ok)
		}
		if val != e.val {
			t.Errorf("Example %d check %+v: trie.Get val was %v (expected %v)", i, e, val, e.val)
		}
	}
}

func TestDelReturnsStatus(t *testing.T) {
	trie := NewTrie[interface{}]()
	path := []string{"foo"}
	trie.Set(path, "bar")
	ok := trie.Del(path)
	if !ok {
		t.Error("trie.Del didn't return true when deleting a key that exists")
	}
	ok = trie.Del(path)
	if ok {
		t.Error("trie.Del didn't return false when deleting a key that didn't exist")
	}
}

func buildExampleTrie(t *testing.T, fixtures []FixtureRow) *Trie[interface{}] {
	trie := NewTrie[interface{}]()
	for _, f := range fixtures {
		switch f.method {
		case Set:
			trie.Set(f.path, f.val)
			t.Logf("trie.Set(path:%v, val:%v)", f.path, f.val)
		case Del:
			trie.Del(f.path)
			t.Logf("trie.Del(path:%v)", f.path)
		default:
			t.Errorf("Unrecognised method %v in FixtureRow %v", f.method, f)
		}
	}
	return trie
}
