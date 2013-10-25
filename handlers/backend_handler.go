package handlers

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"syscall"
	"time"
)

func NewBackendHandler(backendUrl *url.URL, connectTimeout, headerTimeout time.Duration) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(backendUrl)
	proxy.Transport = newBackendTransport(connectTimeout, headerTimeout)

	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)

		// Set the Host header to match the backend hostname instead of the one from the incoming request.
		req.Host = backendUrl.Host

		// Setting a blank User-Agent causes the http lib not to output one, whereas if there
		// is no header, it will output a default one.
		// See: http://code.google.com/p/go/source/browse/src/pkg/net/http/request.go?name=go1.1.2#349
		if _, present := req.Header["User-Agent"]; !present {
			req.Header.Set("User-Agent", "")
		}

		populateViaHeader(req.Header, fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor))
	}

	return proxy
}

func populateViaHeader(header http.Header, httpVersion string) {
	via := httpVersion + " router"
	if prior, ok := header["Via"]; ok {
		via = strings.Join(prior, ", ") + ", " + via
	}
	header.Set("Via", via)
}

type backendTransport struct {
	wrapped *http.Transport
}

// Construct a backendTransport that wraps an http.Transport and implements http.RoundTripper.
// This allows us to intercept the response from the backend and modify it before it's copied
// back to the client.
func newBackendTransport(connectTimeout, headerTimeout time.Duration) (transport *backendTransport) {
	transport = &backendTransport{&http.Transport{}}

	transport.wrapped.Dial = func(network, address string) (net.Conn, error) {
		return net.DialTimeout(network, address, connectTimeout)
	}
	// Allow the proxy to keep more than the default (2) keepalive connections
	// per upstream.
	transport.wrapped.MaxIdleConnsPerHost = 20
	transport.wrapped.ResponseHeaderTimeout = headerTimeout
	return
}

func (bt *backendTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = bt.wrapped.RoundTrip(req)
	if err == nil {
		populateViaHeader(resp.Header, fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor))
	} else {
		// Intercept timeout errors and generate an HTTP error response
		switch typedError := err.(type) {
		case *net.OpError:
			if typedError.Timeout() {
				return newErrorResponse(504), nil
			} else if typedError.Err == syscall.ECONNREFUSED {
				return newErrorResponse(502), nil
			}
		default:
			if err.Error() == "net/http: timeout awaiting response headers" {
				return newErrorResponse(504), nil
			}
		}
	}
	return
}

func newErrorResponse(status int) (resp *http.Response) {
	resp = &http.Response{StatusCode: status}
	resp.Body = ioutil.NopCloser(strings.NewReader(""))
	return
}
