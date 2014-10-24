package integration

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend selection", func() {

	It("should 404 with no routes", func() {
		reloadRoutes()
		resp := routerRequest("/")
		Expect(resp.StatusCode).To(Equal(404))

		resp = routerRequest("/foo")
		Expect(resp.StatusCode).To(Equal(404))
	})

	Describe("simple exact routes", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = startSimpleBackend("backend 1")
			backend2 = startSimpleBackend("backend 2")
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addBackendRoute("/foo", "backend-1")
			addBackendRoute("/bar", "backend-2")
			addBackendRoute("/baz", "backend-1")
			reloadRoutes()
		})
		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		It("should route a matching request to the corresponding backend", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/bar")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest("/baz")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})

		It("should 404 for children of the exact route", func() {
			resp := routerRequest("/foo/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should 404 for non-matching requests", func() {
			resp := routerRequest("/wibble")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/foo.json")
			Expect(resp.StatusCode).To(Equal(404))
		})
	})
})
