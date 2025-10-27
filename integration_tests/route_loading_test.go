package integration

import (
	"net/http/httptest"
	"time"

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

	Context("when content store listeners are disabled", func() {
		BeforeEach(func() {
			err := startRouterWithContentStoreListenersDisabled(3270, 3271)
			Expect(err).NotTo(HaveOccurred())

			addRoute("/known", NewBackendRoute("backend-1"))
			reloadRoutes(3271)
		})

		AfterEach(func() {
			stopRouter(3270)
		})

		It("does not load the new route when it is added to the database", func() {
			addRoute("/listener-disabled", NewBackendRoute("backend-1"))

			// Use a "Consistently" test to ensure we're not
			// seeing success because of a race condition
			Consistently(func() int {
				resp := routerRequest(3270, "/listener-disabled")
				return resp.StatusCode
			}).WithTimeout(time.Second * 5).
				WithPolling(time.Millisecond * 500).
				Should(Equal(404))
		})
	})
})

func startRouterWithContentStoreListenersDisabled(port, apiPort int) error {
	extraEnv := []string{
		"ROUTER_ENABLE_CONTENT_STORE_UPDATES=false",
		"BACKEND_URL_backend-1=http://" + backends["backend-1"],
		"ROUTER_ROUTE_RELOAD_INTERVAL=60m",
	}
	return startRouter(port, apiPort, extraEnv)
}
