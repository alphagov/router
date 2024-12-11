package router

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/router/handlers"
	"github.com/rs/zerolog"
)

func loadBackendsFromEnv(connTimeout, headerTimeout time.Duration, logger zerolog.Logger) (backends map[string]http.Handler) {
	backends = make(map[string]http.Handler)

	for _, envvar := range os.Environ() {
		pair := strings.SplitN(envvar, "=", 2)

		if !strings.HasPrefix(pair[0], "BACKEND_URL_") {
			continue
		}

		backendID := strings.TrimPrefix(pair[0], "BACKEND_URL_")
		backendURL := pair[1]

		if backendURL == "" {
			logger.Warn().Msgf("no URL for backend %s provided, skipping", backendID)
			continue
		}

		backend, err := url.Parse(backendURL)
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to parse URL %s for backend %s, skipping", backendURL, backendID)
			continue
		}

		backends[backendID] = handlers.NewBackendHandler(
			backendID,
			backend,
			connTimeout,
			headerTimeout,
			logger,
		)
	}

	return
}
