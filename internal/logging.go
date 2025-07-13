package main

import (
	"flag"
	"log"
	"os"
	"time"
)

var (
	logFile            = flag.String("log-file", "", "Log output to file (optional)")
	lastWarn           time.Time
	lastTimeoutWarn    time.Time
	lastDeserializeErr time.Time
	lastInsertErr      time.Time
	throttleInterval   = 5 * time.Second
)

func setupLogging() {
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(f)
	}
}

func logInfo(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func logWarn(format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}

func logError(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}

func throttledLogWarn(last *time.Time, format string, v ...any) {
	if time.Since(*last) > throttleInterval {
		logWarn(format, v...)
		*last = time.Now()
	}
}

func throttledLogError(last *time.Time, format string, v ...any) {
	if time.Since(*last) > throttleInterval {
		logError(format, v...)
		*last = time.Now()
	}
}
