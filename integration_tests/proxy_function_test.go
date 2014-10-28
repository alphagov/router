package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Functioning as a reverse proxy", func() {

	Describe("connecting to the backend", func() {
		It("should return a 502 if the connection to the backend is refused", func() {
			addBackend("not-running", "http://localhost:3164/")
			addBackendRoute("/not-running", "not-running")
			reloadRoutes()

			req, err := http.NewRequest("GET", routerURL("/not-running"), nil)
			Expect(err).To(BeNil())
			req.Header.Set("X-Varnish", "12345678")

			resp := doRequest(req)
			Expect(resp.StatusCode).To(Equal(502))

			logDetails := lastRouterErrorLogEntry()
			Expect(logDetails.Fields).To(Equal(map[string]interface{}{
				"error":          "dial tcp 127.0.0.1:3164: connection refused",
				"request":        "GET /not-running HTTP/1.1",
				"request_method": "GET",
				"status":         float64(502), // All numbers in JSON are floating point
				"upstream_addr":  "localhost:3164",
				"varnish_id":     "12345678",
			}))
			Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})

		if os.Getenv("RUN_FIREWALL_DEPENDENT_TESTS") != "" {
			// This test requires a firewall block rule for connections to localhost:3170
			// This is necessary to simulate a connection timeout
			It("should log and return a 504 if the connection times out in the configured time", func() {
				startRouter(3167, 3166, envMap{"ROUTER_BACKEND_CONNECT_TIMEOUT": "0.3s"})
				defer stopRouter(3167)
				addBackend("firewall-blocked", "http://localhost:3170/")
				addBackendRoute("/blocked", "firewall-blocked")
				reloadRoutes(3166)

				req, err := http.NewRequest("GET", routerURL("/blocked", 3167), nil)
				Expect(err).To(BeNil())
				req.Header.Set("X-Varnish", "12341111")

				start := time.Now()
				resp := doRequest(req)
				duration := time.Now().Sub(start)

				Expect(resp.StatusCode).To(Equal(504))
				Expect(duration).To(BeNumerically("~", 320*time.Millisecond, 20*time.Millisecond)) // 300 - 340 ms

				logDetails := lastRouterErrorLogEntry()
				Expect(logDetails.Fields).To(Equal(map[string]interface{}{
					"error":          "dial tcp 127.0.0.1:3170: i/o timeout",
					"request":        "GET /blocked HTTP/1.1",
					"request_method": "GET",
					"status":         float64(504), // All numbers in JSON are floating point
					"upstream_addr":  "localhost:3170",
					"varnish_id":     "12341111",
				}))
				Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
			})
		} else {
			PIt("connect timeout requires firewall block rule")
		}

		Describe("response header timeout", func() {
			var (
				tarpit1 *httptest.Server
				tarpit2 *httptest.Server
			)

			BeforeEach(func() {
				startRouter(3167, 3166, envMap{"ROUTER_BACKEND_HEADER_TIMEOUT": "0.3s"})
				tarpit1 = startTarpitBackend(time.Second)
				tarpit2 = startTarpitBackend(100*time.Millisecond, 500*time.Millisecond)
				addBackend("tarpit1", tarpit1.URL)
				addBackend("tarpit2", tarpit2.URL)
				addBackendRoute("/tarpit1", "tarpit1")
				addBackendRoute("/tarpit2", "tarpit2")
				reloadRoutes(3166)
			})

			AfterEach(func() {
				tarpit1.Close()
				tarpit2.Close()
				stopRouter(3167)
			})

			It("should log and return a 504 if a backend takes longer than the configured response timeout to start returning a response", func() {
				req := newRequest("GET", routerURL("/tarpit1", 3167))
				req.Header.Set("X-Varnish", "12341112")
				resp := doRequest(req)
				Expect(resp.StatusCode).To(Equal(504))

				logDetails := lastRouterErrorLogEntry()
				tarpitURL, _ := url.Parse(tarpit1.URL)
				Expect(logDetails.Fields).To(Equal(map[string]interface{}{
					"error":          "net/http: timeout awaiting response headers",
					"request":        "GET /tarpit1 HTTP/1.1",
					"request_method": "GET",
					"status":         float64(504), // All numbers in JSON are floating point
					"upstream_addr":  tarpitURL.Host,
					"varnish_id":     "12341112",
				}))
				Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
			})

			It("should still return the response if the body takes longer than the header timeout", func() {
				resp := routerRequest("/tarpit2", 3167)
				Expect(resp.StatusCode).To(Equal(200))
				Expect(readBody(resp)).To(Equal("Tarpit\n"))
			})
		})
	})
})
