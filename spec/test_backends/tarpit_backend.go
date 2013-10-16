package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var port = flag.Int("port", 3160, "The port to listen on")
var responseDelay = flag.Duration("response-delay", 2 * time.Second, "Delay in seconds before start of response")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(*responseDelay)
		fmt.Fprintln(w, "Tarpit")
	})

	addr := fmt.Sprintf(":%d", *port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
