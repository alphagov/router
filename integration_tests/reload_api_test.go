package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("reload API endpoint", func() {

	Describe("request handling", func() {
		It("should return 202 for POST /reload", func() {
			resp := doRequest(newRequest("POST", routerURL(apiPort, "/reload")))
			Expect(resp).To(HaveHTTPStatus(202))
			Expect(readBody(resp)).To(Equal("Reload queued"))
		})

		It("should return 404 for POST /foo", func() {
			resp := doRequest(newRequest("POST", routerURL(apiPort, "/foo")))
			Expect(resp).To(HaveHTTPStatus(404))
		})

		It("should return 404 for POST /reload/foo", func() {
			resp := doRequest(newRequest("POST", routerURL(apiPort, "/reload/foo")))
			Expect(resp).To(HaveHTTPStatus(404))
		})

		It("should return 405 for GET /reload", func() {
			resp := doRequest(newRequest("GET", routerURL(apiPort, "/reload")))
			Expect(resp).To(HaveHTTPStatus(405))
			Expect(resp.Header.Get("Allow")).To(Equal("POST"))
		})

		It("eventually reloads the routes", func() {
			addRoute("/foo", NewRedirectRoute("/qux", "prefix"))
			addRoute("/bar", NewRedirectRoute("/qux", "prefix"))
			doRequest(newRequest("POST", routerURL(apiPort, "/reload")))

			Eventually(func() int {
				return routerRequest(routerPort, "/foo").StatusCode
			}, time.Second*3).Should(Equal(301))

			Eventually(func() int {
				return routerRequest(routerPort, "/bar").StatusCode
			}, time.Second*3).Should(Equal(301))
		})
	})

	Describe("healthcheck", func() {
		It("should return HTTP 200 OK on GET", func() {
			resp := doRequest(newRequest("GET", routerURL(apiPort, "/healthcheck")))
			Expect(resp).To(HaveHTTPStatus(200))
			Expect(readBody(resp)).To(Equal("OK"))
		})

		It("should return HTTP 405 Method Not Allowed on POST", func() {
			resp := doRequest(newRequest("POST", routerURL(apiPort, "/healthcheck")))
			Expect(resp).To(HaveHTTPStatus(405))
			Expect(resp.Header.Get("Allow")).To(Equal("GET"))
		})
	})

	Describe("memory stats", func() {
		It("should return memory statistics", func() {
			addRoute("/foo", NewRedirectRoute("/bar", "prefix"))
			addRoute("/baz", NewRedirectRoute("/qux", "prefix"))
			addRoute("/foo", NewRedirectRoute("/bar/baz"))
			reloadRoutes(apiPort)

			resp := doRequest(newRequest("GET", routerURL(apiPort, "/memory-stats")))
			Expect(resp).To(HaveHTTPStatus(200))

			var data map[string]interface{}
			readJSONBody(resp, &data)

			Expect(data).To(HaveKey("Alloc"))
			Expect(data).To(HaveKey("HeapInuse"))
		})
	})
})
