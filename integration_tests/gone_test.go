package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gone routes", func() {

	BeforeEach(func() {
		addRoute("/foo", NewGoneRoute())
		addRoute("/bar", NewGoneRoute("prefix"))
		reloadRoutes()
	})

	It("should support an exact gone route", func() {
		resp := routerRequest("/foo")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))

		resp = routerRequest("/foo/bar")
		Expect(resp.StatusCode).To(Equal(404))
		Expect(readBody(resp)).To(Equal("404 page not found\n"))
	})

	It("should support a prefix gone route", func() {
		resp := routerRequest("/bar")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))

		resp = routerRequest("/bar/baz")
		Expect(resp.StatusCode).To(Equal(410))
		Expect(readBody(resp)).To(Equal("410 Gone\n"))
	})
})
