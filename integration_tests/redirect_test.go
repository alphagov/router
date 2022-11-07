package integration

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redirection", func() {

	Describe("exact redirects", func() {
		BeforeEach(func() {
			addRoute("/foo", NewRedirectRoute("/bar"))
			addRoute("/foo-temp", NewRedirectRoute("/bar", "exact", "temporary"))
			addRoute("/query-temp", NewRedirectRoute("/bar?query=true", "exact"))
			addRoute("/fragment", NewRedirectRoute("/bar#section", "exact"))
			addRoute("/preserve-query", NewRedirectRoute("/qux", "exact", "permanent", "preserve"))
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

		It("should not preserve the query string for the source by default", func() {
			resp := routerRequest("/foo?baz=qux")
			Expect(resp.Header.Get("Location")).To(Equal("/bar"))
		})

		It("should preserve the query string for the source if specified", func() {
			resp := routerRequest("/preserve-query?foo=bar")
			Expect(resp.Header.Get("Location")).To(Equal("/qux?foo=bar"))
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
			addRoute("/foo", NewRedirectRoute("/bar", "prefix"))
			addRoute("/foo-temp", NewRedirectRoute("/bar-temp", "prefix", "temporary"))
			addRoute("/qux", NewRedirectRoute("/baz", "prefix", "temporary", "ignore"))
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

		It("should preserve extra path sections when redirecting by default", func() {
			resp := routerRequest("/foo/baz")
			Expect(resp.Header.Get("Location")).To(Equal("/bar/baz"))
		})

		It("should ignore extra path sections when redirecting if specified", func() {
			resp := routerRequest("/qux/quux")
			Expect(resp.Header.Get("Location")).To(Equal("/baz"))
		})

		It("should preserve the query string when redirecting by default", func() {
			resp := routerRequest("/foo?baz=qux")
			Expect(resp.Header.Get("Location")).To(Equal("/bar?baz=qux"))
		})

		It("should not preserve the query string when redirecting if specified", func() {
			resp := routerRequest("/qux/quux?foo=bar")
			Expect(resp.Header.Get("Location")).To(Equal("/baz"))
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

		It("should handle path-preserving redirects with special characters", func() {
			addRoute("/foo%20bar", NewRedirectRoute("/bar%20baz", "prefix"))
			reloadRoutes()

			resp := routerRequest("/foo bar/something")
			Expect(resp.StatusCode).To(Equal(301))
			Expect(resp.Header.Get("Location")).To(Equal("/bar%20baz/something"))
		})
	})

	Describe("external redirects", func() {
		BeforeEach(func() {
			addRoute("/foo", NewRedirectRoute("http://foo.example.com/foo"))
			addRoute("/baz", NewRedirectRoute("http://foo.example.com/baz", "exact", "permanent", "preserve"))
			addRoute("/bar", NewRedirectRoute("http://bar.example.com/bar", "prefix"))
			addRoute("/qux", NewRedirectRoute("http://bar.example.com/qux", "prefix", "permanent", "ignore"))
			reloadRoutes()
		})

		Describe("exact redirect", func() {
			It("should redirect to the external URL", func() {
				resp := routerRequest("/foo")
				Expect(resp.Header.Get("Location")).To(Equal("http://foo.example.com/foo"))
			})

			It("should not preserve the query string by default", func() {
				resp := routerRequest("/foo?foo=qux")
				Expect(resp.Header.Get("Location")).To(Equal("http://foo.example.com/foo"))
			})

			It("should preserve the query string if specified", func() {
				resp := routerRequest("/baz?foo=qux")
				Expect(resp.Header.Get("Location")).To(Equal("http://foo.example.com/baz?foo=qux"))
			})
		})

		Describe("prefix redirect", func() {
			It("should redirect to the external URL", func() {
				resp := routerRequest("/bar")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar"))
			})

			It("should preserve extra path sections when redirecting by default", func() {
				resp := routerRequest("/bar/baz")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar/baz"))
			})

			It("should ignore extra path sections when redirecting if specified", func() {
				resp := routerRequest("/qux/baz")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/qux"))
			})

			It("should preserve the query string when redirecting", func() {
				resp := routerRequest("/bar?baz=qux")
				Expect(resp.Header.Get("Location")).To(Equal("http://bar.example.com/bar?baz=qux"))
			})
		})
	})

	Describe("redirects with a _ga parameter", func() {
		BeforeEach(func() {
			addRoute("/foo", NewRedirectRoute("https://hmrc.service.gov.uk/pay", "prefix", "permanent", "ignore"))
			addRoute("/bar", NewRedirectRoute("https://bar.service.gov.uk/bar", "exact", "temporary", "preserve"))
			addRoute("/baz", NewRedirectRoute("https://gov.uk/baz-luhrmann", "exact", "permanent", "ignore"))
			addRoute("/pay-tax", NewRedirectRoute("https://tax.service.gov.uk/pay", "exact", "permanent", "ignore"))
			addRoute("/biz-bank", NewRedirectRoute("https://british-business-bank.co.uk", "prefix", "permanent", "ignore"))
			addRoute("/query-paramed", NewRedirectRoute("https://param.servicegov.uk?included-param=true", "exact", "permanent", "ignore"))
			reloadRoutes()
		})

		It("should only preserve the _ga parameter when redirecting to service URLs that want to ignore query params", func() {
			resp := routerRequest("/foo?_ga=identifier&blah=xyz")
			Expect(resp.Header.Get("Location")).To(Equal("https://hmrc.service.gov.uk/pay?_ga=identifier"))
		})

		It("should retain all params when redirecting to a route that wants them", func() {
			resp := routerRequest("/bar?wanted=param&_ga=xyz&blah=xyz")
			Expect(resp.Header.Get("Location")).To(Equal("https://bar.service.gov.uk/bar?wanted=param&_ga=xyz&blah=xyz"))
		})

		It("should preserve the _ga parameter when redirecting to gov.uk URLs", func() {
			resp := routerRequest("/baz?_ga=identifier")
			Expect(resp.Header.Get("Location")).To(Equal("https://gov.uk/baz-luhrmann?_ga=identifier"))
		})

		It("should preserve the _ga parameter when redirecting to service.gov.uk URLs", func() {
			resp := routerRequest("/pay-tax?_ga=12345")
			Expect(resp.Header.Get("Location")).To(Equal("https://tax.service.gov.uk/pay?_ga=12345"))
		})

		It("should preserve only the first _ga parameter", func() {
			resp := routerRequest("/pay-tax/?_ga=12345&_ga=6789")
			Expect(resp.Header.Get("Location")).To(Equal("https://tax.service.gov.uk/pay?_ga=12345"))
		})

		It("should preserve the _ga param when redirecting to british business bank", func() {
			resp := routerRequest("/biz-bank?unwanted=param&_ga=12345")
			Expect(resp.Header.Get("Location")).To(Equal("https://british-business-bank.co.uk?_ga=12345"))
		})

		It("should preserve the _ga param and any existing query string that the target URL has", func() {
			resp := routerRequest("/query-paramed?unwanted_param=blah&_ga=12345")
			// https://param.servicegov.uk?included-param=true?unwanted_param=blah&_ga=12345
			Expect(resp.Header.Get("Location")).To(Equal("https://param.servicegov.uk?_ga=12345&included-param=true"))
		})
	})
})
