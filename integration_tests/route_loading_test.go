package integration

import (
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loading routes from the db", func() {
	var (
		backend1 *httptest.Server
		backend2 *httptest.Server
	)

	BeforeEach(func() {
		backend1 = startSimpleBackend("backend 1")
		backend2 = startSimpleBackend("backend 2")
		os.Setenv("BACKEND_URL_backend-1", backend1.URL)
		os.Setenv("BACKEND_URL_backend-2", backend2.URL)
	})
	AfterEach(func() {
		os.Unsetenv("BACKEND_URL_backend-1")
		os.Unsetenv("BACKEND_URL_backend-2")
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
