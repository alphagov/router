package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	// revive:disable:dot-imports
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
	"github.com/onsi/gomega/ghttp"
)

func startSimpleBackend(identifier string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(identifier))
		Expect(err).NotTo(HaveOccurred())
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
		_, err := w.Write([]byte(body))
		Expect(err).NotTo(HaveOccurred())
	}))
}

func startRecordingBackend() *ghttp.Server {
	return startRecordingServer(false)
}

func startRecordingTLSBackend() *ghttp.Server {
	return startRecordingServer(true)
}

func startRecordingServer(tls bool) (server *ghttp.Server) {
	if tls {
		server = ghttp.NewTLSServer()
	} else {
		server = ghttp.NewServer()
	}

	server.AllowUnhandledRequests = true
	server.UnhandledRequestStatusCode = http.StatusOK
	return server
}
