package integration

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Functioning as a reverse proxy", func() {

	Describe("connecting to the backend", func() {
		It("should return a 502 if the connection to the backend is refused", func() {
			addBackend("not-running", "http://127.0.0.1:3164/")
			addRoute("/not-running", NewBackendRoute("not-running"))
			reloadRoutes(apiPort)

			req, err := http.NewRequest(http.MethodGet, routerURL(routerPort, "/not-running"), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Varnish", "12345678")

			resp := doRequest(req)
			Expect(resp.StatusCode).To(Equal(502))

			logDetails := lastRouterErrorLogEntry()
			Expect(logDetails.Fields).To(Equal(map[string]interface{}{
				"error":          "dial tcp 127.0.0.1:3164: connect: connection refused",
				"request":        "GET /not-running HTTP/1.1",
				"request_method": "GET",
				"status":         float64(502), // All numbers in JSON are floating point
				"upstream_addr":  "127.0.0.1:3164",
				"varnish_id":     "12345678",
			}))
			Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should log and return a 504 if the connection times out in the configured time", func() {
			err := startRouter(3167, 3166, []string{"ROUTER_BACKEND_CONNECT_TIMEOUT=0.3s"})
			Expect(err).NotTo(HaveOccurred())
			defer stopRouter(3167)

			addBackend("black-hole", "http://240.0.0.0:1234/")
			addRoute("/should-time-out", NewBackendRoute("black-hole"))
			reloadRoutes(3166)

			req, err := http.NewRequest(http.MethodGet, routerURL(3167, "/should-time-out"), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Varnish", "12345678")

			start := time.Now()
			resp := doRequest(req)
			duration := time.Since(start)

			Expect(resp.StatusCode).To(Equal(504))
			Expect(duration).To(BeNumerically("~", 320*time.Millisecond, 20*time.Millisecond)) // 300 - 340 ms

			logDetails := lastRouterErrorLogEntry()
			Expect(logDetails.Fields).To(Equal(map[string]interface{}{
				"error":          "dial tcp 240.0.0.0:1234: i/o timeout",
				"request":        "GET /should-time-out HTTP/1.1",
				"request_method": "GET",
				"status":         float64(504), // All numbers in JSON are floating point
				"upstream_addr":  "240.0.0.0:1234",
				"varnish_id":     "12345678",
			}))
			Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})

		Describe("response header timeout", func() {
			var (
				tarpit1 *httptest.Server
				tarpit2 *httptest.Server
			)

			BeforeEach(func() {
				err := startRouter(3167, 3166, []string{"ROUTER_BACKEND_HEADER_TIMEOUT=0.3s"})
				Expect(err).NotTo(HaveOccurred())
				tarpit1 = startTarpitBackend(time.Second)
				tarpit2 = startTarpitBackend(100*time.Millisecond, 500*time.Millisecond)
				addBackend("tarpit1", tarpit1.URL)
				addBackend("tarpit2", tarpit2.URL)
				addRoute("/tarpit1", NewBackendRoute("tarpit1"))
				addRoute("/tarpit2", NewBackendRoute("tarpit2"))
				reloadRoutes(3166)
			})

			AfterEach(func() {
				tarpit1.Close()
				tarpit2.Close()
				stopRouter(3167)
			})

			It("should log and return a 504 if a backend takes longer than the configured response timeout to start returning a response", func() {
				req := newRequest(http.MethodGet, routerURL(3167, "/tarpit1"))
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
				resp := routerRequest(3167, "/tarpit2")
				Expect(resp.StatusCode).To(Equal(200))
				Expect(readBody(resp)).To(Equal("Tarpit\n"))
			})
		})
	})

	Describe("header handling", func() {
		var (
			recorder    *ghttp.Server
			recorderURL *url.URL
		)

		BeforeEach(func() {
			recorder = startRecordingBackend()
			recorderURL, _ = url.Parse(recorder.URL())
			addBackend("backend", recorder.URL())
			addRoute("/foo", NewBackendRoute("backend", "prefix"))
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			recorder.Close()
		})

		It("should pass through most http headers to the backend", func() {
			resp := routerRequestWithHeaders(routerPort, "/foo", map[string]string{
				"Foo":        "bar",
				"User-Agent": "Router test suite 2.7182",
			})
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.Header.Get("Foo")).To(Equal("bar"))
			Expect(beReq.Header.Get("User-Agent")).To(Equal("Router test suite 2.7182"))
		})

		It("should set the Host header to the backend hostname", func() {
			resp := routerRequestWithHeaders(routerPort, "/foo", map[string]string{
				"Host": "www.example.com",
			})
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.Host).To(Equal(recorderURL.Host))
		})

		It("should not add a default User-Agent if there isn't one in the request", func() {
			// Most http libraries add a default User-Agent header.
			resp := routerRequest(routerPort, "/foo")
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			_, ok := beReq.Header[textproto.CanonicalMIMEHeaderKey("User-Agent")]
			Expect(ok).To(BeFalse())
		})

		It("should add the client IP to X-Forwardrd-For", func() {
			resp := routerRequest(routerPort, "/foo")
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.Header.Get("X-Forwarded-For")).To(Equal("127.0.0.1"))

			resp = routerRequestWithHeaders(routerPort, "/foo", map[string]string{
				"X-Forwarded-For": "10.9.8.7",
			})
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(2))
			beReq = recorder.ReceivedRequests()[1]
			Expect(beReq.Header.Get("X-Forwarded-For")).To(Equal("10.9.8.7, 127.0.0.1"))
		})

		Describe("setting the Via header", func() {
			// See https://tools.ietf.org/html/rfc2616#section-14.45

			It("should add itself to the Via request header for an HTTP/1.1 request", func() {
				resp := routerRequest(routerPort, "/foo")
				Expect(resp.StatusCode).To(Equal(200))

				Expect(recorder.ReceivedRequests()).To(HaveLen(1))
				beReq := recorder.ReceivedRequests()[0]
				Expect(beReq.Header.Get("Via")).To(Equal("1.1 router"))

				resp = routerRequestWithHeaders(routerPort, "/foo", map[string]string{
					"Via": "1.0 fred, 1.1 barney",
				})
				Expect(resp.StatusCode).To(Equal(200))

				Expect(recorder.ReceivedRequests()).To(HaveLen(2))
				beReq = recorder.ReceivedRequests()[1]
				Expect(beReq.Header.Get("Via")).To(Equal("1.0 fred, 1.1 barney, 1.1 router"))
			})

			It("should add itself to the Via request header for an HTTP/1.0 request", func() {
				req := newRequest(http.MethodGet, routerURL(routerPort, "/foo"))
				resp := doHTTP10Request(req)
				Expect(resp.StatusCode).To(Equal(200))

				Expect(recorder.ReceivedRequests()).To(HaveLen(1))
				beReq := recorder.ReceivedRequests()[0]
				Expect(beReq.Header.Get("Via")).To(Equal("1.0 router"))

				req = newRequestWithHeaders("GET", routerURL(routerPort, "/foo"), map[string]string{
					"Via": "1.0 fred, 1.1 barney",
				})
				resp = doHTTP10Request(req)
				Expect(resp.StatusCode).To(Equal(200))

				Expect(recorder.ReceivedRequests()).To(HaveLen(2))
				beReq = recorder.ReceivedRequests()[1]
				Expect(beReq.Header.Get("Via")).To(Equal("1.0 fred, 1.1 barney, 1.0 router"))
			})

			It("should add itself to the Via response heaver", func() {
				resp := routerRequest(routerPort, "/foo")
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get("Via")).To(Equal("1.1 router"))

				recorder.AppendHandlers(ghttp.RespondWith(200, "body", http.Header{
					"Via": []string{"1.0 fred, 1.1 barney"},
				}))
				resp = routerRequest(routerPort, "/foo")
				Expect(resp.StatusCode).To(Equal(200))
				Expect(resp.Header.Get("Via")).To(Equal("1.0 fred, 1.1 barney, 1.1 router"))
			})
		})
	})

	Describe("request verb, path, query and body handling", func() {
		var (
			recorder *ghttp.Server
		)

		BeforeEach(func() {
			recorder = startRecordingBackend()
			addBackend("backend", recorder.URL())
			addRoute("/foo", NewBackendRoute("backend", "prefix"))
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			recorder.Close()
		})

		It("should use the same verb and path when proxying", func() {
			recorder.AppendHandlers(
				ghttp.VerifyRequest("POST", "/foo"),
				ghttp.VerifyRequest("DELETE", "/foo/bar/baz.json"),
			)

			req := newRequest("POST", routerURL(routerPort, "/foo"))
			resp := doRequest(req)
			Expect(resp.StatusCode).To(Equal(200))

			req = newRequest("DELETE", routerURL(routerPort, "/foo/bar/baz.json"))
			resp = doRequest(req)
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(2))
		})

		It("should pass through the query string unmodified", func() {
			recorder.AppendHandlers(
				ghttp.VerifyRequest("GET", "/foo/bar", "baz=qux"),
			)
			resp := routerRequest(routerPort, "/foo/bar?baz=qux")
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
		})

		It("should pass through the body unmodified", func() {
			recorder.AppendHandlers(func(w http.ResponseWriter, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				req.Body.Close()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("I am the request body.  Woohoo!"))
			})

			req := newRequest("POST", routerURL(routerPort, "/foo"))
			req.Body = io.NopCloser(strings.NewReader("I am the request body.  Woohoo!"))
			resp := doRequest(req)
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
		})
	})

	Describe("handling a backend with a non '/' path", func() {
		var (
			recorder *ghttp.Server
		)

		BeforeEach(func() {
			recorder = startRecordingBackend()
			addBackend("backend", recorder.URL()+"/something")
			addRoute("/foo/bar", NewBackendRoute("backend", "prefix"))
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			recorder.Close()
		})

		It("should merge the 2 paths", func() {
			resp := routerRequest(routerPort, "/foo/bar")
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.URL.RequestURI()).To(Equal("/something/foo/bar"))
		})

		It("should preserve the request query string", func() {
			resp := routerRequest(routerPort, "/foo/bar?baz=qux")
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.URL.RequestURI()).To(Equal("/something/foo/bar?baz=qux"))
		})
	})

	Describe("handling HTTP/1.0 requests", func() {
		var (
			recorder *ghttp.Server
		)

		BeforeEach(func() {
			recorder = startRecordingBackend()
			addBackend("backend", recorder.URL())
			addRoute("/foo", NewBackendRoute("backend", "prefix"))
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			recorder.Close()
		})

		It("should work with incoming HTTP/1.1 requests", func() {
			req := newRequest("GET", routerURL(routerPort, "/foo"))
			resp := doHTTP10Request(req)
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.URL.RequestURI()).To(Equal("/foo"))
		})

		It("should proxy to the backend as HTTP/1.1 requests", func() {
			req := newRequest("GET", routerURL(routerPort, "/foo"))
			resp := doHTTP10Request(req)
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.Proto).To(Equal("HTTP/1.1"))
		})
	})

	Describe("handling requests to a HTTPS backend", func() {
		var recorder *ghttp.Server

		BeforeEach(func() {
			err := startRouter(3167, 3166, []string{"ROUTER_TLS_SKIP_VERIFY=1"})
			Expect(err).NotTo(HaveOccurred())
			recorder = startRecordingTLSBackend()
			addBackend("backend", recorder.URL())
			addRoute("/foo", NewBackendRoute("backend", "prefix"))
			reloadRoutes(3166)
		})

		AfterEach(func() {
			recorder.Close()
			stopRouter(3167)
		})

		It("should correctly reverse proxy to a HTTPS backend", func() {
			req := newRequest("GET", routerURL(3167, "/foo"))
			resp := doRequest(req)
			Expect(resp.StatusCode).To(Equal(200))

			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			beReq := recorder.ReceivedRequests()[0]
			Expect(beReq.URL.RequestURI()).To(Equal("/foo"))
		})
	})
})
