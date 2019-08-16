package handlers

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/alphagov/router/logger"
)

var TLSSkipVerify bool

func NewBackendHandler(backendURL *url.URL, connectTimeout, headerTimeout time.Duration, logger logger.Logger) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	proxy.Transport = newBackendTransport(connectTimeout, headerTimeout, logger)

	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)

		// Set the Host header to match the backend hostname instead of the one from the incoming request.
		req.Host = backendURL.Host

		// Setting a blank User-Agent causes the http lib not to output one, whereas if there
		// is no header, it will output a default one.
		// See: https://github.com/golang/go/blob/release-branch.go1.5/src/net/http/request.go#L419
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
	logger  logger.Logger
}

// Construct a backendTransport that wraps an http.Transport and implements http.RoundTripper.
// This allows us to intercept the response from the backend and modify it before it's copied
// back to the client.
func newBackendTransport(connectTimeout, headerTimeout time.Duration, logger logger.Logger) (transport *backendTransport) {
	transport = &backendTransport{&http.Transport{}, logger}

	transport.wrapped.Dial = func(network, address string) (net.Conn, error) {
		return net.DialTimeout(network, address, connectTimeout)
	}
	// Allow the proxy to keep more than the default (2) keepalive connections
	// per upstream.
	transport.wrapped.MaxIdleConnsPerHost = 20
	transport.wrapped.ResponseHeaderTimeout = headerTimeout

	if TLSSkipVerify {
		transport.wrapped.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return
}

func (bt *backendTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = bt.wrapped.RoundTrip(req)
	if err == nil {
		populateViaHeader(resp.Header, fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor))
	} else {
		// Log the error (deferred to allow special case error handling to add/change details)
		logDetails := map[string]interface{}{"error": err.Error(), "status": 500}
		defer bt.logger.LogFromBackendRequest(logDetails, req)
		defer logger.NotifySentry(logger.ReportableError{ Error: err, Request: req, Response: resp })

		// Intercept some specific errors and generate an appropriate HTTP error response
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				logDetails["status"] = 504
				return newErrorResponse(504), nil
			}
		}
		if strings.Contains(err.Error(), "connection refused") {
			logDetails["status"] = 502
			return newErrorResponse(502), nil
		}

		// 500 for all other errors
		return newErrorResponse(500), nil
	}
	return
}

func newErrorResponse(status int) (resp *http.Response) {
	resp = &http.Response{StatusCode: status}
	resp.Body = ioutil.NopCloser(strings.NewReader(""))
	return
}
