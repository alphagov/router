package main

import (
	"net/http"
)

func newApiHandler(rout *Router) http.Handler {
	mux := http.NewServeMux()

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

	return mux
}
