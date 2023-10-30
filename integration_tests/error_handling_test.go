package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("error handling", func() {
	Describe("when no routes are loaded", func() {
		BeforeEach(func() {
			reloadRoutes(apiPort)
		})

		It("should forward to the default backend", func() {
			resp := routerRequest(routerPort, "/")
			Expect(resp).To(HaveHTTPHeaderWithValue("Server", "dummy-default-backend"))

			resp = routerRequest(routerPort, "/foo")
			Expect(resp).To(HaveHTTPStatus(404))
			Expect(resp).To(HaveHTTPHeaderWithValue("Server", "dummy-default-backend"))
		})
	})

	Describe("handling a panic", func() {
		BeforeEach(func() {
			addRoute("/boom", Route{Handler: "boom"})
			reloadRoutes(apiPort)
		})

		It("should return a 500 error to the client", func() {
			resp := routerRequest(routerPort, "/boom")
			Expect(resp).To(HaveHTTPStatus(500))
		})

		It("should log the fact", func() {
			routerRequest(routerPort, "/boom")

			logDetails := lastRouterErrorLogEntry()
			Expect(logDetails.Fields).To(Equal(map[string]interface{}{
				"error":          "panic: Boom!!!",
				"request":        "GET /boom HTTP/1.1",
				"request_method": "GET",
				"status":         float64(500), // All numbers in JSON are floating point
				"varnish_id":     "",
			}))
			Expect(logDetails.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})
	})
})
