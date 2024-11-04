package router

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/router/handlers"
	"github.com/alphagov/router/logger"
)

func loadBackendsFromEnv(connTimeout, headerTimeout time.Duration, logger logger.Logger) (backends map[string]http.Handler) {
	backends = make(map[string]http.Handler)

	for _, envvar := range os.Environ() {
		pair := strings.SplitN(envvar, "=", 2)

		if !strings.HasPrefix(pair[0], "BACKEND_URL_") {
			continue
		}

		backendID := strings.TrimPrefix(pair[0], "BACKEND_URL_")
		backendURL := pair[1]

		if backendURL == "" {
			logWarn(fmt.Errorf("router: couldn't find URL for backend %s, skipping", backendID))
			continue
		}

		backend, err := url.Parse(backendURL)
		if err != nil {
			logWarn(fmt.Errorf("router: couldn't parse URL %s for backend %s (error: %w), skipping", backendURL, backendID, err))
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
