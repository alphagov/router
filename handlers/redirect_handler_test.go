package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
)

var _ = Describe("A redirect handler", func() {
	var handler http.Handler
	var rr *httptest.ResponseRecorder
	const url = "https://source.example.com/source/path/subpath?q1=a&q2=b"

	BeforeEach(func() {
		rr = httptest.NewRecorder()
	})

	// These behaviours apply to all combinations of both NewRedirectHandler flags.
	for _, preserve := range []bool{true, false} {
		Context(fmt.Sprintf("where preserve=%t", preserve), func() {
			BeforeEach(func() {
				handler = NewRedirectHandler("/source", "/target", preserve)
				handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
			})

			It("allows its response to be cached publicly for 30m", func() {
				Expect(rr.Result().Header.Get("Cache-Control")).To(
					SatisfyAll(ContainSubstring("public"), ContainSubstring("max-age=1800")))
			})

			It("returns an expires header with an RFC1123 datetime 30m in the future", func() {
				Expect(rr.Result().Header.Get("Expires")).To(WithTransform(
					func(s string) time.Time {
						t, err := time.Parse(time.RFC1123, s)
						Expect(err).NotTo(HaveOccurred())
						return t
					},
					BeTemporally("~", time.Now().Add(30*time.Minute), time.Minute)))
			})
		})
	}

	Context("where preserve=true", func() {
		BeforeEach(func() {
			handler = NewRedirectHandler("/source", "/target", true)
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
		})

		It("returns the original path in the location header", func() {
			Expect(rr.Result().Header.Get("Location")).To(Equal("/target/path/subpath?q1=a&q2=b"))
		})
	})

	Context("where preserve=false", func() {
		BeforeEach(func() {
			handler = NewRedirectHandler("/source", "/target", false)
		})

		It("returns only the configured path in the location header", func() {
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
			Expect(rr.Result().Header.Get("Location")).To(Equal("/target"))
		})

		It("still preserves the _ga query parameter as a special case", func() {
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
				"https://source.example.com/source?_ga=dontbeevil", nil))
			Expect(rr.Result().Header.Get("Location")).To(Equal("/target?_ga=dontbeevil"))
		})
	})

	DescribeTable("responds with the right HTTP status",
		EntryDescription("preserve=%t -> HTTP %d"),
		Entry(nil, false, http.StatusMovedPermanently),
		Entry(nil, true, http.StatusMovedPermanently),
		func(preserve bool, expectedStatus int) {
			handler = NewRedirectHandler("/source", "/target", preserve)
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))
			Expect(rr.Result().StatusCode).To(Equal(expectedStatus))
		})

	DescribeTable("increments the redirect-count metric with the right labels",
		EntryDescription("preserve=%t -> {redirect_type=%s}"),
		Entry(nil, false, "redirect-handler"),
		Entry(nil, false, "redirect-handler"),
		Entry(nil, true, "path-preserving-redirect-handler"),
		Entry(nil, true, "path-preserving-redirect-handler"),
		func(preserve bool, typeLabel string) {
			lbls := prometheus.Labels{"redirect_type": typeLabel}
			before := promtest.ToFloat64(redirectCountMetric.With(lbls))

			handler = NewRedirectHandler("/source", "/target", preserve)
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, url, nil))

			after := promtest.ToFloat64(redirectCountMetric.With(lbls))
			Expect(after - before).To(BeNumerically("~", 1.0))
		},
	)
})
