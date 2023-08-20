package integration

import (
	"fmt"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loading routes from the db", func() {
	var (
		backend1 *httptest.Server
		backend2 *httptest.Server
		backend3 *httptest.Server
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

	Context("a backend an env var overriding the backend_url", func() {
		BeforeEach(func() {
			// This tests the behaviour of backend.ParseURL overriding the backend_url
			// provided in the DB with the value of an env var
			blackHole := "240.0.0.0/foo"
			backend3 = startSimpleBackend("backend 3")
			addBackend("backend-3", blackHole)

			stopRouter(routerPort)
			err := startRouter(routerPort, apiPort, []string{fmt.Sprintf("BACKEND_URL_backend-3=%s", backend3.URL)})
			Expect(err).NotTo(HaveOccurred())

			addRoute("/oof", NewBackendRoute("backend-3"))
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			stopRouter(routerPort)
			err := startRouter(routerPort, apiPort, nil)
			Expect(err).NotTo(HaveOccurred())
			backend3.Close()
		})

		It("should send requests to the backend_url provided in the env var", func() {
			resp := routerRequest(routerPort, "/oof")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(Equal("backend 3"))
		})
	})
})
