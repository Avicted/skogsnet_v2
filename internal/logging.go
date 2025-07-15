package main

import (
	"flag"
	"io"
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
	lastWeatherErr     time.Time
	throttleInterval   = 5 * time.Second
)

var logInfo = logInfoImpl
var logWarn = logWarnImpl
var logError = logErrorImpl
var logFatal = logFatalImpl

func setupLogging() {
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logError("Failed to open log file: %v", err)
			osExit(1)
		}

		multiWriter := io.MultiWriter(f, os.Stdout)
		log.SetOutput(multiWriter)
	}
}

func logInfoImpl(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func logWarnImpl(format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}

func logErrorImpl(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}

func logFatalImpl(format string, v ...any) {
	log.Printf("[FATAL] "+format, v...)
	osExit(1)
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
