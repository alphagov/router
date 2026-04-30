package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Probe routes", func() {
	Context("with no other routes loaded", func() {
		var probeBackend *httptest.Server

		BeforeEach(func() {
			clearRoutes()
			probeBackend = startSimpleBackend("probe backend test response", backends["router-probe-backend"])
			reloadRoutes(apiPort)
		})

		AfterEach(func() {
			probeBackend.Close()
		})

		for _, probeBackendPath := range []string{
			"/__probe__/gone",
			"/__probe__/router-redirect",
			"/__probe__/ok",
			"/__probe__/redirect",
			"/__probe__/redirected",
			"/__probe__/not-found",
			"/__probe__/internal-server-error",
			"/__probe__/get",
			"/__probe__/post",
			"/__probe__/headers/get",
			"/__probe__/headers/post",
			"/__probe__/__canary__",
		} {
			It(fmt.Sprintf("should respond 503 to requests for %s", probeBackendPath), func() {
				resp := routerRequest(routerPort, probeBackendPath)
				Expect(resp.StatusCode).To(Equal(503))
			})
		}
	})

	Context("with other non-probe routes loaded", func() {
		BeforeEach(func() {
			addRoute("/foo", NewGoneRoute())
			reloadRoutes(apiPort)
		})
		It("should respond to the /__probe__/gone route", func() {
			resp := routerRequest(routerPort, "/__probe__/gone")
			Expect(resp.StatusCode).To(Equal(http.StatusGone))
			Expect(readBody(resp)).To(Equal("410 Gone\n"))
		})

		It("should redirect from /__probe__/router-redirect route", func() {
			resp := routerRequest(routerPort, "/__probe__/router-redirect")
			Expect(resp.StatusCode).To(Equal(http.StatusMovedPermanently))
			Expect(resp.Header.Get("Location")).To(Equal("/__probe__/redirected"))
			Expect(readBody(resp)).To(Equal("<a href=\"/__probe__/redirected\">Moved Permanently</a>.\n\n"))
		})

		Context("with a probe backend", func() {
			var probeBackend *httptest.Server

			BeforeEach(func() {
				probeBackend = startSimpleBackend("probe backend test response", backends["router-probe-backend"])
			})

			AfterEach(func() {
				probeBackend.Close()
			})

			for _, probeBackendPath := range []string{
				"/__probe__/ok",
				"/__probe__/redirect",
				"/__probe__/redirected",
				"/__probe__/not-found",
				"/__probe__/internal-server-error",
				"/__probe__/get",
				"/__probe__/post",
				"/__probe__/headers/get",
				"/__probe__/headers/post",
				"/__probe__/__canary__",
			} {
				It(fmt.Sprintf("should forward requests for %s to the probe backend", probeBackendPath), func() {
					resp := routerRequest(routerPort, probeBackendPath)
					Expect(resp.StatusCode).To(Equal(200))
					Expect(readBody(resp)).To(Equal("probe backend test response"))
				})
			}
		})
	})

})
