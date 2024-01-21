package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("/metrics API endpoint", func() {
	Context("response body", func() {
		var responseBody string

		BeforeEach(func() {
			resp := doRequest(newRequest("GET", routerURL(apiPort, "/metrics")))
			Expect(resp).To(HaveHTTPStatus(200))
			responseBody = readBody(resp)
		})

		It("should contain at least one metric", func() {
			Expect(responseBody).To(ContainSubstring("router_"))
		})
	})
})
