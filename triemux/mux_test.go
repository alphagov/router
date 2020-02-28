package triemux

import (
	"crypto/sha1"
	"fmt"
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
	{"/foo/bar/", []string{"foo", "bar"}},
	{"/foo//bar/", []string{"foo", "bar"}},
	{"/foo/////bar/", []string{"foo", "bar"}},
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

type MethodSet []string

var gets, empty, readonly MethodSet = MethodSet{"GET"}, MethodSet{}, MethodSet{"GET", "HEAD"}
var writeonly, readandwrite MethodSet = MethodSet{"PUT", "POST"}, MethodSet{"GET", "HEAD", "PUT", "POST"}
var get, post, put, head, delete string = "GET", "POST", "PUT", "HEAD", "DELETE"

type Registration struct {
	path    string
	methods MethodSet
	prefix  bool
	handler http.Handler
}

type Check struct {
	path    string
	method  string
	code    int
	handler http.Handler
}

type LookupExample struct {
	registrations []Registration
	checks        []Check
}

var lookupExamples = []LookupExample{
	{ // simple routes
		registrations: []Registration{
			{"/foo", empty, false, a},
			{"/bar", empty, false, b},
		},
		checks: []Check{
			{"/foo", get, 200, a},
			{"/bar", get, 200, b},
			{"/baz", get, 404, nil},
		},
	},
	{ // a prefix route
		registrations: []Registration{
			{"/foo", gets, true, a},
			{"/bar", gets, false, b},
		},
		checks: []Check{
			{"/foo", get, 200, a},
			{"/bar", get, 200, b},
			{"/baz", get, 404, nil},
			{"/foo/bar", get, 200, a},
		},
	},
	{ // a prefix route with an exact route child
		registrations: []Registration{
			{"/foo", gets, true, a},
			{"/foo/bar", gets, false, b},
		},
		checks: []Check{
			{"/foo", get, 200, a},
			{"/foo/baz", get, 200, a},
			{"/foo/bar", get, 200, b},
			{"/foo/bar/bat", get, 200, a},
		},
	},
	{ // a prefix route with an exact route child with a prefix route child
		registrations: []Registration{
			{"/foo", gets, true, a},
			{"/foo/bar", gets, false, b},
			{"/foo/bar/baz", gets, true, c},
		},
		checks: []Check{
			{"/foo", get, 200, a},
			{"/foo/baz", get, 200, a},
			{"/foo/bar", get, 200, b},
			{"/foo/bar/bat", get, 200, a},
			{"/foo/bar/baz", get, 200, c},
			{"/foo/bar/baz/qux", get, 200, c},
		},
	},
	{ // a prefix route with an exact route at the same level
		registrations: []Registration{
			{"/foo", gets, false, a},
			{"/foo", gets, true, b},
		},
		checks: []Check{
			{"/foo", get, 200, a},
			{"/foo/baz", get, 200, b},
			{"/foo/bar", get, 200, b},
			{"/bar", get, 404, nil},
		},
	},
	{ // prefix route on the root
		registrations: []Registration{
			{"/", gets, true, a},
		},
		checks: []Check{
			{"/anything", get, 200, a},
			{"", get, 200, a},
			{"/the/hell", get, 200, a},
			{"///you//",get,  200, a},
			{"!like!", get, 200, a},
		},
	},
	{ // exact route on the root
		registrations: []Registration{
			{"/", gets, false, a},
			{"/foo", gets, false, b},
		},
		checks: []Check{
			{"/", get, 200, a},
			{"/foo", get, 200, b},
			{"/bar", get, 404, nil},
		},
	},
	{ // routes with empty method sets
		registrations: []Registration{
			{"/",    empty, false, a },
			{"/foo", nil,   false, b },
		},
		checks: []Check{
			{"/",    get,    200, a   },
			{"/",    head,   200, a   },
			{"/",    post,   200, a   },
			{"/",    put,    200, a   },
			{"/",    delete, 200, a   },
			{"/foo", get,    200, b   },
			{"/foo", head,   200, b   },
			{"/foo", post,   200, b   },
			{"/foo", put,    200, b   },
			{"/foo", delete, 200, b   },
			{"/bar", get,    404, nil },
			{"/bar", head,   404, nil },
			{"/bar", post,   404, nil },
		},
	},
	{ // invalid request methods
		registrations: []Registration{
			{"/",    gets,         false, a },
			{"/foo", readonly,     false, b },
			{"/bar", writeonly,    false, b },
			{"/zat", readandwrite, false, b },
			{"/qux", empty,        false, b },
		},
		checks: []Check{
			{"/",    get,    200, a   },
			{"/",    post,   405, nil },
			{"/",    put,    405, nil },
			{"/",    head,   405, nil },
			{"/foo", get,    200, b   },
			{"/foo", head,   200, b   },
			{"/foo", post,   405, nil },
			{"/foo", put,    405, nil },
			{"/foo", delete, 405, nil },
			{"/bar", get,    405, nil },
			{"/bar", head,   405, nil },
			{"/bar", post,   200, b   },
			{"/bar", put,    200, b   },
			{"/bar", delete, 405, nil },
			{"/zat", get,    200, b   },
			{"/zat", head,   200, b   },
			{"/zat", post,   200, b   },
			{"/zat", put,    200, b   },
			{"/zat", delete, 405, nil },
			{"/qux", get,    200, b   },
			{"/qux", head,   200, b   },
			{"/qux", post,   200, b   },
			{"/qux", put,    200, b   },
			{"/qux", delete, 200, b   },
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
		mux.Handle(r.path, r.methods, r.prefix, r.handler)
	}
	for _, c := range ex.checks {
		handler, err := mux.lookup(c.method, c.path)
		if err != nil && err.code != c.code {
			t.Errorf("Expected lookup(%v) code to be %v, was %v", c.path, c.code, err.code)
		}
		if err == nil && c.code >= 400 {
			t.Errorf("Expected lookup(%v) error code to be %v, but no error was raised", c.path, c.code)
		}
		if handler != c.handler {
			t.Errorf("Expected lookup(%v) to map to handler %v, was %v", c.path, c.handler, handler)
		}
	}
}

var statsExample = []Registration{
	{"/", gets, false, a},
	{"/foo", gets, true, a},
	{"/bar", gets, false, a},
}

func TestRouteCount(t *testing.T) {
	mux := NewMux()
	for _, reg := range statsExample {
		mux.Handle(reg.path, reg.methods, reg.prefix, reg.handler)
	}
	actual := mux.RouteCount()
	if actual != 3 {
		t.Errorf("Expected count to be 3, was %d", actual)
	}
}

func TestChecksum(t *testing.T) {
	mux := NewMux()
	hash := sha1.New()
	for _, reg := range statsExample {
		mux.Handle(reg.path, reg.methods, reg.prefix, reg.handler)
		hash.Write([]byte(fmt.Sprintf("%s(%v)%s", reg.path, reg.prefix, strings.Join(reg.methods,","))))
	}
	expected := fmt.Sprintf("%x", hash.Sum(nil))
	actual := fmt.Sprintf("%x", mux.RouteChecksum())
	if expected != actual {
		t.Errorf("Expected checksum to be %s, was %s", expected, actual)
	}
}

func loadStrings(filename string) []string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return strings.Split(string(content), "\n")
}

func benchSetup() *Mux {
	routes := loadStrings("testdata/routes")

	tm := NewMux()
	tm.Handle("/government", gets, true, a)

	for _, l := range routes {
		tm.Handle(l, gets, false, b)
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
		tm.lookup(get, urls[perm[i%len(urls)]])
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
		tm.lookup(get, urls[perm[i%len(urls)]])
	}
}

// Test worst-case lookup behaviour (see comment in findlongestmatch for
// details)
func BenchmarkLookupMalicious(b *testing.B) {
	b.StopTimer()
	tm := benchSetup()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tm.lookup(get, "/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/x/")
	}
}
