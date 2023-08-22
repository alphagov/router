package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/alphagov/router/handlers"
	router "github.com/alphagov/router/lib"
	"github.com/prometheus/client_golang/prometheus"
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
ROUTER_DEBUG=                    Enable debug output if non-empty

Timeouts: (values must be parseable by https://pkg.go.dev/time#ParseDuration)

ROUTER_BACKEND_CONNECT_TIMEOUT=1s  Connect timeout when connecting to backends
ROUTER_BACKEND_HEADER_TIMEOUT=15s  Timeout for backend response headers to be returned
ROUTER_FRONTEND_READ_TIMEOUT=60s   See https://cs.opensource.google/go/go/+/master:src/net/http/server.go?q=symbol:ReadTimeout
ROUTER_FRONTEND_WRITE_TIMEOUT=60s  See https://cs.opensource.google/go/go/+/master:src/net/http/server.go?q=symbol:WriteTimeout
`
	fmt.Fprintf(os.Stderr, helpstring, router.VersionInfo(), os.Args[0])
	const ErrUsage = 64
	os.Exit(ErrUsage)
}

func getenv(key string, defaultVal string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return defaultVal
}

func getenvDuration(key string, defaultVal string) time.Duration {
	s := getenv(key, defaultVal)
	return mustParseDuration(s)
}

func mustParseDuration(s string) (d time.Duration) {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func listenAndServeOrFatal(addr string, handler http.Handler, rTimeout time.Duration, wTimeout time.Duration) {
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  rTimeout,
		WriteTimeout: wTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	returnVersion := flag.Bool("version", false, "")
	flag.Usage = usage
	flag.Parse()

	fmt.Printf("GOV.UK Router %s\n", router.VersionInfo())
	if *returnVersion {
		os.Exit(0)
	}

	router.EnableDebugOutput = os.Getenv("ROUTER_DEBUG") != ""
	var (
		pubAddr         = getenv("ROUTER_PUBADDR", ":8080")
		apiAddr         = getenv("ROUTER_APIADDR", ":8081")
		postgresURL     = getenv("DATABASE_URL", "postgresql://postgres@127.0.0.1:27017/router?sslmode=disable")
		postgresDbName  = getenv("DATABASE_NAME", "router")
		dbPollInterval  = getenv("ROUTER_POLL_INTERVAL", "2s")
		errorLogFile    = getenv("ROUTER_ERROR_LOG", "STDERR")
		tlsSkipVerify   = os.Getenv("ROUTER_TLS_SKIP_VERIFY") != ""
		beConnTimeout   = getenvDuration("ROUTER_BACKEND_CONNECT_TIMEOUT", "1s")
		beHeaderTimeout = getenvDuration("ROUTER_BACKEND_HEADER_TIMEOUT", "20s")
		feReadTimeout   = getenvDuration("ROUTER_FRONTEND_READ_TIMEOUT", "60s")
		feWriteTimeout  = getenvDuration("ROUTER_FRONTEND_WRITE_TIMEOUT", "60s")
	)

	log.Printf("using frontend read timeout: %v", feReadTimeout)
	log.Printf("using frontend write timeout: %v", feWriteTimeout)
	log.Printf("using GOMAXPROCS value of %d", runtime.GOMAXPROCS(0))

	if tlsSkipVerify {
		handlers.TLSSkipVerify = true
		log.Printf("skipping verification of TLS certificates; " +
			"Do not use this option in a production environment.")
	}

	router.RegisterMetrics(prometheus.DefaultRegisterer)

	rout, err := router.NewRouter(router.Options{
		MongoURL:             postgresURL,
		MongoDBName:          postgresDbName,
		MongoPollInterval:    dbPollInterval,
		BackendConnTimeout:   beConnTimeout,
		BackendHeaderTimeout: beHeaderTimeout,
		LogFileName:          errorLogFile,
	})
	if err != nil {
		log.Fatal(err)
	}
	go rout.SelfUpdateRoutes()

	go listenAndServeOrFatal(pubAddr, rout, feReadTimeout, feWriteTimeout)
	log.Printf("router: listening for requests on %v", pubAddr)

	api, err := router.NewAPIHandler(rout)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("router: listening for API requests on %v", apiAddr)
	listenAndServeOrFatal(apiAddr, api, feReadTimeout, feWriteTimeout)
}
