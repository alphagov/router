package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Logger interface {
	Log(fields map[string]interface{})
	LogFromClientRequest(fields map[string]interface{}, req *http.Request)
	LogFromBackendRequest(fields map[string]interface{}, req *http.Request)
}

type logEntry struct {
	Timestamp time.Time              `json:"@timestamp"`
	Fields    map[string]interface{} `json:"@fields"`
}

type jsonLogger struct {
	mu     sync.Mutex
	writer io.Writer
}

func New(output interface{}) (logger Logger, err error) {
	l := &jsonLogger{}
	switch out := output.(type) {
	case io.Writer:
		l.writer = out
	case string:
		if out == "STDERR" {
			l.writer = os.Stderr
		} else if out == "STDOUT" {
			l.writer = os.Stdout
		} else {
			l.writer, err = os.OpenFile(out, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("Invalid output type %T(%v)", output, output)
	}
	return l, nil
}

func (l *jsonLogger) writeLine(line []byte) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.writer.Write(append(line, 10))
	return
}

func (l *jsonLogger) Log(fields map[string]interface{}) {
	entry := &logEntry{time.Now(), fields}
	line, err := json.Marshal(entry)
	if err != nil {
		log.Printf("router/logger: Error encoding JSON: %v", err)
	}
	err = l.writeLine(line)
	return
}

func (l *jsonLogger) LogFromClientRequest(fields map[string]interface{}, req *http.Request) {
	fields["request_method"] = req.Method
	fields["request"] = fmt.Sprintf("%s %s %s", req.Method, req.RequestURI, req.Proto)
	fields["varnish_id"] = req.Header.Get("X-Varnish")

	l.Log(fields)
}

func (l *jsonLogger) LogFromBackendRequest(fields map[string]interface{}, req *http.Request) {
	// The request at this point is the request to the backend, not the original client request,
	// hence the backend host details are in the req.Host field
	fields["upstream_addr"] = req.Host

	l.LogFromClientRequest(fields, req)
}
