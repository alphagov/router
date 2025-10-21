package integration

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flat File Route Loading", func() {
	var (
		testRoutesFile string
		backend1       *httptest.Server
	)

	BeforeEach(func() {
		// Get absolute path to test routes file
		testRoutesFile, _ = filepath.Abs("testdata/sample_routes.jsonl")
		backend1 = startSimpleBackend("backend-1", backends["backend-1"])
	})

	AfterEach(func() {
		if backend1 != nil {
			backend1.Close()
		}
	})

	Describe("loading routes from JSONL file", func() {
		BeforeEach(func() {
			err := startRouterWithFile(3170, 3171, testRoutesFile)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			stopRouter(3170)
		})

		It("should route backend requests correctly", func() {
			resp := routerRequest(3170, "/")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(readBody(resp)).To(Equal("backend-1"))
		})

		It("should handle prefix routes", func() {
			resp := routerRequest(3170, "/foo/bar/baz")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(readBody(resp)).To(Equal("backend-1"))
		})

		It("should handle exact redirects", func() {
			resp := routerRequest(3170, "/old-path")
			Expect(resp.StatusCode).To(Equal(http.StatusMovedPermanently))
			Expect(resp.Header.Get("Location")).To(Equal("/new-path"))
		})

		It("should handle prefix redirects with segment preservation", func() {
			resp := routerRequest(3170, "/moved/extra/path")
			Expect(resp.StatusCode).To(Equal(http.StatusMovedPermanently))
			Expect(resp.Header.Get("Location")).To(Equal("/destination/extra/path"))
		})

		It("should handle gone routes", func() {
			resp := routerRequest(3170, "/gone")
			Expect(resp.StatusCode).To(Equal(http.StatusGone))
		})

		It("should return 404 for unknown routes", func() {
			resp := routerRequest(3170, "/unknown")
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})

func startRouterWithFile(port, apiPort int, routesFile string) error {
	extraEnv := []string{
		"ROUTER_ROUTES_FILE=" + routesFile,
		"BACKEND_URL_backend-1=http://" + backends["backend-1"],
	}
	return startRouter(port, apiPort, extraEnv)
}
