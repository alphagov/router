package router

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route", func() {
	var route *Route

	BeforeEach(func() {
		route = &Route{}
	})

	Describe("backend", func() {
		Context("when schema is 'gone', but content items has details", func() {
			It("should return a backend", func() {
				backendID := "some-backend"
				route.SchemaName = stringPtr("gone")
				route.BackendID = &backendID
				route.Details = stringPtr(`{"key": "value"}`)
				Expect(route.backend()).To(Equal(&backendID))
			})

			It("should return 'frontend' if backend is nil", func() {
				route.SchemaName = stringPtr("gone")
				route.Details = stringPtr(`{"key": "value"}`)
				Expect(*route.backend()).To(Equal("frontend"))
			})
		})

		Context("when schema is not 'gone'", func() {
			It("should return a backend", func() {
				backendID := "some-backend"
				route.BackendID = &backendID
				Expect(route.backend()).To(Equal(&backendID))
			})
		})
	})

	Describe("handlerType", func() {
		Context("when route is a redirect", func() {
			It("should return 'redirect'", func() {
				route.SchemaName = stringPtr("redirect")
				Expect(route.handlerType()).To(Equal(HandlerTypeRedirect))
			})
		})

		Context("when route is gone", func() {
			It("should return 'gone'", func() {
				route.SchemaName = stringPtr("gone")
				Expect(route.handlerType()).To(Equal(HandlerTypeGone))
			})
		})

		Context("when route is neither redirect nor gone", func() {
			It("should return 'backend'", func() {
				Expect(route.handlerType()).To(Equal(HandlerTypeBackend))
			})
		})
	})

	Describe("gone", func() {
		Context("when schema is 'gone' and details is nil", func() {
			It("should return true", func() {
				route.SchemaName = stringPtr("gone")
				Expect(route.gone()).To(BeTrue())
			})
		})

		Context("when schema is 'gone' and details is empty", func() {
			It("should return true", func() {
				route.SchemaName = stringPtr("gone")
				details := "{}"
				route.Details = &details
				Expect(route.gone()).To(BeTrue())
			})
		})

		Context("when schema is 'gone' and details is not empty", func() {
			It("should return false", func() {
				route.SchemaName = stringPtr("gone")
				details := `{"key1": "value", "key2": null}`
				route.Details = &details
				Expect(route.gone()).To(BeFalse())
			})
		})

		Context("when schema is 'gone' and details is invalid json", func() {
			It("should return true", func() {
				route.SchemaName = stringPtr("gone")
				details := "{invalid}"
				route.Details = &details
				Expect(route.gone()).To(BeTrue())
			})
		})

		Context("when schema is not 'gone'", func() {
			It("should return false", func() {
				Expect(route.gone()).To(BeFalse())
			})
		})
	})

	Describe("redirect", func() {
		Context("when schema is 'redirect'", func() {
			It("should return true", func() {
				route.SchemaName = stringPtr("redirect")
				Expect(route.redirect()).To(BeTrue())
			})
		})

		Context("when schema is not 'redirect'", func() {
			It("should return false", func() {
				route.SchemaName = stringPtr("guidance")
				Expect(route.redirect()).To(BeFalse())
			})
		})
	})

	Describe("segmentsMode", func() {
		Context("when segments mode is nil and schema is 'redirect'", func() {
			It("should return 'preserve' if route type is 'prefix'", func() {
				route.SchemaName = stringPtr("redirect")
				route.RouteType = stringPtr("prefix")
				Expect(route.segmentsMode()).To(Equal("preserve"))
			})

			It("should return 'ignore' if route type is not 'prefix'", func() {
				route.SchemaName = stringPtr("redirect")
				route.RouteType = stringPtr("other")
				Expect(route.segmentsMode()).To(Equal("ignore"))
			})
		})

		Context("when segments mode is set and schema is redirect", func() {
			It("should return set segments mode regardless if prefix", func() {
				route.SchemaName = stringPtr("redirect")
				route.RouteType = stringPtr("prefix")
				route.SegmentsMode = stringPtr("ignore")
				Expect(route.segmentsMode()).To(Equal("ignore"))
			})

			It("should return set segments mode regardless if not prefix", func() {
				route.SchemaName = stringPtr("redirect")
				route.RouteType = stringPtr("exact")
				route.SegmentsMode = stringPtr("preserve")
				Expect(route.segmentsMode()).To(Equal("preserve"))
			})
		})
	})
})

func stringPtr(s string) *string {
	return &s
}
