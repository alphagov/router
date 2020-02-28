package integration

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("metrics API endpoint", func() {
	It("should return HTTP OK when receiving a request for /metrics", func() {
		resp := doRequest(newRequest("GET", routerAPIURL("/metrics")))
		Expect(resp.StatusCode).To(Equal(200))
	})

	Context("when looking at metrics returned in response", func() {
		var (
			responseBody string
		)

		BeforeEach(func() {
			resp := doRequest(newRequest("GET", routerAPIURL("/metrics")))
			Expect(resp.StatusCode).To(Equal(200))
			responseBody = readBody(resp)
		})

		It("should contain router internal server error metrics", func() {
			Skip("Metric is a vector so it is not registered by default")
			Expect(responseBody).To(ContainSubstring("router_internal_server_error_count"))
		})

		It("should contain routing table metrics", func() {
			Expect(responseBody).To(ContainSubstring("router_route_reload_count"))
			Expect(responseBody).To(ContainSubstring("router_route_reload_error_count"))

			Expect(responseBody).To(ContainSubstring("router_routes_count"))
		})
	})
})
