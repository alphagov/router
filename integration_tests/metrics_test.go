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
})
