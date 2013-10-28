package logger

import (
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"
)

type logEntry struct {
	Timestamp time.Time               `json:"@timestamp"`
	Fields    *map[string]interface{} `json:"@fields"`
}

type Logger struct {
	mu     sync.Mutex
	writer io.Writer
}

func New(w io.Writer) (l *Logger) {
	l = &Logger{writer: w}
	return
}

func (l *Logger) writeLine(line []byte) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.writer.Write(append(line, 10))
	return
}

func (l *Logger) Log(fields *map[string]interface{}) {
	entry := &logEntry{time.Now(), fields}
	line, err := json.Marshal(entry)
	if err != nil {
		log.Printf("router/logger: Error encoding JSON: %v", err)
	}
	err = l.writeLine(line)
	return
}
