package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gone routes", func() {

	BeforeEach(func() {
		addRoute("/foo", NewGoneRoute())
		addRoute("/bar", NewGoneRoute("prefix"))
		reloadRoutes(apiPort)
	})

	It("should support an exact gone route", func() {
		resp := routerRequest(routerPort, "/foo")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))

		resp = routerRequest(routerPort, "/foo/bar")
		Expect(resp.StatusCode).To(Equal(404))
		Expect(readBody(resp)).To(Equal("404 page not found\n"))
	})

	It("should support a prefix gone route", func() {
		resp := routerRequest(routerPort, "/bar")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))

		resp = routerRequest(routerPort, "/bar/baz")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))
	})
})
