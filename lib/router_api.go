package router

import (
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewAPIHandler(rout *Router) (api http.Handler, err error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		select {
		case rout.ReloadChan <- true:
		default:
		}
		rout.Logger.Info().Msg("reload queued")
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("Reload queued"))
		if err != nil {
			rout.Logger.Warn().Err(err).Msg("failed to write response")
		}
	})

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		_, err := w.Write([]byte("OK"))
		if err != nil {
			rout.Logger.Warn().Err(err).Msg("failed to write response")
		}
	})

	mux.HandleFunc("/memory-stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)

		jsonData, err := json.MarshalIndent(memStats, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			rout.Logger.Error().Err(err).Msg("failed to marshal memory stats")
			return
		}

		_, err = w.Write(jsonData)
		if err != nil {
			rout.Logger.Warn().Err(err).Msg("failed to write response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.Handle("/metrics", promhttp.Handler())

	return mux, nil
}
