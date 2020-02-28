package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/textproto"

	. "github.com/onsi/gomega"
)

func routerRequest(path string, optionalPort ...int) *http.Response {
	return routerRequestWithMethod("GET", path, optionalPort...)
}

func routerRequestWithMethod(method string, path string, optionalPort ...int) *http.Response {
	return doRequest(newRequest(method, routerURL(path, optionalPort...)))
}

func routerRequestWithHeaders(path string, headers map[string]string, optionalPort ...int) *http.Response {
	return doRequest(newRequestWithHeaders("GET", routerURL(path, optionalPort...), headers))
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	Expect(err).To(BeNil())
	return req
}

func newRequestWithHeaders(method, url string, headers map[string]string) *http.Request {
	req := newRequest(method, url)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func doRequest(req *http.Request) *http.Response {
	if _, ok := req.Header[textproto.CanonicalMIMEHeaderKey("User-Agent")]; !ok {
		// Setting a blank User-Agent causes the http lib not to output one, whereas if there
		// is no header, it will output a default one.
		// See: https://github.com/golang/go/blob/release-branch.go1.5/src/net/http/request.go#L419
		req.Header.Set("User-Agent", "")
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	Expect(err).To(BeNil())
	return resp
}

func doHTTP10Request(req *http.Request) *http.Response {
	conn, err := net.Dial("tcp", req.URL.Host)
	Expect(err).To(BeNil())
	defer conn.Close()

	if req.Method == "" {
		req.Method = "GET"
	}
	req.Proto = "HTTP/1.0"
	req.ProtoMinor = 0
	fmt.Fprintf(conn, "%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto)
	err = req.Header.Write(conn)
	Expect(err).To(BeNil())
	fmt.Fprintf(conn, "\r\n")

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	Expect(err).To(BeNil())
	return resp
}

func readBody(resp *http.Response) string {
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	return string(bytes)
}

func readJSONBody(resp *http.Response, data interface{}) {
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(bytes, data)
	Expect(err).To(BeNil())
}
