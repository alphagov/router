package main

import (
	"flag"
	"labix.org/v2/mgo"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "Address on which to serve public requests")
var apiAddr = flag.String("apiAddr", ":8081", "Address on which to receive reload requests")
var mongo = flag.String("mongo", "localhost", "Address of mongo cluster (e.g. 'mongo1,mongo2,mongo3')")
var dbName = flag.String("dbName", "router", "Name of mongo database to use")

var quit = make(chan int)

func main() {
	flag.Parse()

	log.Println("mgo: connecting to", *mongo)
	sess, err := mgo.Dial(*mongo)
	if err != nil {
		log.Fatalln("mgo:", err)
	}
	defer sess.Close()
	sess.SetMode(mgo.Monotonic, true)

	rout := Router{db: sess.DB(*dbName)}
	rout.ReloadRoutes()

	log.Println("router: listening for requests on " + *addr)
	log.Println("router: listening for refresh on " + *apiAddr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		rout.ReloadRoutes()
	})

	go http.ListenAndServe(*addr, &rout)
	go http.ListenAndServe(*apiAddr, nil)

	<-quit
}
