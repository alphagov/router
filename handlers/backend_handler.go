package handlers

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var TLSSkipVerify bool

func NewBackendHandler(
	backendID string,
	backendURL *url.URL,
	connectTimeout, headerTimeout time.Duration,
	logger zerolog.Logger,
) http.Handler {

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	proxy.Transport = newBackendTransport(
		backendID,
		connectTimeout, headerTimeout,
		logger,
	)

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
	backendID string

	wrapped *http.Transport
	logger  zerolog.Logger
}

// Construct a backendTransport that wraps an http.Transport and implements http.RoundTripper.
// This allows us to intercept the response from the backend and modify it before it's copied
// back to the client.
func newBackendTransport(
	backendID string,
	connectTimeout, headerTimeout time.Duration,
	logger zerolog.Logger,
) *backendTransport {

	transport := http.Transport{}

	transport.DialContext = (&net.Dialer{
		Timeout:   connectTimeout,   // Configured by caller
		KeepAlive: 30 * time.Second, // same as DefaultTransport
		DualStack: true,             // same as DefaultTransport
	}).DialContext

	// Remember, we have one transport per backend
	//
	// Using the below settings, and (for example) we have 25 backends
	//   25 * 60 = 1500
	// we will have a maximum of 1500 open idle connections
	//
	// The Go http.DefaultTransport sets this to 100,
	// we set to 60 because of potential file handle limits
	// because we have multiple backends
	transport.MaxIdleConns = 60
	// This is an arbitrarily selected number that is less than 60
	transport.MaxIdleConnsPerHost = 20
	// By default, idle connections do not expire,
	// unless they are closed by the other end of the connection,
	// and sometimes the other end will silently close the connection.
	// We should expire idle connections after a while.
	//
	// We arbitrarily chose 10 minutes
	transport.IdleConnTimeout = 10 * time.Minute

	// If we do not configure the timeouts, then connections will hang
	//
	// Configured by the caller
	transport.ResponseHeaderTimeout = headerTimeout
	//
	// Same values as http.DefaultTransport
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ExpectContinueTimeout = 1 * time.Second

	if TLSSkipVerify {
		// #nosec G402 -- TODO: fix tests to use TLS properly.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &backendTransport{backendID, &transport, logger}
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func (bt *backendTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var responseCode int
	var startTime = time.Now()

	backendRequestCountMetric.With(prometheus.Labels{
		"backend_id":     bt.backendID,
		"request_method": req.Method,
	}).Inc()

	defer func() {
		durationSeconds := time.Since(startTime).Seconds()

		backendResponseDurationSecondsMetric.With(prometheus.Labels{
			"backend_id":     bt.backendID,
			"request_method": req.Method,
			"response_code":  fmt.Sprintf("%d", responseCode),
		}).Observe(durationSeconds)
	}()

	resp, err = bt.wrapped.RoundTrip(req)
	if err != nil {
		var nerr net.Error
		switch {
		case errors.Is(err, syscall.ECONNREFUSED):
			responseCode = http.StatusBadGateway
		case errors.As(err, &nerr) && nerr.Timeout():
			responseCode = http.StatusGatewayTimeout
		default:
			responseCode = http.StatusInternalServerError
		}
		closeBody(resp)
		bt.logger.Error().
			Err(err).
			Int("status", responseCode).
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Msg("backend request error")

		return newErrorResponse(responseCode), nil
	}
	responseCode = resp.StatusCode
	populateViaHeader(resp.Header, fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor))
	return
}

func newErrorResponse(status int) (resp *http.Response) {
	resp = &http.Response{StatusCode: status}
	resp.Body = io.NopCloser(strings.NewReader(""))
	return
}
