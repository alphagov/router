package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/alphagov/router-postgres/handlers"
)

type redirectTableEntry struct {
	preserve  bool
	temporary bool
}

var _ = Describe("Redirect handlers", func() {
	entries := []TableEntry{
		Entry(
			"when redirects are temporary and paths are preserved",
			redirectTableEntry{preserve: true, temporary: true},
		),
		Entry(
			"when redirects are temporary and paths are not preserved",
			redirectTableEntry{preserve: false, temporary: true},
		),
		Entry(
			"when redirects are not temporary and paths are preserved",
			redirectTableEntry{preserve: true, temporary: false},
		),
		Entry(
			"when redirects are not temporary and paths are not preserved",
			redirectTableEntry{preserve: false, temporary: false},
		),
	}

	DescribeTable(
		"handlers",
		func(t redirectTableEntry) {
			rw := httptest.NewRecorder()

			handler := handlers.NewRedirectHandler(
				"/source-prefix", "/target-prefix",
				t.preserve, t.temporary,
			)

			var (
				redirectCode string
				redirectType string
			)

			if t.temporary {
				redirectCode = "302"
			} else {
				redirectCode = "301"
			}

			if t.preserve {
				redirectType = "path-preserving-redirect-handler"
			} else {
				redirectType = "redirect-handler"
			}

			labels := prometheus.Labels{
				"redirect_code": redirectCode,
				"redirect_type": redirectType,
			}

			beforeCount := promtest.ToFloat64(
				handlers.RedirectHandlerRedirectCountMetric.With(labels),
			)

			handler.ServeHTTP(
				rw,
				httptest.NewRequest(
					"GET",
					"https://source.gov.uk/source-prefix/path/subpath?query1=a&query2=b",
					nil,
				),
			)

			if t.temporary {
				// HTTP 302 is returned instead of 307
				// because we want the route to be cached temporarily
				// and not rerequested immediately
				Expect(rw.Result().StatusCode).To(
					Equal(http.StatusFound),
					"when the redirect is temporary we should return HTTP 302",
				)
			} else {
				Expect(rw.Result().StatusCode).To(
					Equal(http.StatusMovedPermanently),
					"when the redirect is permanent we should return HTTP 301",
				)
			}

			if t.preserve {
				Expect(rw.Result().Header.Get("Location")).To(
					Equal("/target-prefix/path/subpath?query1=a&query2=b"),
				)
			} else {
				Expect(rw.Result().Header.Get("Location")).To(
					Equal("/target-prefix"),
					"when we do not preserve the path, we redirect straight to target",
				)
			}

			Expect(rw.Result().Header.Get("Cache-Control")).To(
				SatisfyAll(
					ContainSubstring("public"),
					ContainSubstring("max-age=1800"),
				),
				"Declare public and cachable for 30 minutes",
			)

			Expect(rw.Result().Header.Get("Expires")).To(
				WithTransform(
					func(timestr string) time.Time {
						t, err := time.Parse(time.RFC1123, timestr)
						Expect(err).NotTo(HaveOccurred(), "Not RFC1123 compliant")
						return t
					},
					BeTemporally("~", time.Now().Add(30*time.Minute), 1*time.Second),
				),
				"Be RFC1123 compliant and expire around 30 minutes in the future",
			)

			afterCount := promtest.ToFloat64(
				handlers.RedirectHandlerRedirectCountMetric.With(labels),
			)

			Expect(afterCount-beforeCount).To(
				Equal(1.0),
				"Making a request should increment the redirect handler count metric",
			)
		},
		entries...,
	)

	Context("when we are not preserving paths", func() {
		var (
			rw      *httptest.ResponseRecorder
			handler http.Handler
		)

		BeforeEach(func() {
			rw = httptest.NewRecorder()

			handler = handlers.NewRedirectHandler(
				"/source-prefix", "/target-prefix",
				false, // preserve
				true,  // temporary
			)
		})

		Context("when the _ga query param is present", func() {
			It("should persist _ga to the query params", func() {
				handler.ServeHTTP(
					rw,
					httptest.NewRequest(
						"GET",
						"https://source.gov.uk/source-prefix?_ga=dontbeevil",
						nil,
					),
				)

				Expect(rw.Result().Header.Get("Location")).To(
					Equal("/target-prefix?_ga=dontbeevil"),
					"Preserve the _ga query parameter",
				)
			})
		})

		Context("when the _ga query param is not present", func() {
			It("should not add _ga to the query params", func() {
				handler.ServeHTTP(
					rw,
					httptest.NewRequest(
						"GET",
						"https://source.gov.uk/source-prefix?param=begood",
						nil,
					),
				)

				Expect(rw.Result().Header.Get("Location")).To(
					Equal("/target-prefix"),
					"Do not have any query params",
				)
			})
		})

		Context("metrics", func() {
			It("should increment the metric with redirect-handler label", func() {
				labels := prometheus.Labels{
					"redirect_code": "302",
					"redirect_type": "redirect-handler",
				}

				beforeCount := promtest.ToFloat64(
					handlers.RedirectHandlerRedirectCountMetric.With(labels),
				)

				handler.ServeHTTP(
					rw,
					httptest.NewRequest(
						"GET",
						"https://source.gov.uk/source-prefix",
						nil,
					),
				)

				Expect(rw.Result().Header.Get("Location")).To(
					Equal("/target-prefix"),
				)

				afterCount := promtest.ToFloat64(
					handlers.RedirectHandlerRedirectCountMetric.With(labels),
				)

				Expect(afterCount-beforeCount).To(
					Equal(1.0),
					"Making a request should increment the redirect handler count metric",
				)
			})
		})
	})
})
