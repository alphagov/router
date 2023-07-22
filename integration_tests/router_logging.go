package integration

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	. "github.com/onsi/gomega"
)

var (
	tempLogfile *os.File
)

func setupTempLogfile() error {
	file, err := os.CreateTemp("", "router_error_log")
	if err != nil {
		return err
	}
	tempLogfile = file
	return nil
}

func resetTempLogfile() {
	tempLogfile.Seek(0, 0)
	tempLogfile.Truncate(0)
}

func cleanupTempLogfile() {
	if tempLogfile != nil {
		tempLogfile.Close()
		os.Remove(tempLogfile.Name())
	}
}

type routerLogEntry struct {
	Timestamp time.Time              `json:"@timestamp"`
	Fields    map[string]interface{} `json:"@fields"`
}

func lastRouterErrorLogLine() []byte {
	var line []byte

	Eventually(func() ([]byte, error) {
		scanner := bufio.NewScanner(tempLogfile)
		for scanner.Scan() {
			line = scanner.Bytes()
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return line, nil
	}).ShouldNot(BeNil(), "No log line found after 1 second")

	return line
}

func lastRouterErrorLogEntry() *routerLogEntry {
	line := lastRouterErrorLogLine()
	var entry *routerLogEntry
	err := json.Unmarshal(line, &entry)
	Expect(err).To(BeNil())
	return entry
}
