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
		Expect(resp).To(HaveHTTPStatus(410))

		resp = routerRequest(routerPort, "/foo/no-match")
		Expect(resp).To(HaveHTTPStatus(404))
	})

	It("should support a prefix gone route", func() {
		resp := routerRequest(routerPort, "/bar")
		Expect(resp).To(HaveHTTPStatus(410))

		resp = routerRequest(routerPort, "/bar/baz")
		Expect(resp).To(HaveHTTPStatus(410))
	})
})
