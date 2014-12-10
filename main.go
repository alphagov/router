package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/alext/tablecloth"
)

var (
	pubAddr               = getenvDefault("ROUTER_PUBADDR", ":8080")
	apiAddr               = getenvDefault("ROUTER_APIADDR", ":8081")
	mongoUrl              = getenvDefault("ROUTER_MONGO_URL", "localhost")
	mongoDbName           = getenvDefault("ROUTER_MONGO_DB", "router")
	errorLogFile          = getenvDefault("ROUTER_ERROR_LOG", "STDERR")
	enableDebugOutput     = getenvDefault("DEBUG", "") != ""
	backendConnectTimeout = getenvDefault("ROUTER_BACKEND_CONNECT_TIMEOUT", "1s")
	backendHeaderTimeout  = getenvDefault("ROUTER_BACKEND_HEADER_TIMEOUT", "15s")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s\n", os.Args[0])
	helpstring := `
The following environment variables and defaults are available:

ROUTER_PUBADDR=:8080        Address on which to serve public requests
ROUTER_APIADDR=:8081        Address on which to receive reload requests
ROUTER_MONGO_URL=localhost  Address of mongo cluster (e.g. 'mongo1,mongo2,mongo3')
ROUTER_MONGO_DB=router      Name of mongo database to use
ROUTER_ERROR_LOG=STDERR     File to log errors to (in JSON format)
DEBUG=                      Whether to enable debug output - set to anything to enable

Timeouts: (values must be parseable by http://golang.org/pkg/time/#ParseDuration)

ROUTER_BACKEND_CONNECT_TIMEOUT=1s  Connect timeout when connecting to backends
ROUTER_BACKEND_HEADER_TIMEOUT=15s  Timeout for backend response headers to be returned
`
	fmt.Fprintf(os.Stderr, helpstring)
	os.Exit(2)
}

func getenvDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		val = defaultVal
	}

	return val
}

func logWarn(msg ...interface{}) {
	log.Println(msg...)
}

func logInfo(msg ...interface{}) {
	log.Println(msg...)
}

func logDebug(msg ...interface{}) {
	if enableDebugOutput {
		log.Println(msg...)
	}
}

func catchListenAndServe(addr string, handler http.Handler, ident string, wg *sync.WaitGroup) {
	defer wg.Done()
	err := tablecloth.ListenAndServe(addr, handler, ident)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		// Use all available cores if not otherwise specified
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	logInfo(fmt.Sprintf("router: using GOMAXPROCS value of %d", runtime.GOMAXPROCS(0)))

	// Set working dir for tablecloth if available This is to allow restarts to
	// pick up new versions.
	// See http://godoc.org/github.com/alext/tablecloth#pkg-variables for details
	if wd := os.Getenv("GOVUK_APP_ROOT"); wd != "" {
		tablecloth.WorkingDir = wd
	}

	flag.Usage = usage
	flag.Parse()

	rout, err := NewRouter(mongoUrl, mongoDbName, backendConnectTimeout, backendHeaderTimeout, errorLogFile)
	if err != nil {
		log.Fatal(err)
	}
	rout.ReloadRoutes()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go catchListenAndServe(pubAddr, rout, "proxy", wg)
	logInfo("router: listening for requests on " + pubAddr)

	api := newApiHandler(rout)
	go catchListenAndServe(apiAddr, api, "api", wg)
	logInfo("router: listening for refresh on " + apiAddr)

	wg.Wait()
}
