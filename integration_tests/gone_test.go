package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gone routes", func() {

	BeforeEach(func() {
		addGoneRoute("/foo")
		addGoneRoute("/bar", "prefix")
		reloadRoutes()
	})

	It("should support an exact gone route", func() {
		resp := routerRequest("/foo")
		Expect(resp.StatusCode).To(Equal(410))

		resp = routerRequest("/foo/bar")
		Expect(resp.StatusCode).To(Equal(404))
	})

	It("should support a prefix gone route", func() {
		resp := routerRequest("/bar")
		Expect(resp.StatusCode).To(Equal(410))

		resp = routerRequest("/bar/baz")
		Expect(resp.StatusCode).To(Equal(410))
	})
})
