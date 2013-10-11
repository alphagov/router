package main

import (
	"flag"
	"log"
	"net/http"
	"runtime"
)

var pubAddr = flag.String("pubAddr", ":8080", "Address on which to serve public requests")
var apiAddr = flag.String("apiAddr", ":8081", "Address on which to receive reload requests")
var mongoUrl = flag.String("mongoUrl", "localhost", "Address of mongo cluster (e.g. 'mongo1,mongo2,mongo3')")
var mongoDbName = flag.String("mongoDbName", "router", "Name of mongo database to use")

var quit = make(chan int)

func main() {
	// Use all available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	rout := NewRouter(*mongoUrl, *mongoDbName)
	rout.ReloadRoutes()

	go func() {
		err := http.ListenAndServe(*pubAddr, rout)
		if err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("router: listening for requests on " + *pubAddr)

	// This applies to DefaultServeMux, below.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		rout.ReloadRoutes()
	})
	go func() {
		err := http.ListenAndServe(*apiAddr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("router: listening for refresh on " + *apiAddr)

	<-quit
}
