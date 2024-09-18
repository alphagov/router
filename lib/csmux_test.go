package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("ContentStoreMux", func() {
	var (
		server *ghttp.Server
		mux    *ContentStoreMux
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		os.Setenv("CONTENT_STORE_BEARER_TOKEN", "test-token")
		os.Setenv("BACKEND_URL_content-store", server.URL())
		var err error
		mux, err = NewContentStoreMux()
		Expect(err).NotTo(HaveOccurred())
		Expect(mux).NotTo(BeNil())
	})

	AfterEach(func() {
		server.Close()
		os.Unsetenv("CONTENT_STORE_BEARER_TOKEN")
		os.Unsetenv("BACKEND_URL_content-store")
	})

	Describe("NewContentStoreMux", func() {
		It("should create a new ContentStoreMux with valid environment variables", func() {
			Expect(mux.BearerToken).To(Equal("Bearer test-token"))
			Expect(mux.ContentStoreURL).To(Equal(server.URL()))
		})

		It("should return an error if environment variables are missing", func() {
			os.Unsetenv("CONTENT_STORE_BEARER_TOKEN")
			os.Unsetenv("BACKEND_URL_content-store")

			mux, err := NewContentStoreMux()
			Expect(err).To(HaveOccurred())
			Expect(mux).To(BeNil())
		})
	})

	Describe("ServeHTTP", func() {
		var (
			recorder *httptest.ResponseRecorder
			request  *http.Request
			backends map[string]http.Handler
		)

		BeforeEach(func() {
			recorder = httptest.NewRecorder()
			backends = map[string]http.Handler{
				"backend1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("backend1 response"))
				}),
			}
		})

		Context("when the route is a redirect", func() {
			BeforeEach(func() {
				route := &CSRoute{
					Path:         StringPtr("/redirect"),
					MatchType:    StringPtr("exact"),
					Backend:      StringPtr("redirect"),
					Destination:  StringPtr("http://example.com"),
					SegmentsMode: StringPtr("preserve"),
				}
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/redirect"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, route),
					),
				)
				request = httptest.NewRequest("GET", "/redirect", nil)
			})

			It("should return 301 Moved Permanently with the correct Location header", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusMovedPermanently))
				Expect(recorder.Header().Get("Location")).To(Equal("http://example.com"))
			})
		})

		Context("when the route is gone", func() {
			BeforeEach(func() {
				route := &CSRoute{
					Path:      StringPtr("/gone"),
					MatchType: StringPtr("exact"),
					Backend:   StringPtr("gone"),
				}
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/gone"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, route),
					),
				)
				request = httptest.NewRequest("GET", "/gone", nil)
			})

			It("should return 410 Gone", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusGone))
			})
		})
		Context("when the route is found", func() {
			BeforeEach(func() {
				route := &CSRoute{
					Path:        StringPtr("/test"),
					MatchType:   StringPtr("exact"),
					Backend:     StringPtr("backend1"),
					Destination: nil,
				}
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/test"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, route),
					),
				)
				request = httptest.NewRequest("GET", "/test", nil)
			})

			It("should route to the correct backend", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("backend1 response"))
			})
		})

		Context("when the route is found but backend handler isn't present", func() {
			BeforeEach(func() {
				route := &CSRoute{
					Path:        StringPtr("/missing-backend"),
					MatchType:   StringPtr("exact"),
					Backend:     StringPtr("missing-backend"),
					Destination: nil,
				}
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/missing-backend"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, route),
					),
				)
				request = httptest.NewRequest("GET", "/missing-backend", nil)
			})

			It("should return 500 Internal Server Error", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when the route is not found", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/notfound"),
						ghttp.RespondWith(http.StatusNotFound, nil),
					),
				)
				request = httptest.NewRequest("GET", "/notfound", nil)
			})

			It("should return 404 Not Found", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})

		Context("when the content store returns an error", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routes", "path=/error"),
						ghttp.RespondWith(http.StatusInternalServerError, nil),
					),
				)
				request = httptest.NewRequest("GET", "/error", nil)
			})

			It("should return 500 Internal Server Error", func() {
				mux.ServeHTTP(recorder, request, backends)
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})

func StringPtr(s string) *string {
	return &s
}

func TestContentStoreMux(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ContentStoreMux Suite")
}
