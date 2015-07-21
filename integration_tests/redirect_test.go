package integration

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redirection", func() {

	Describe("exact redirects", func() {
		BeforeEach(func() {
			addRedirectRoute("/foo", "/bar")
			addRedirectRoute("/foo-temp", "/bar", "exact", "temporary")
			addRedirectRoute("/query-temp", "/bar?query=true", "exact")
			addRedirectRoute("/fragment", "/bar#section", "exact")
			reloadRoutes()
		})

		It("should redirect permanently by default", func() {
			resp := routerRequest("/foo")
			Expect(resp.StatusCode).To(Equal(301))
		})

		It("should redirect temporarily when asked to", func() {
			resp := routerRequest("/foo-temp")
			Expect(resp.StatusCode).To(Equal(302))
		})

		It("should contain the redirect location", func() {
			resp := routerRequest("/foo")
			Expect(resp.Header.Get("Location")).To(Equal("/bar"))
		})

		It("should not preserve the query string for the source", func() {
			resp := routerRequest("/foo?baz=qux")
			Expect(resp.Header.Get("Location")).To(Equal("/bar"))
		})

		It("should preserve the query string for the target", func() {
			resp := routerRequest("/query-temp")
			Expect(resp.Header.Get("Location")).To(Equal("/bar?query=true"))
		})

		It("should preserve the fragment for the target", func() {
			resp := routerRequest("/fragment")
			Expect(resp.Header.Get("Location")).To(Equal("/bar#section"))
		})

		It("should contain cache headers of 30 mins", func() {
			resp := routerRequest("/foo")
			Expect(resp.Header.Get("Cache-Control")).To(Equal("max-age=1800, public"))

			Expect(
				time.Parse(time.RFC1123, resp.Header.Get("Expires")),
			).To(BeTemporally(
				"~",
				time.Now().Add(30*time.Minute),
				time.Second,
			))
		})
	})

	Describe("prefix redirects", func() {
		BeforeEach(func() {
			addRedirectRoute("/foo", "/bar", "prefix")
			addRedirectRoute("/foo-temp", "/bar-temp", "prefix", "temporary")
			reloadRoutes()
		})

		It("should redirect permanently to the destination", func() {
			resp := routerRequest("/foo")
			Expect(resp.StatusCode).To(Equal(301))
			Expect(resp.Header.Get("Location")).To(Equal("/bar"))
		})

		It("should redirect temporarily to the destination when asked to", func() {
			resp := routerRequest("/foo-temp")
			Expect(resp.StatusCode).To(Equal(302))
			Expect(resp.Header.Get("Location")).To(Equal("/bar-temp"))
		})

		It("should preserve extra path sections when redirecting", func() {
			resp := routerRequest("/foo/baz")
			Expect(resp.Header.Get("Location")).To(Equal("/bar/baz"))
		})

		It("should preserve the query string when redirecting", func() {
			resp := routerRequest("/foo?baz=qux")
			Expect(resp.Header.Get("Location")).To(Equal("/bar?baz=qux"))
		})

		It("should contain cache headers of 30 mins", func() {
			resp := routerRequest("/foo")
			Expect(resp.Header.Get("Cache-Control")).To(Equal("max-age=1800, public"))

			Expect(
				time.Parse(time.RFC1123, resp.Header.Get("Expires")),
			).To(BeTemporally(
				"~",
				time.Now().Add(30*time.Minute),
				time.Second,
			))
		})
	})

	Describe("external redirects", func() {
		BeforeEach(func() {
			addRedirectRoute("/foo", "http://foo.example.com/foo")
			addRedirectRoute("/bar", "http://bar.example.com/bar", "prefix")
			reloadRoutes()
		})

		Describe("exact redirect", func() {
			It("should redirect to the external URL", func() {
				resp := routerRequest("/foo")
				Expect(resp.Header.Get("Location")).To(Equal("http://foo.example.com/foo"))
			})

			It("should not preserve the query string", func() {
				resp := routerRequest("/foo?baz=qux")
				Expect(resp.Header.Get("Location")).To(Equal("http://foo.example.com/foo"))
			})
		})

		Describe("prefix redirect", func() {
			It("should redirect to the external URL", func() {
				resp := routerRequest("/bar")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar"))
			})

			It("should preserve extra path sections when redirecting", func() {
				resp := routerRequest("/bar/baz")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar/baz"))
			})

			It("should preserve the query string when redirecting", func() {
				resp := routerRequest("/bar?baz=qux")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar?baz=qux"))
			})
		})
	})
})
