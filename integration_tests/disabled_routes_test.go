package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("marking routes as disabled", func() {

	Describe("handling a disabled route", func() {
		BeforeEach(func() {
			addRoute("/unavailable", ContentItem{DocumentType: "gone", Routes: []Route{Route{Disabled: true}}})
			addRoute("/something-live", NewRedirectRoute("/somewhere-else"))
			reloadRoutes()
		})

		It("should return a 503 to the client", func() {
			resp := routerRequest("/unavailable")
			Expect(resp.StatusCode).To(Equal(503))
		})

		It("should continue to route other requests", func() {
			resp := routerRequest("/something-live")
			Expect(resp.StatusCode).To(Equal(301))
			Expect(resp.Header.Get("Location")).To(Equal("/somewhere-else"))
		})
	})
})
