package integration

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
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
		addBackend("backend-1", backend1.URL)
		addBackend("backend-2", backend2.URL)
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
			reloadRoutes()
		})

		It("should skip the invalid route", func() {
			resp := routerRequest("/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should continue to load other routes", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/baz")
			Expect(readBody(resp)).To(Equal("backend 2"))
		})
	})

	Context("a route with a non-existent backend", func() {
		BeforeEach(func() {
			addRoute("/foo", NewBackendRoute("backend-1"))
			addRoute("/bar", NewBackendRoute("backend-non-existent"))
			addRoute("/baz", NewBackendRoute("backend-2"))
			addRoute("/qux", NewBackendRoute("backend-1"))
			reloadRoutes()
		})

		It("should skip the invalid route", func() {
			resp := routerRequest("/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should continue to load other routes", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/baz")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest("/qux")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})
	})

	Context("a route with methods", func() {
		ReadOnlyMethods := []string{"GET", "HEAD"}
		WriteOnlyMethods := []string{"POST", "PUT", "PATCH"}
		ReadWriteMethods := []string{"GET", "HEAD", "POST", "PUT", "PATCH"}

		BeforeEach(func() {
			addRoute("/accept-all", NewBackendRoute("backend-1"))
			addRoute("/read", NewBackendRouteWithMethods("backend-1", ReadOnlyMethods))
			addRoute("/write", NewBackendRouteWithMethods("backend-1", WriteOnlyMethods))
			addRoute("/readwrite", NewBackendRouteWithMethods("backend-2", ReadWriteMethods))
			reloadRoutes()
		})

		It("should accept all methods when methods are not unset", func() {
      for _, method := range ReadWriteMethods {
				resp := routerRequestWithMethod(method, "/accept-all")
				Expect(resp.StatusCode).To(Equal(200))
			}

			resp := routerRequestWithMethod("JUNK_METHOD", "/accept-all")
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should accept only specified methods when methods are set", func() {
			resp := routerRequestWithMethod("POST", "/read")
			Expect(resp.StatusCode).To(Equal(405))

      for _, method := range ReadOnlyMethods {
				resp := routerRequestWithMethod(method, "/read")
				Expect(resp.StatusCode).To(Equal(200))

				resp = routerRequestWithMethod(method, "/readwrite")
				Expect(resp.StatusCode).To(Equal(200))

				resp = routerRequestWithMethod(method, "/write")
				Expect(resp.StatusCode).To(Equal(405))
			}

			for _, method := range WriteOnlyMethods {
				resp := routerRequestWithMethod(method, "/write")
				Expect(resp.StatusCode).To(Equal(200))

				resp = routerRequestWithMethod(method, "/readwrite")
				Expect(resp.StatusCode).To(Equal(200))

				resp = routerRequestWithMethod(method, "/read")
				Expect(resp.StatusCode).To(Equal(405))
			}

			resp = routerRequestWithMethod("FOO_METHOD", "/readwrite")
			Expect(resp.StatusCode).To(Equal(405))

			resp = routerRequestWithMethod("JUNK_METHOD", "/read")
			Expect(resp.StatusCode).To(Equal(405))
		})
	})
})
