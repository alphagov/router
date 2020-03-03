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

	Context("when methods might be specified", func() {
		readOnlyMethods := []string{"GET", "HEAD"}
		writeOnlyMethods := []string{"POST", "PUT", "PATCH"}
		junkMethods := []string{"FOO", "PSOT", "BAR"}
		badMethods := []string{"TRACE", "DEBUG"}

		Context("routes without methods", func() {
			BeforeEach(func() {
				addRoute("/accept-all", NewBackendRoute("backend-1"))
				reloadRoutes()
			})

			It("should be able to accept the request", func() {
				for _, method := range readOnlyMethods {
					resp := routerRequestWithMethod(method, "/accept-all")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range writeOnlyMethods {
					resp := routerRequestWithMethod(method, "/accept-all")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range junkMethods {
					resp := routerRequestWithMethod(method, "/accept-all")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range badMethods {
					resp := routerRequestWithMethod(method, "/accept-all")
					Expect(resp.StatusCode).To(Equal(200))
				}
			})
		})

		Context("routes with methods", func() {
			BeforeEach(func() {
				addRoute("/read", NewBackendRouteWithMethods("backend-1", readOnlyMethods))
				addRoute("/readwrite", NewBackendRouteWithMethods("backend-2", append(readOnlyMethods, writeOnlyMethods...)))
				reloadRoutes()
			})

			It("can accept only read methods", func() {
				addRoute("/read", NewBackendRouteWithMethods("backend-1", readOnlyMethods))
				reloadRoutes()

				for _, method := range readOnlyMethods {
					resp := routerRequestWithMethod(method, "/read")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range writeOnlyMethods {
					resp := routerRequestWithMethod(method, "/read")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range junkMethods {
					resp := routerRequestWithMethod(method, "/read")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range badMethods {
					resp := routerRequestWithMethod(method, "/read")
					Expect(resp.StatusCode).To(Equal(405))
				}
			})

			It("can accept only write methods", func() {
				addRoute("/write", NewBackendRouteWithMethods("backend-1", writeOnlyMethods))
				reloadRoutes()

				for _, method := range readOnlyMethods {
					resp := routerRequestWithMethod(method, "/write")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range writeOnlyMethods {
					resp := routerRequestWithMethod(method, "/write")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range junkMethods {
					resp := routerRequestWithMethod(method, "/write")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range badMethods {
					resp := routerRequestWithMethod(method, "/write")
					Expect(resp.StatusCode).To(Equal(405))
				}
			})

			It("can accept only read and write methods", func() {
				addRoute("/readwrite", NewBackendRouteWithMethods("backend-1", append(readOnlyMethods, writeOnlyMethods...)))
				reloadRoutes()

				for _, method := range readOnlyMethods {
					resp := routerRequestWithMethod(method, "/readwrite")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range writeOnlyMethods {
					resp := routerRequestWithMethod(method, "/readwrite")
					Expect(resp.StatusCode).To(Equal(200))
				}

				for _, method := range junkMethods {
					resp := routerRequestWithMethod(method, "/readwrite")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range badMethods {
					resp := routerRequestWithMethod(method, "/readwrite")
					Expect(resp.StatusCode).To(Equal(405))
				}
			})
		})

		Context("when routes overlap", func() {
			BeforeEach(func() {
				addRoute("/overlap", NewBackendRouteWithMethods("backend-1", readOnlyMethods))
				addRoute("/overlap", NewBackendRouteWithMethods("backend-2", writeOnlyMethods))
				reloadRoutes()
			})

			It("should allow you to read from the read only backend", func() {
				for _, method := range readOnlyMethods {
					resp := routerRequestWithMethod(method, "/overlap")
					Expect(resp.StatusCode).To(Equal(200))
					Expect(readBody(resp)).To(Equal("backend 1"))
				}
			})

			It("should allow you to write to the write only backend", func() {
				for _, method := range writeOnlyMethods {
					resp := routerRequestWithMethod(method, "/overlap")
					Expect(resp.StatusCode).To(Equal(200))
					Expect(readBody(resp)).To(Equal("backend 2"))
				}
			})

			It("should reject other methods", func() {
				for _, method := range junkMethods {
					resp := routerRequestWithMethod(method, "/overlap")
					Expect(resp.StatusCode).To(Equal(405))
				}

				for _, method := range badMethods {
					resp := routerRequestWithMethod(method, "/overlap")
					Expect(resp.StatusCode).To(Equal(405))
				}
			})
		})
	})
})
