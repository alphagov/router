package integration

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
)

func routerRequest(path string, optionalPort ...int) *http.Response {
	return doRequest(newRequest("GET", routerURL(path, optionalPort...)))
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	Expect(err).To(BeNil())
	return req
}

func doRequest(req *http.Request) *http.Response {
	resp, err := http.DefaultTransport.RoundTrip(req)
	Expect(err).To(BeNil())
	return resp
}

func readBody(resp *http.Response) string {
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	return string(bytes)
}
