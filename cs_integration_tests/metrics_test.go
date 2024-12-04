package integration

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("/metrics API endpoint", func() {
	Context("response body", func() {
		It("should contain at least one metric", func() {
			resp := doRequest(newRequest("GET", routerURL(apiPort, "/metrics")))
			Expect(resp.StatusCode).To(Equal(200))
			Expect(readBody(resp)).To(ContainSubstring("router_"))
		})
	})
})
