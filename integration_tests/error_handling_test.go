package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("error handling", func() {

	Describe("handling an empty routing table", func() {
		BeforeEach(func() {
			reloadRoutes()
		})

		It("should return a 503 error to the client", func() {
			resp := routerRequest("/")
			Expect(resp.StatusCode).To(Equal(503))

			resp = routerRequest("/foo")
			Expect(resp.StatusCode).To(Equal(503))
		})
	})

	Describe("handling a panic", func() {
		BeforeEach(func() {
			addRoute("/boom", Route{Handler: "boom"})
			reloadRoutes()
		})

		It("should return a 500 error to the client", func() {
			resp := routerRequest("/boom")
			Expect(resp.StatusCode).To(Equal(500))
		})

		It("should log the fact", func() {
			routerRequest("/boom")

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
