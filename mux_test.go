package proxymux

import (
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
)

type SplitExample struct {
	in  string
	out []string
}

var splitExamples = []SplitExample{
	{"", []string{}},
	{"/", []string{}},
	{"foo", []string{"foo"}},
	{"/foo", []string{"foo"}},
	{"/füßball", []string{"füßball"}},
	{"/foo/bar", []string{"foo", "bar"}},
	{"///foo/bar", []string{"foo", "bar"}},
	{"foo/bar", []string{"foo", "bar"}},
	{"/foo/bar/", []string{"foo", "bar", ""}},
	{"/foo//bar/", []string{"foo", "", "bar", ""}},
}

func TestSplitpath(t *testing.T) {
	for _, ex := range splitExamples {
		testSplitpath(t, ex)
	}
}

func testSplitpath(t *testing.T, ex SplitExample) {
	out := splitpath(ex.in)
	if len(out) != len(ex.out) {
		t.Errorf("splitpath(%v) was not %v", ex.in, ex.out)
	}
	for i := range ex.out {
		if out[i] != ex.out[i] {
			t.Errorf("splitpath(%v) differed from %v at component %d "+
				"(expected %v, got %v)", out, ex.out, i, ex.out[i], out[i])
		}
	}
}

type Registration struct {
	path    string
	prefix  bool
	backend int
}

type Check struct {
	path    string
	ok      bool
	backend int
}

type LookupExample struct {
	registrations []Registration
	checks        []Check
}

var lookupExamples = []LookupExample{
	{ // simple routes
		registrations: []Registration{
			{"/foo", false, 1},
			{"/bar", false, 2},
		},
		checks: []Check{
			{"/foo", true, 1},
			{"/bar", true, 2},
			{"/baz", false, 0},
		},
	},
	{ // a prefix route
		registrations: []Registration{
			{"/foo", true, 1},
			{"/bar", false, 2},
		},
		checks: []Check{
			{"/foo", true, 1},
			{"/bar", true, 2},
			{"/baz", false, 0},
			{"/foo/bar", true, 1},
		},
	},
	{ // a prefix route with an exact route child
		registrations: []Registration{
			{"/foo", true, 1},
			{"/foo/bar", false, 2},
		},
		checks: []Check{
			{"/foo", true, 1},
			{"/foo/baz", true, 1},
			{"/foo/bar", true, 2},
			{"/foo/bar/bat", true, 1},
		},
	},
	{ // a prefix route with an exact route child with a prefix route child
		registrations: []Registration{
			{"/foo", true, 1},
			{"/foo/bar", false, 2},
			{"/foo/bar/baz", true, 3},
		},
		checks: []Check{
			{"/foo", true, 1},
			{"/foo/baz", true, 1},
			{"/foo/bar", true, 2},
			{"/foo/bar/bat", true, 1},
			{"/foo/bar/baz", true, 3},
			{"/foo/bar/baz/qux", true, 3},
		},
	},
	{ // prefix route on the root
		registrations: []Registration{
			{"/", true, 123},
		},
		checks: []Check{
			{"/anything", true, 123},
			{"", true, 123},
			{"/the/hell", true, 123},
			{"///you//", true, 123},
			{"!like!", true, 123},
		},
	},
	{ // exact route on the root
		registrations: []Registration{
			{"/", false, 123},
			{"/foo", false, 456},
		},
		checks: []Check{
			{"/", true, 123},
			{"/foo", true, 456},
			{"/bar", false, 0},
		},
	},
}

func TestLookup(t *testing.T) {
	for _, ex := range lookupExamples {
		testLookup(t, ex)
	}
}

func testLookup(t *testing.T, ex LookupExample) {
	mux := NewMux()
	for _, r := range ex.registrations {
		t.Logf("Register(path:%v, prefix:%v, backend:%v)", r.path, r.prefix, r.backend)
		mux.Register(r.path, r.prefix, r.backend)
	}
	for _, c := range ex.checks {
		id, ok := mux.Lookup(c.path)
		if ok != c.ok {
			t.Errorf("Expected lookup(%v) ok to be %v, was %v", c.path, c.ok, ok)
		}
		if id != c.backend {
			t.Errorf("Expected lookup(%v) to map to backend %d, was %d", c.path, c.backend, id)
		}
	}
}

func loadStrings(filename string) []string {
	content, err := ioutil.ReadFile("testdata/routes")
	if err != nil {
		panic(err)
	}
	return strings.Split(string(content), "\n")
}

func benchSetup() *Mux {
	routes := loadStrings("testdata/routes")

	pm := NewMux()
	pm.Register("/government", true, 123)

	for _, l := range routes {
		pm.Register(l, false, 456)
	}
	return pm
}

// Test behaviour looking up extant urls
func BenchmarkLookup(b *testing.B) {
	b.StopTimer()
	pm := benchSetup()
	urls := loadStrings("testdata/urls")
	perm := rand.Perm(len(urls))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		pm.Lookup(urls[perm[i%len(urls)]])
	}
}

// Test behaviour when looking up nonexistent urls
func BenchmarkLookupBogus(b *testing.B) {
	b.StopTimer()
	pm := benchSetup()
	urls := loadStrings("testdata/bogus")
	perm := rand.Perm(len(urls))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		pm.Lookup(urls[perm[i%len(urls)]])
	}
}

// Test worst-case lookup behaviour (see comment in findlongestmatch for
// details)
func BenchmarkLookupMalicious(b *testing.B) {
	b.StopTimer()
	pm := benchSetup()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		pm.Lookup("/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/")
	}
}
