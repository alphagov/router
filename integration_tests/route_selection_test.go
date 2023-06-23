package integration

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Route selection", func() {

	Describe("simple exact routes", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = startSimpleBackend("backend 1")
			backend2 = startSimpleBackend("backend 2")
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addRoute("/foo", NewBackendRoute("backend-1"))
			addRoute("/bar", NewBackendRoute("backend-2"))
			addRoute("/baz", NewBackendRoute("backend-1"))
			reloadRoutes()
		})
		AfterEach(func() {
			clearRoutes()
			backend1.Close()
			backend2.Close()
		})

		It("should route a matching request to the corresponding backend", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/bar")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest("/baz")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})

		It("should 404 for children of the exact route", func() {
			resp := routerRequest("/foo/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should 404 for non-matching requests", func() {
			resp := routerRequest("/wibble")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/foo.json")
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("simple prefix routes", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = startSimpleBackend("backend 1")
			backend2 = startSimpleBackend("backend 2")
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addRoute("/foo", NewBackendRoute("backend-1", "prefix"))
			addRoute("/bar", NewBackendRoute("backend-2", "prefix"))
			addRoute("/baz", NewBackendRoute("backend-1", "prefix"))
			reloadRoutes()
		})
		AfterEach(func() {
			clearRoutes()
			backend1.Close()
			backend2.Close()
		})

		It("should route requests for the prefix to the backend", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/bar")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest("/baz")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})

		It("should route requests for the children of the prefix to the backend", func() {
			resp := routerRequest("/foo/bar")
			Expect(readBody(resp)).To(Equal("backend 1"))

			resp = routerRequest("/bar/foo.json")
			Expect(readBody(resp)).To(Equal("backend 2"))

			resp = routerRequest("/baz/fooey/kablooie")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})

		It("should 404 for non-matching requests", func() {
			resp := routerRequest("/wibble")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/")
			Expect(resp.StatusCode).To(Equal(404))

			resp = routerRequest("/foo.json")
			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Describe("prefix route with children", func() {
		var (
			outer *httptest.Server
			inner *httptest.Server
		)

		BeforeEach(func() {
			outer = startSimpleBackend("outer")
			inner = startSimpleBackend("inner")
			addBackend("outer-backend", outer.URL)
			addBackend("inner-backend", inner.URL)
			addRoute("/foo", NewBackendRoute("outer-backend", "prefix"))
			reloadRoutes()
		})
		AfterEach(func() {
			clearRoutes()
			outer.Close()
			inner.Close()
		})

		Describe("with an exact child", func() {
			BeforeEach(func() {
				addRoute("/foo/bar", NewBackendRoute("inner-backend"))
				reloadRoutes()
			})

			It("should route the prefix to the outer backend", func() {
				resp := routerRequest("/foo")
				Expect(readBody(resp)).To(Equal("outer"))
			})

			It("should route the exact child to the inner backend", func() {
				resp := routerRequest("/foo/bar")
				Expect(readBody(resp)).To(Equal("inner"))
			})

			It("should route the children of the exact child to the outer backend", func() {
				resp := routerRequest("/foo/bar/baz")
				Expect(readBody(resp)).To(Equal("outer"))
			})
		})

		Describe("with a prefix child", func() {
			BeforeEach(func() {
				addRoute("/foo/bar", NewBackendRoute("inner-backend", "prefix"))
				reloadRoutes()
			})

			It("should route the outer prefix to the outer backend", func() {
				resp := routerRequest("/foo")
				Expect(readBody(resp)).To(Equal("outer"))
			})

			It("should route the inner prefix to the inner backend", func() {
				resp := routerRequest("/foo/bar")
				Expect(readBody(resp)).To(Equal("inner"))
			})

			It("should route the children of the inner prefix to the inner backend", func() {
				resp := routerRequest("/foo/bar/baz")
				Expect(readBody(resp)).To(Equal("inner"))
			})

			It("should route other children of the outer prefix to the outer backend", func() {
				resp := routerRequest("/foo/baz")
				Expect(readBody(resp)).To(Equal("outer"))

				resp = routerRequest("/foo/bar.json")
				Expect(readBody(resp)).To(Equal("outer"))
			})
		})

		Describe("with an exact child and a deeper prefix child", func() {
			var (
				innerer *httptest.Server
			)
			BeforeEach(func() {
				innerer = startSimpleBackend("innerer")
				addBackend("innerer-backend", innerer.URL)
				addRoute("/foo/bar", NewBackendRoute("inner-backend"))
				addRoute("/foo/bar/baz", NewBackendRoute("innerer-backend", "prefix"))
				reloadRoutes()
			})
			AfterEach(func() {
				clearRoutes()
				innerer.Close()
			})

			It("should route the outer prefix to the outer backend", func() {
				resp := routerRequest("/foo")
				Expect(readBody(resp)).To(Equal("outer"))

				resp = routerRequest("/foo/baz")
				Expect(readBody(resp)).To(Equal("outer"))

				resp = routerRequest("/foo/bar.json")
				Expect(readBody(resp)).To(Equal("outer"))
			})

			It("should route the exact route to the inner backend", func() {
				resp := routerRequest("/foo/bar")
				Expect(readBody(resp)).To(Equal("inner"))
			})

			It("should route other children of the exact route to the outer backend", func() {
				resp := routerRequest("/foo/bar/wibble")
				Expect(readBody(resp)).To(Equal("outer"))

				resp = routerRequest("/foo/bar/baz.json")
				Expect(readBody(resp)).To(Equal("outer"))
			})

			It("should route the inner prefix route to the innerer backend", func() {
				resp := routerRequest("/foo/bar/baz")
				Expect(readBody(resp)).To(Equal("innerer"))
			})

			It("should route children of the inner prefix route to the innerer backend", func() {
				resp := routerRequest("/foo/bar/baz/wibble")
				Expect(readBody(resp)).To(Equal("innerer"))
			})
		})
	})

	Describe("prefix and exact route at the same level", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = startSimpleBackend("backend 1")
			backend2 = startSimpleBackend("backend 2")
			addBackend("backend-1", backend1.URL)
			addBackend("backend-2", backend2.URL)
			addRoute("/foo", NewBackendRoute("backend-1", "prefix"))
			addRoute("/foo", NewBackendRoute("backend-2"))
			reloadRoutes()
		})
		AfterEach(func() {
			clearRoutes()
			backend1.Close()
			backend2.Close()
		})

		It("should route the exact route to the exact backend", func() {
			resp := routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("backend 2"))
		})

		It("should route children of the route to the prefix backend", func() {
			resp := routerRequest("/foo/bar")
			Expect(readBody(resp)).To(Equal("backend 1"))
		})
	})

	Describe("routes at the root level", func() {
		var (
			root  *httptest.Server
			other *httptest.Server
		)

		BeforeEach(func() {
			root = startSimpleBackend("root backend")
			other = startSimpleBackend("other backend")
			addBackend("root", root.URL)
			addBackend("other", other.URL)
			addRoute("/foo", NewBackendRoute("other"))
		})
		AfterEach(func() {
			clearRoutes()
			root.Close()
			other.Close()
		})

		It("should handle an exact route at the root level", func() {
			addRoute("/", NewBackendRoute("root"))
			reloadRoutes()

			resp := routerRequest("/")
			Expect(readBody(resp)).To(Equal("root backend"))

			resp = routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("other backend"))

			resp = routerRequest("/bar")
			Expect(resp.StatusCode).To(Equal(404))
		})

		It("should handle a prefix route at the root level", func() {
			addRoute("/", NewBackendRoute("root", "prefix"))
			reloadRoutes()

			resp := routerRequest("/")
			Expect(readBody(resp)).To(Equal("root backend"))

			resp = routerRequest("/foo")
			Expect(readBody(resp)).To(Equal("other backend"))

			resp = routerRequest("/bar")
			Expect(readBody(resp)).To(Equal("root backend"))
		})
	})

	Describe("double slashes", func() {
		var (
			root     *httptest.Server
			recorder *ghttp.Server
		)

		BeforeEach(func() {
			root = startSimpleBackend("fallthrough")
			recorder = startRecordingBackend()
			addBackend("root", root.URL)
			addBackend("other", recorder.URL())
			addRoute("/", NewBackendRoute("root", "prefix"))
			addRoute("/foo/bar", NewBackendRoute("other", "prefix"))
			reloadRoutes()
		})
		AfterEach(func() {
			clearRoutes()
			root.Close()
			recorder.Close()
		})

		It("should not be redirected by our simple test backend", func() {
			resp := routerRequest("//")
			Expect(readBody(resp)).To(Equal("fallthrough"))
		})

		It("should not be redirected by our recorder backend", func() {
			resp := routerRequest("/foo/bar/baz//qux")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			Expect(recorder.ReceivedRequests()[0].URL.Path).To(Equal("/foo/bar/baz//qux"))
		})

		It("should collapse double slashes when looking up route, but pass request as-is", func() {
			resp := routerRequest("/foo//bar")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			Expect(recorder.ReceivedRequests()[0].URL.Path).To(Equal("/foo//bar"))
		})
	})

	Describe("special characters in paths", func() {
		var recorder *ghttp.Server

		BeforeEach(func() {
			recorder = startRecordingBackend()
			addBackend("backend", recorder.URL())
		})
		AfterEach(func() {
			clearRoutes()
			recorder.Close()
		})

		It("should handle spaces (%20) in paths", func() {
			addRoute("/foo%20bar", NewBackendRoute("backend"))
			reloadRoutes()

			resp := routerRequest("/foo bar")
			Expect(resp.StatusCode).To(Equal(200))
			Expect(recorder.ReceivedRequests()).To(HaveLen(1))
			Expect(recorder.ReceivedRequests()[0].RequestURI).To(Equal("/foo%20bar"))
		})
	})
})
