package logger

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	sentry "github.com/getsentry/sentry-go"
)

type RecoveredError struct {
	ErrorMessage string
}

func (re RecoveredError) Error() string {
	return re.ErrorMessage
}

type ReportableError struct {
	Error    error
	Request  *http.Request
	Response *http.Response
}

func (re ReportableError) hint() *sentry.EventHint {
	return &sentry.EventHint{
		Request:  re.Request,
		Response: re.Response,
	}
}

func (re ReportableError) scope() *sentry.Scope {
	scope := sentry.NewScope()
	if re.hint().Request != nil {
		scope.SetRequest(re.hint().Request)
	}
	if re.hint().Response != nil {
		scope.SetExtra("Response Status", re.hint().Response.Status)
	}
	return scope
}

func (re ReportableError) timeoutError() bool {
	var oerr *net.OpError
	if errors.As(re.Error, &oerr) {
		return oerr.Timeout()
	}
	return false
}

func (re ReportableError) ignorableError() bool {
	// We don't want to hear about timeouts. These get visibility elsewhere.
	return re.timeoutError()
}

func NotifySentry(re ReportableError) {
	if re.ignorableError() {
		return
	}

	// We don't need to set SENTRY_ENVIRONMENT, SENTRY_DSN or SENTRY_RELEASE
	// in ClientOptions as they are automatically picked up as env vars.
	// https://docs.sentry.io/platforms/go/config/
	client, err := sentry.NewClient(sentry.ClientOptions{})

	if err != nil {
		log.Printf("router: Sentry initialization failed: %v\n", err)
		return
	}

	hub := sentry.NewHub(client, re.scope())
	hub.CaptureException(re.Error)
	sentry.Flush(time.Second * 5)
}
