package triemux

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

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

type DummyHandler struct {
	id string
}

func (dh *DummyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

var a, b, c *DummyHandler = &DummyHandler{"a"}, &DummyHandler{"b"}, &DummyHandler{"c"}

type Registration struct {
	path    string
	prefix  bool
	handler http.Handler
}

type Check struct {
	path    string
	ok      bool
	handler http.Handler
}

type LookupExample struct {
	registrations []Registration
	checks        []Check
}

var lookupExamples = []LookupExample{
	{ // simple routes
		registrations: []Registration{
			{"/foo", false, a},
			{"/bar", false, b},
		},
		checks: []Check{
			{"/foo", true, a},
			{"/bar", true, b},
			{"/baz", false, nil},
		},
	},
	{ // a prefix route
		registrations: []Registration{
			{"/foo", true, a},
			{"/bar", false, b},
		},
		checks: []Check{
			{"/foo", true, a},
			{"/bar", true, b},
			{"/baz", false, nil},
			{"/foo/bar", true, a},
		},
	},
	{ // a prefix route with an exact route child
		registrations: []Registration{
			{"/foo", true, a},
			{"/foo/bar", false, b},
		},
		checks: []Check{
			{"/foo", true, a},
			{"/foo/baz", true, a},
			{"/foo/bar", true, b},
			{"/foo/bar/bat", true, a},
		},
	},
	{ // a prefix route with an exact route child with a prefix route child
		registrations: []Registration{
			{"/foo", true, a},
			{"/foo/bar", false, b},
			{"/foo/bar/baz", true, c},
		},
		checks: []Check{
			{"/foo", true, a},
			{"/foo/baz", true, a},
			{"/foo/bar", true, b},
			{"/foo/bar/bat", true, a},
			{"/foo/bar/baz", true, c},
			{"/foo/bar/baz/qux", true, c},
		},
	},
	{ // a prefix route with an exact route at the same level
		registrations: []Registration{
			{"/foo", false, a},
			{"/foo", true, b},
		},
		checks: []Check{
			{"/foo", true, a},
			{"/foo/baz", true, b},
			{"/foo/bar", true, b},
			{"/bar", false, nil},
		},
	},
	{ // prefix route on the root
		registrations: []Registration{
			{"/", true, a},
		},
		checks: []Check{
			{"/anything", true, a},
			{"", true, a},
			{"/the/hell", true, a},
			{"///you//", true, a},
			{"!like!", true, a},
		},
	},
	{ // exact route on the root
		registrations: []Registration{
			{"/", false, a},
			{"/foo", false, b},
		},
		checks: []Check{
			{"/", true, a},
			{"/foo", true, b},
			{"/bar", false, nil},
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
		t.Logf("Register(path:%v, prefix:%v, handler:%v)", r.path, r.prefix, r.handler)
		mux.Handle(r.path, r.prefix, r.handler)
	}
	for _, c := range ex.checks {
		handler, ok := mux.lookup(c.path)
		if ok != c.ok {
			t.Errorf("Expected lookup(%v) ok to be %v, was %v", c.path, c.ok, ok)
		}
		if handler != c.handler {
			t.Errorf("Expected lookup(%v) to map to handler %v, was %v", c.path, c.handler, handler)
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

	tm := NewMux()
	tm.Handle("/government", true, a)

	for _, l := range routes {
		tm.Handle(l, false, b)
	}
	return tm
}

// Test behaviour looking up extant urls
func BenchmarkLookup(b *testing.B) {
	b.StopTimer()
	tm := benchSetup()
	urls := loadStrings("testdata/urls")
	perm := rand.Perm(len(urls))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tm.lookup(urls[perm[i%len(urls)]])
	}
}

// Test behaviour when looking up nonexistent urls
func BenchmarkLookupBogus(b *testing.B) {
	b.StopTimer()
	tm := benchSetup()
	urls := loadStrings("testdata/bogus")
	perm := rand.Perm(len(urls))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tm.lookup(urls[perm[i%len(urls)]])
	}
}

// Test worst-case lookup behaviour (see comment in findlongestmatch for
// details)
func BenchmarkLookupMalicious(b *testing.B) {
	b.StopTimer()
	tm := benchSetup()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tm.lookup("/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/")
	}
}
