package main

import (
	"fmt"
	"net/http"
)

var quit = make(chan int)

func makeDebugServer(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, name, ":", r.URL.Path)
	}
}

func main() {
	debug1 := http.NewServeMux()
	debug1.HandleFunc("/", makeDebugServer("first"))
	debug2 := http.NewServeMux()
	debug2.HandleFunc("/", makeDebugServer("second"))

	go http.ListenAndServe(":6061", debug1)
	go http.ListenAndServe(":6062", debug2)

	<-quit
}
