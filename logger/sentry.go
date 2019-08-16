package logger

import (
  "log"
  "time"
  "net/http"

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
  return &sentry.Scope{}
}

func NotifySentry(re ReportableError) {
  // We don't need to set SENTRY_ENVIRONMENT, SENTRY_DSN or SENTRY_RELEASE
  // in ClientOptions as they are automatically picked up as env vars.
  // https://docs.sentry.io/platforms/go/config/
  client, err := sentry.NewClient(sentry.ClientOptions{})
  if err != nil {
    log.Printf("router: Sentry initialization failed: %v\n", err)
    return
  }

  client.CaptureException(re.Error, re.hint(), re.scope())
  client.Flush(time.Second * 5)
}
