package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("error handling", func() {

	Describe("handling an empty routing table", func() {
		BeforeEach(func() {
			reloadRoutes(apiPort)
		})

		It("should return a 503 error to the client", func() {
			resp := routerRequest(routerPort, "/")
			Expect(resp.StatusCode).To(Equal(503))

			resp = routerRequest(routerPort, "/foo")
			Expect(resp.StatusCode).To(Equal(503))
		})
	})
})
