package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

var port = flag.Int("port", 3160, "The port to listen on")
var responseDelay = flag.Duration("response-delay", 2 * time.Second, "Delay in seconds before start of response")
var bodyDelay = flag.Duration("body-delay", 0, "Delay in seconds before start of response")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := "Tarpit\n"

		if *responseDelay > 0 {
			time.Sleep(*responseDelay)
		}
		w.Header().Add("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()

		if *bodyDelay > 0 {
			time.Sleep(*bodyDelay)
		}
		w.Write([]byte(body))
	})

	addr := fmt.Sprintf(":%d", *port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
