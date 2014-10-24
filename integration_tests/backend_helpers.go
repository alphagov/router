package integration

import (
	"net/http"
	"net/http/httptest"
)

func startSimpleBackend(identifier string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(identifier))
	}))
}
