package integration

import (
	"crypto/sha1"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("reload API endpoint", func() {

	Describe("request handling", func() {
		It("should return 200 for POST /reload", func() {
			resp := doRequest(newRequest("POST", routerAPIURL("/reload")))
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should return 404 for POST /foo", func() {
			resp := doRequest(newRequest("POST", routerAPIURL("/foo")))
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should return 404 for POST /reload/foo", func() {
			resp := doRequest(newRequest("POST", routerAPIURL("/reload/foo")))
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should return 405 for GET /reload", func() {
			resp := doRequest(newRequest("GET", routerAPIURL("/reload")))
			Expect(resp.StatusCode).To(Equal(405))
			Expect(resp.Header.Get("Allow")).To(Equal("POST"))
		})

		Context("with a non-zero reload interval", func() {
			newRouterPort := 7999
			newApiPort := 8000
			BeforeEach(func() {
				err := startRouter(newRouterPort, newApiPort, map[string]string{"ROUTER_RELOAD_INTERVAL": "100ms"})
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				stopRouter(newRouterPort)
			})

			It("should return 'already in progress' for requests within timeout", func() {
				resp := doRequest(newRequest("POST", routerURL("/reload", newApiPort)))
				resp2 := doRequest(newRequest("POST", routerURL("/reload", newApiPort)))
				Expect(readBody(resp)).To(Equal("Reload triggered"))
				Expect(readBody(resp2)).To(Equal("Reload already in progress"))
			})

			It("should return 'triggered' for requests after timeout", func() {
				resp := doRequest(newRequest("POST", routerURL("/reload", newApiPort)))
				time.Sleep(time.Second * 2)
				resp2 := doRequest(newRequest("POST", routerURL("/reload", newApiPort)))
				Expect(readBody(resp)).To(Equal("Reload triggered"))
				Expect(readBody(resp2)).To(Equal("Reload triggered"))
			})
		})
	})

	Describe("healthcheck", func() {
		It("should return 200 and sting 'OK' on /healthcheck", func() {
			resp := doRequest(newRequest("GET", routerAPIURL("/healthcheck")))
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("OK"))
		})

		It("should return 405 for other verbs", func() {
			resp := doRequest(newRequest("POST", routerAPIURL("/healthcheck")))
			Expect(resp.StatusCode).To(Equal(405))
			Expect(resp.Header.Get("Allow")).To(Equal("GET"))
		})
	})

	Describe("route stats", func() {

		Context("with some routes loaded", func() {
			var data map[string]map[string]interface{}

			BeforeEach(func() {
				addRoute("/foo", NewRedirectRoute("/bar", "prefix"))
				addRoute("/baz", NewRedirectRoute("/qux", "prefix"))
				addRoute("/foo", NewRedirectRoute("/bar/baz"))
				reloadRoutes()
				resp := doRequest(newRequest("GET", routerAPIURL("/stats")))
				Expect(resp.StatusCode).To(Equal(200))
				readJSONBody(resp, &data)
			})

			It("should return the number of routes loaded", func() {
				Expect(data["routes"]["count"]).To(BeEquivalentTo(3))
			})

			It("should return a checksum calculated from the sorted paths and route_types", func() {
				hash := sha1.New()
				hash.Write([]byte("/baz(true)"))
				hash.Write([]byte("/foo(false)"))
				hash.Write([]byte("/foo(true)"))
				Expect(data["routes"]["checksum"]).To(Equal(fmt.Sprintf("%x", hash.Sum(nil))))
			})
		})

		Context("with no routes", func() {
			var data map[string]map[string]interface{}

			BeforeEach(func() {
				reloadRoutes()

				resp := doRequest(newRequest("GET", routerAPIURL("/stats")))
				Expect(resp.StatusCode).To(Equal(200))
				readJSONBody(resp, &data)
			})

			It("should return the number of routes loaded", func() {
				Expect(data["routes"]["count"]).To(BeEquivalentTo(0))
			})

			It("should return a checksum of empty string", func() {
				hash := sha1.New()
				Expect(data["routes"]["checksum"]).To(Equal(fmt.Sprintf("%x", hash.Sum(nil))))
			})
		})

		It("should return 405 for other verbs", func() {
			resp := doRequest(newRequest("POST", routerAPIURL("/stats")))
			Expect(resp.StatusCode).To(Equal(405))
			Expect(resp.Header.Get("Allow")).To(Equal("GET"))
		})
	})

	Describe("memory stats", func() {
		It("should return memory statistics", func() {
			addRoute("/foo", NewRedirectRoute("/bar", "prefix"))
			addRoute("/baz", NewRedirectRoute("/qux", "prefix"))
			addRoute("/foo", NewRedirectRoute("/bar/baz"))
			reloadRoutes()

			resp := doRequest(newRequest("GET", routerAPIURL("/memory-stats")))
			Expect(resp.StatusCode).To(Equal(200))

			var data map[string]interface{}
			readJSONBody(resp, &data)

			Expect(data).To(HaveKey("Alloc"))
			Expect(data).To(HaveKey("HeapInuse"))
		})
	})
})
