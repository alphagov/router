package integration

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loading routes from the db", func() {
	var (
		backend1 *httptest.Server
		backend2 *httptest.Server
	)

	BeforeEach(func() {
		backend1 = startSimpleBackend("backend 1", backends["backend-1"])
		backend2 = startSimpleBackend("backend 2", backends["backend-2"])
	})
	AfterEach(func() {
		backend1.Close()
		backend2.Close()
	})

	Context("a route with an unrecognised handler type", func() {
		BeforeEach(func() {
			addRoute("/foo", NewBackendRoute("backend-1"))
			addRoute("/bar", Route{Handler: "fooey"})
			addRoute("/baz", NewBackendRoute("backend-2"))
			reloadRoutes(apiPort)
		})

		It("should skip the invalid route", func() {
			resp := routerRequest(routerPort, "/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should continue to load other routes", func() {
			resp := routerRequest(routerPort, "/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest(routerPort, "/baz")
			Expect(readBody(resp)).To(Equal("backend 2"))
		})
	})

	Context("a route with a non-existent backend", func() {
		BeforeEach(func() {
			addRoute("/foo", NewBackendRoute("backend-1"))
			addRoute("/bar", NewBackendRoute("backend-non-existent"))
			addRoute("/baz", NewBackendRoute("backend-2"))
			addRoute("/qux", NewBackendRoute("backend-1"))
			reloadRoutes(apiPort)
		})

		It("should skip the invalid route", func() {
			resp := routerRequest(routerPort, "/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should continue to load other routes", func() {
			resp := routerRequest(routerPort, "/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest(routerPort, "/baz")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest(routerPort, "/qux")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})
	})
})
