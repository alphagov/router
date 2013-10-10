package main

import (
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
	// Use all available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	rout := NewRouter(mongoUrl, mongoDbName)
	rout.ReloadRoutes()

	go catchListenAndServe(pubAddr, rout)
	log.Println("router: listening for requests on " + pubAddr)

	// This applies to DefaultServeMux, below.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		rout.ReloadRoutes()
	})
	go catchListenAndServe(apiAddr, nil)
	log.Println("router: listening for refresh on " + apiAddr)

	<-dontQuit
}
