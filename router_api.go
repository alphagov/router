package main

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

func newAPIHandler(rout *Router, reloadInterval string) (api http.Handler, err error) {
	reloadDuration, err := time.ParseDuration(reloadInterval)
	if err != nil {
		return nil, err
	}
	reloadChan := make(chan bool)
	go func() {
		// Rate-limit reloads to 1 per RELOAD_INTERVAL.
		// This goroutine blocks until it receives a message on reloadChan, then
		// waits for the timeout before calling reload.
		for range reloadChan {
			time.Sleep(reloadDuration)
			rout.ReloadRoutes()
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Send a message to the reload goroutine, which will start a new timeout
		// before reloading, or do nothing if one is already in progress.
		select {
		case reloadChan <- true:
			logInfo("router: reload triggered")
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("Reload triggered"))
		default:
			logInfo("router: reload already in progress")
			w.Write([]byte("Reload already in progress"))
		}
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
			http.Error(w, err.Error(), 500)
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
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write(jsonData)
		w.Write([]byte("\n"))
	})

	return mux, nil
}
