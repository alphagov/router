package integration

import (
	"net/http"
	"net/http/httptest"

	"github.com/onsi/gomega/ghttp"
)

func startSimpleBackend(identifier string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(identifier))
	}))
}

func startRecordingBackend() *ghttp.Server {
	server := ghttp.NewServer()
	server.AllowUnhandledRequests = true
	server.UnhandledRequestStatusCode = http.StatusOK
	return server
}
