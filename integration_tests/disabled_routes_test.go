package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("marking routes as disabled", func() {

	Describe("handling a disabled route", func() {
		BeforeEach(func() {
			addRoute("/unavailable", Route{Handler: "gone", Disabled: true})
			addRoute("/something-live", NewRedirectRoute("/somewhere-else"))
			reloadRoutes(apiPort)
		})

		It("should return a 503 to the client", func() {
			resp := routerRequest(routerPort, "/unavailable")
			Expect(resp).To(HaveHTTPStatus(503))
		})

		It("should continue to route other requests", func() {
			resp := routerRequest(routerPort, "/something-live")
			Expect(resp).To(HaveHTTPStatus(301))
			Expect(resp.Header.Get("Location")).To(Equal("/somewhere-else"))
		})
	})
})
