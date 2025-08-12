package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	// revive:disable:dot-imports
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
	"github.com/onsi/gomega/ghttp"
)

var backends = map[string]string{
	"backend-1":   "127.0.0.1:6789",
	"backend-2":   "127.0.0.1:6790",
	"outer":       "127.0.0.1:6792",
	"inner":       "127.0.0.1:6793",
	"innerer":     "127.0.0.1:6794",
	"root":        "127.0.0.1:6795",
	"other":       "127.0.0.1:6796",
	"fallthrough": "127.0.0.1:6797",
	"down":        "127.0.0.1:6798",
	"slow-1":      "127.0.0.1:6799",
	"slow-2":      "127.0.0.1:6800",
	"backend":     "127.0.0.1:6801",
	"be":          "127.0.0.1:6802",
	"not-running": "127.0.0.1:6803",
	"with-path":   "127.0.0.1:6804",
}

func startSimpleBackend(identifier, host string) *httptest.Server {
	listenConfig := net.ListenConfig{}
	l, err := listenConfig.Listen(context.Background(), "tcp", host)
	Expect(err).NotTo(HaveOccurred())

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(identifier))
		Expect(err).NotTo(HaveOccurred())
	}))
	_ = ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	return ts
}

func startTarpitBackend(host string, delays ...time.Duration) *httptest.Server {
	responseDelay := 2 * time.Second
	if len(delays) > 0 {
		responseDelay = delays[0]
	}
	bodyDelay := 0 * time.Second
	if len(delays) > 1 {
		bodyDelay = delays[1]
	}

	listenConfig := net.ListenConfig{}
	l, err := listenConfig.Listen(context.Background(), "tcp", host)
	Expect(err).NotTo(HaveOccurred())

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	_ = ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	return ts
}

func startRecordingBackend(tls bool, host string) *ghttp.Server {
	listenConfig := net.ListenConfig{}
	l, err := listenConfig.Listen(context.Background(), "tcp", host)
	Expect(err).NotTo(HaveOccurred())

	ts := ghttp.NewUnstartedServer()
	_ = ts.HTTPTestServer.Listener.Close()
	ts.HTTPTestServer.Listener = l
	if tls {
		ts.HTTPTestServer.StartTLS()
	} else {
		ts.Start()
	}

	ts.AllowUnhandledRequests = true
	ts.UnhandledRequestStatusCode = http.StatusOK
	return ts
}
