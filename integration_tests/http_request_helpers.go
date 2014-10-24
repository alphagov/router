package integration

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
)

func routerRequest(path string) *http.Response {
	resp, err := http.Get("http://localhost:3169" + path)
	Expect(err).To(BeNil())
	return resp
}

func readBody(resp *http.Response) string {
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	return string(bytes)
}
