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

	"github.com/getsentry/sentry-go"
	sentryzerolog "github.com/getsentry/sentry-go/zerolog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

func usage() {
	helpstring := `
GOV.UK Router %s
Usage: %s [-version] [-export-routes]

Flags:
  -version          Print version and exit
  -export-routes    Dump routes from database to stdout in JSONL format and exit

The following environment variables and defaults are available:

ROUTER_PUBADDR=:8080             Address on which to serve public requests
ROUTER_APIADDR=:8081             Address on which to receive reload requests
ROUTER_ERROR_LOG=STDERR          File to log errors to (in JSON format)
ROUTER_DEBUG=                    Enable debug output if non-empty
ROUTER_ROUTES_FILE=              Load routes from a JSONL file instead of PostgreSQL if non-empty

Timeouts: (values must be parseable by https://pkg.go.dev/time#ParseDuration)

ROUTER_BACKEND_CONNECT_TIMEOUT=1s  Connect timeout when connecting to backends
ROUTER_BACKEND_HEADER_TIMEOUT=20s  Timeout for backend response headers to be returned
ROUTER_FRONTEND_READ_TIMEOUT=60s   See https://cs.opensource.google/go/go/+/master:src/net/http/server.go?q=symbol:ReadTimeout
ROUTER_FRONTEND_WRITE_TIMEOUT=60s  See https://cs.opensource.google/go/go/+/master:src/net/http/server.go?q=symbol:WriteTimeout
ROUTER_ROUTE_RELOAD_INTERVAL=1m  Interval for periodic route reloads
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
	returnVersion := flag.Bool("version", false, "Print version and exit")
	exportRoutes := flag.Bool("export-routes", false, "Export routes from database to JSONL format and exit")
	flag.Usage = usage
	flag.Parse()

	fmt.Fprintf(os.Stderr, "GOV.UK Router %s\n", router.VersionInfo())
	if *returnVersion {
		os.Exit(0)
	}

	if *exportRoutes {
		// Configure logger for export mode (logs to stderr, routes to stdout)
		logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		if err := router.ExportRoutes(os.Stdout, logger); err != nil {
			logger.Fatal().Err(err).Msg("failed to export routes")
		}
		os.Exit(0)
	}

	// Initialize Sentry
	if err := sentry.Init(sentry.ClientOptions{}); err != nil {
		panic(err)
	}

	defer sentry.Flush(2 * time.Second)

	// Configure Sentry Zerolog Writer
	writer, err := sentryzerolog.New(sentryzerolog.Config{
		ClientOptions: sentry.ClientOptions{},
		Options: sentryzerolog.Options{
			Levels:          []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel},
			FlushTimeout:    3 * time.Second,
			WithBreadcrumbs: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = writer.Close()
	}()

	// Initialize Zerolog
	m := zerolog.MultiLevelWriter(os.Stderr, writer)
	logger := zerolog.New(m).With().Timestamp().Logger()

	var (
		pubAddr             = getenv("ROUTER_PUBADDR", ":8080")
		apiAddr             = getenv("ROUTER_APIADDR", ":8081")
		tlsSkipVerify       = os.Getenv("ROUTER_TLS_SKIP_VERIFY") != ""
		beConnTimeout       = getenvDuration("ROUTER_BACKEND_CONNECT_TIMEOUT", "1s")
		beHeaderTimeout     = getenvDuration("ROUTER_BACKEND_HEADER_TIMEOUT", "20s")
		feReadTimeout       = getenvDuration("ROUTER_FRONTEND_READ_TIMEOUT", "60s")
		feWriteTimeout      = getenvDuration("ROUTER_FRONTEND_WRITE_TIMEOUT", "60s")
		routeReloadInterval = getenvDuration("ROUTER_ROUTE_RELOAD_INTERVAL", "1m")
	)

	logger.Info().Msgf("frontend read timeout: %v", feReadTimeout)
	logger.Info().Msgf("frontend write timeout: %v", feWriteTimeout)
	logger.Info().Msgf("GOMAXPROCS value of %d", runtime.GOMAXPROCS(0))

	if tlsSkipVerify {
		handlers.TLSSkipVerify = true
		logger.Warn().Msg("skipping verification of TLS certificates; Do not use this option in a production environment.")
	}

	router.RegisterMetrics(prometheus.DefaultRegisterer)

	rout, err := router.NewRouter(router.Options{
		BackendConnTimeout:   beConnTimeout,
		BackendHeaderTimeout: beHeaderTimeout,
		RouteReloadInterval:  routeReloadInterval,
		Logger:               logger,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create router")
	}
	go rout.PeriodicRouteUpdates()

	go listenAndServeOrFatal(pubAddr, rout, feReadTimeout, feWriteTimeout)
	logger.Info().Msgf("listening for requests on %v", pubAddr)

	api, err := router.NewAPIHandler(rout)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create API handler")
	}

	logger.Info().Msgf("listening for API requests on %v", apiAddr)
	listenAndServeOrFatal(apiAddr, api, feReadTimeout, feWriteTimeout)
}
