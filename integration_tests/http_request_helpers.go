package integration

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
)

func routerRequest(path string) *http.Response {
	req, err := http.NewRequest("GET", routerURL(path), nil)
	Expect(err).To(BeNil())
	resp, err := http.DefaultTransport.RoundTrip(req)
	Expect(err).To(BeNil())
	return resp
}

func readBody(resp *http.Response) string {
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	return string(bytes)
}
