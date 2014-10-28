package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/onsi/gomega/ghttp"
)

func startSimpleBackend(identifier string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(identifier))
	}))
}

func startTarpitBackend(delays ...time.Duration) *httptest.Server {
	responseDelay := 2 * time.Second
	if len(delays) > 0 {
		responseDelay = delays[0]
	}
	bodyDelay := 0 * time.Second
	if len(delays) > 1 {
		bodyDelay = delays[1]
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := "Tarpit\n"

		if responseDelay > 0 {
			time.Sleep(responseDelay)
		}
		w.Header().Add("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()

		if bodyDelay > 0 {
			time.Sleep(bodyDelay)
		}
		w.Write([]byte(body))
	}))
}

func startRecordingBackend() *ghttp.Server {
	server := ghttp.NewServer()
	server.AllowUnhandledRequests = true
	server.UnhandledRequestStatusCode = http.StatusOK
	return server
}
