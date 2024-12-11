package triemux

import (
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"

	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"
)

func TestSplitPath(t *testing.T) {
	tests := []struct {
		in  string
		out []string
	}{
		{"", []string{}},
		{"/", []string{}},
		{"foo", []string{"foo"}},
		{"/foo", []string{"foo"}},
		{"/füßball", []string{"füßball"}},
		{"/foo/bar", []string{"foo", "bar"}},
		{"///foo/bar", []string{"foo", "bar"}},
		{"foo/bar", []string{"foo", "bar"}},
		{"/foo/bar/", []string{"foo", "bar"}},
		{"/foo//bar/", []string{"foo", "bar"}},
		{"/foo/////bar/", []string{"foo", "bar"}},
	}

	for _, ex := range tests {
		out := splitPath(ex.in)
		if len(out) != len(ex.out) {
			t.Errorf("splitPath(%v) was not %v", ex.in, ex.out)
		}
		for i := range ex.out {
			if out[i] != ex.out[i] {
				t.Errorf("splitPath(%v) differed from %v at component %d "+
					"(expected %v, got %v)", out, ex.out, i, ex.out[i], out[i])
			}
		}
	}
}

func TestShouldRedirToLowercasePath(t *testing.T) {
	tests := []struct {
		in  string
		out bool
	}{
		{"/GOVERNMENT/GUIDANCE", true},
		{"/GoVeRnMeNt/gUiDaNcE", false},
		{"/government/guidance", false},
	}

	for _, ex := range tests {
		out := shouldRedirToLowercasePath(ex.in)
		if out != ex.out {
			t.Errorf("shouldRedirToLowercasePath(%v): expected %v, got %v", ex.in, ex.out, out)
		}
	}
}

type DummyHandler struct{ id string }

func (dh *DummyHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

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
	beforeCount := promtest.ToFloat64(entryNotFoundCountMetric)

	for _, ex := range lookupExamples {
		testLookup(t, ex)
	}

	afterCount := promtest.ToFloat64(entryNotFoundCountMetric)
	notFoundCount := afterCount - beforeCount

	var expectedNotFoundCount int

	for _, ex := range lookupExamples {
		for _, c := range ex.checks {
			if !c.ok {
				expectedNotFoundCount++
			}
		}
	}

	if expectedNotFoundCount == 0 {
		t.Errorf("expectedNotFoundCount should not be zero")
	}

	if notFoundCount != float64(expectedNotFoundCount) {
		t.Errorf(
			"Expected notFoundCount (%f) ok to be %f",
			notFoundCount, float64(expectedNotFoundCount),
		)
	}
}

func testLookup(t *testing.T, ex LookupExample) {
	zerologger := zerolog.New(os.Stdout)
	mux := NewMux(zerologger)
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

var statsExample = []Registration{
	{"/", false, a},
	{"/foo", true, a},
	{"/bar", false, a},
}

func TestRouteCount(t *testing.T) {
	zerologger := zerolog.New(os.Stdout)
	mux := NewMux(zerologger)
	for _, reg := range statsExample {
		mux.Handle(reg.path, reg.prefix, reg.handler)
	}
	actual := mux.RouteCount()
	if actual != 3 {
		t.Errorf("Expected count to be 3, was %d", actual)
	}
}

func loadStrings(filename string) []string {
	content, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return strings.Split(string(content), "\n")
}

func benchSetup() *Mux {
	routes := loadStrings("testdata/routes")
	zerologger := zerolog.New(os.Stdout)
	tm := NewMux(zerologger)
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
