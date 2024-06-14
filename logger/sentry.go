package logger

import (
	"errors"
	"log"
	"net"
	"net/http"

	sentry "github.com/getsentry/sentry-go"
)

// TODO: use the Sentry API as intended + remove these wonky, reinvented wheels.
// See https://docs.sentry.io/platforms/go/guides/http/.

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

func InitSentry() {
	if err := sentry.Init(sentry.ClientOptions{}); err != nil {
		log.Printf("sentry.Init failed: %v\n", err)
	}
}

// Timeout returns true if and only if this ReportableError is a timeout.
func (re ReportableError) Timeout() bool {
	var oerr *net.OpError
	if errors.As(re.Error, &oerr) {
		return oerr.Timeout()
	}
	return false
}

// NotifySentry sends an event to sentry.io. Sentry is configurable via the
// environment variables SENTRY_ENVIRONMENT, SENTRY_DSN, SENTRY_RELEASE.
func NotifySentry(re ReportableError) {
	if re.Timeout() {
		return
	}

	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(s *sentry.Scope) {
		if re.Request != nil {
			s.SetRequest(re.Request)
		}
		if re.Response != nil {
			s.SetExtra("Response Status", re.Response.Status)
		}
		hub.CaptureException(re.Error)
	})
}
