package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newAPIHandler(rout *Router) (api http.Handler, err error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Send a message to the Router goroutine which will check the latest
		// oplog optime and start a reload if necessary.
		// If the channel is already full, no message will be sent and the request
		// won't be blocked.
		select {
		case rout.ReloadChan <- true:
		default:
		}
		logInfo("router: reload queued")
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("Reload queued"))
		if err != nil {
			logWarn(err)
		}
	})

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		_, err := w.Write([]byte("OK"))
		if err != nil {
			logWarn(err)
		}
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

		_, err = fmt.Fprintln(w, string(jsonData))
		if err != nil {
			logWarn(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

		_, err = w.Write(jsonData)
		if err != nil {
			logWarn(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.Handle("/metrics", promhttp.Handler())

	return mux, nil
}
