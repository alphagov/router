package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type Logger interface {
	Log(fields *map[string]interface{})
}

type logEntry struct {
	Timestamp time.Time               `json:"@timestamp"`
	Fields    *map[string]interface{} `json:"@fields"`
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

func (l *jsonLogger) Log(fields *map[string]interface{}) {
	entry := &logEntry{time.Now(), fields}
	line, err := json.Marshal(entry)
	if err != nil {
		log.Printf("router/logger: Error encoding JSON: %v", err)
	}
	err = l.writeLine(line)
	return
}
