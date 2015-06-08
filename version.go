package main

import (
	"fmt"
	"runtime"
)

// populated by -ldflags from Makefile
var version = "unknown"

func versionInfo() string {
	return fmt.Sprintf("build: %s (compiler: %s)", version, runtime.Version())
}
