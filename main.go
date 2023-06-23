package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/alext/tablecloth"
	"github.com/alphagov/router-postgres/handlers"
)

var (
	pubAddr               = getenvDefault("ROUTER_PUBADDR", ":8080")
	apiAddr               = getenvDefault("ROUTER_APIADDR", ":8081")
	postgresURL           = getenvDefault("DATABASE_URL", "postgresql://postgres@127.0.0.1:27017/router?sslmode=disable")
	postgresDbName        = getenvDefault("DATABASE_NAME", "router")
	dbPollInterval        = getenvDefault("ROUTER_POLL_INTERVAL", "2s")
	errorLogFile          = getenvDefault("ROUTER_ERROR_LOG", "STDERR")
	tlsSkipVerify         = os.Getenv("ROUTER_TLS_SKIP_VERIFY") != ""
	enableDebugOutput     = os.Getenv("DEBUG") != ""
	backendConnectTimeout = getenvDefault("ROUTER_BACKEND_CONNECT_TIMEOUT", "1s")
	backendHeaderTimeout  = getenvDefault("ROUTER_BACKEND_HEADER_TIMEOUT", "20s")
)

func usage() {
	helpstring := `
GOV.UK Router %s
Usage: %s [-version]

The following environment variables and defaults are available:

ROUTER_PUBADDR=:8080             Address on which to serve public requests
ROUTER_APIADDR=:8081             Address on which to receive reload requests
ROUTER_MONGO_URL=127.0.0.1       Address of mongo cluster (e.g. 'mongo1,mongo2,mongo3')
ROUTER_MONGO_DB=router           Name of mongo database to use
ROUTER_MONGO_POLL_INTERVAL=2s    Interval to poll mongo for route changes
ROUTER_ERROR_LOG=STDERR          File to log errors to (in JSON format)
DEBUG=                           Whether to enable debug output - set to anything to enable

Timeouts: (values must be parseable by http://golang.org/pkg/time/#ParseDuration)

ROUTER_BACKEND_CONNECT_TIMEOUT=1s  Connect timeout when connecting to backends
ROUTER_BACKEND_HEADER_TIMEOUT=15s  Timeout for backend response headers to be returned
`
	fmt.Fprintf(os.Stderr, helpstring, versionInfo(), os.Args[0])
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
	tablecloth.StartupDelay = 60 * time.Second
	err := tablecloth.ListenAndServe(addr, handler, ident)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	returnVersion := flag.Bool("version", false, "")
	flag.Usage = usage
	flag.Parse()
	if *returnVersion {
		fmt.Printf("GOV.UK Router %s\n", versionInfo())
		os.Exit(0)
	}

	initMetrics()

	logInfo(fmt.Sprintf("router: using GOMAXPROCS value of %d", runtime.GOMAXPROCS(0)))

	if tlsSkipVerify {
		handlers.TLSSkipVerify = true
		logWarn("router: Skipping verification of TLS certificates. " +
			"Do not use this option in a production environment.")
	}

	// Set working dir for tablecloth if available This is to allow restarts to
	// pick up new versions.
	// See http://godoc.org/github.com/alext/tablecloth#pkg-variables for details
	if wd := os.Getenv("GOVUK_APP_ROOT"); wd != "" {
		tablecloth.WorkingDir = wd
	}

	rout, err := NewRouter(postgresURL, postgresDbName, dbPollInterval, backendConnectTimeout, backendHeaderTimeout, errorLogFile)
	if err != nil {
		log.Fatal(err)
	}
	go rout.SelfUpdateRoutes()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go catchListenAndServe(pubAddr, rout, "proxy", wg)
	logInfo("router: listening for requests on " + pubAddr)

	api, err := newAPIHandler(rout)
	if err != nil {
		log.Fatal(err)
	}
	go catchListenAndServe(apiAddr, api, "api", wg)
	logInfo("router: listening for refresh on " + apiAddr)

	wg.Wait()
}
