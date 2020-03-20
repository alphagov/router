package main

import (
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newAPIHandler(rout *Router) (api http.Handler, err error) {
	if err != nil {
		return nil, err
	}
	reloadChan := make(chan bool, 1)
	go func(r chan bool) {
		// This goroutine blocks until it receives a message on reloadChan,
		// and will immediately reload again if another message was received
		// during reload.
		for range r {
			logInfo("router: reload triggered")
			rout.ReloadRoutes()
		}
	}(reloadChan)

	mux := http.NewServeMux()

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Send a message to the goroutine which will start reloading immediately.
		// If the channel is already full, no message will be sent and the request
		// won't be blocked.
		select {
		case reloadChan <- true:
		default:
		}
		logInfo("router: reload queued")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Reload queued"))
	})
	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		stats := make(map[string]map[string]interface{})
		stats["routes"] = rout.RouteStats()

		jsonData, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(jsonData)
		w.Write([]byte("\n"))
	})
	mux.HandleFunc("/memory-stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)

		jsonData, err := json.MarshalIndent(memStats, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(jsonData)
		w.Write([]byte("\n"))
	})
	mux.Handle("/metrics", promhttp.Handler())

	return mux, nil
}
