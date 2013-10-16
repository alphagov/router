package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
)

var dontQuit = make(chan int)
var (
	pubAddr     = getenvDefault("ROUTER_PUBADDR", ":8080")
	apiAddr     = getenvDefault("ROUTER_APIADDR", ":8081")
	mongoUrl    = getenvDefault("ROUTER_MONGO_URL", "localhost")
	mongoDbName = getenvDefault("ROUTER_MONGO_DB", "router")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s\n", os.Args[0])
	helpstring := `
The following environment variables and defaults are available:

ROUTER_PUBADDR=:8080        Address on which to serve public requests
ROUTER_APIADDR=:8081        Address on which to receive reload requests
ROUTER_MONGO_URL=localhost  Address of mongo cluster (e.g. 'mongo1,mongo2,mongo3')
ROUTER_MONGO_DB=router      Name of mongo database to use
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

func catchListenAndServe(addr string, handler http.Handler) {
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		// Use all available cores if not otherwise specified
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	log.Printf("router: using GOMAXPROCS value of %d", runtime.GOMAXPROCS(0))

	flag.Usage = usage
	flag.Parse()

	rout := NewRouter(mongoUrl, mongoDbName)
	rout.ReloadRoutes()

	go catchListenAndServe(pubAddr, rout)
	log.Println("router: listening for requests on " + pubAddr)

	api := newApiServeMux(rout)
	go catchListenAndServe(apiAddr, api)
	log.Println("router: listening for refresh on " + apiAddr)

	<-dontQuit
}

func newApiServeMux(rout *Router) (mux *http.ServeMux) {
	mux = http.NewServeMux()

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		rout.ReloadRoutes()
	})
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Write([]byte("OK"))
	})

	return
}
