package integration

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backend selection", func() {

	It("should 404 with no routes", func() {
		resp, err := http.Get("http://localhost:3169/")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(404))

		resp, err = http.Get("http://localhost:3169/foo")
		Expect(err).To(BeNil())
		Expect(resp.StatusCode).To(Equal(404))
	})
})
