package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 3160, "The port to listen on")
var identifier = flag.String("identifier", "simple_backend", "Identifier to be returned in the resposne body")

func identResponse(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, *identifier)
}

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)

	err := http.ListenAndServe(addr, http.HandlerFunc(identResponse))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
