package router

// TODO: remove this file and use rs/zerolog throughout.

import "log"

var EnableDebugOutput bool

func logWarn(msg ...interface{}) {
	log.Println(msg...)
}

func logInfo(msg ...interface{}) {
	log.Println(msg...)
}

func logDebug(msg ...interface{}) {
	if EnableDebugOutput {
		log.Println(msg...)
	}
}
