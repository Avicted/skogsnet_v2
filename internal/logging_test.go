package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSetupLogging(t *testing.T) {
	logFile = new(string)
	*logFile = "test.log"
	defer os.Remove(*logFile)

	setupLogging()

	logInfo("Test log message")

	// Check if the log file was created and contains the log message
	data, err := os.ReadFile(*logFile)
	if err != nil {
		t.Fatalf("Expected log file to be created, got error: %v", err)
	}
	if !strings.Contains(string(data), "[INFO] Test log message") {
		t.Errorf("Expected log file to contain '[INFO] Test log message', got: %s", string(data))
	}
}

func TestSetupLogging_FileOpenError(t *testing.T) {
	// Set logFile to a path that cannot be created
	logFile = new(string)
	*logFile = "/this/path/does/not/exist/test.log"

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Mock osExit to detect exit
	exitCalled := false
	mockExit := func(code int) {
		exitCalled = true
	}
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	setupLogging()

	// Assert that osExit was called
	if !exitCalled {
		t.Errorf("Expected os.Exit to be called when log file cannot be opened")
	}

	// Assert that error was logged
	if !strings.Contains(buf.String(), "Failed to open log file") {
		t.Errorf("Expected error log output, got: %s", buf.String())
	}
}

func TestLogInfoOutput(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	logInfo("Hello %s", "world")
	if !strings.Contains(buf.String(), "[INFO] Hello world") {
		t.Errorf("Expected info log output, got: %s", buf.String())
	}
}

func TestLogWarnOutput(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	logWarn("Warning: %s", "something went wrong")
	if !strings.Contains(buf.String(), "[WARN] Warning: something went wrong") {
		t.Errorf("Expected warn log output, got: %s", buf.String())
	}
}

func TestLogErrorOutput(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	logError("Error occurred: %s", "file not found")
	if !strings.Contains(buf.String(), "[ERROR] Error occurred: file not found") {
		t.Errorf("Expected error log output, got: %s", buf.String())
	}
}

func TestLogFatal(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	exitCalled := false
	mockExit := func(code int) {
		exitCalled = true
	}
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	logFatal("Fatal error: %s", "critical failure")
	if !strings.Contains(buf.String(), "[FATAL] Fatal error: critical failure") {
		t.Errorf("Expected fatal log output, got: %s", buf.String())
	}
	if !exitCalled {
		t.Errorf("Expected os.Exit to be called")
	}
}

func TestThrottledLogWarnOutput(t *testing.T) {
	orig := throttleInterval
	throttleInterval = 10 * time.Millisecond
	defer func() { throttleInterval = orig }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	var last time.Time
	throttledLogWarn(&last, "Throttled warn: %d", 1)
	throttledLogWarn(&last, "Throttled warn: %d", 2)
	time.Sleep(throttleInterval + 5*time.Millisecond)
	throttledLogWarn(&last, "Throttled warn: %d", 3)

	logs := buf.String()
	if !strings.Contains(logs, "Throttled warn: 1") {
		t.Errorf("Expected first throttled warn log")
	}
	if strings.Contains(logs, "Throttled warn: 2") {
		t.Errorf("Second throttled warn log should not appear due to throttling")
	}
	if !strings.Contains(logs, "Throttled warn: 3") {
		t.Errorf("Expected third throttled warn log after interval")
	}
}

func TestThrottledLogErrorOutput(t *testing.T) {
	orig := throttleInterval
	throttleInterval = 10 * time.Millisecond
	defer func() { throttleInterval = orig }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	var last time.Time
	throttledLogError(&last, "Throttled error: %d", 1)
	throttledLogError(&last, "Throttled error: %d", 2)
	time.Sleep(throttleInterval + 5*time.Millisecond)
	throttledLogError(&last, "Throttled error: %d", 3)

	logs := buf.String()
	if !strings.Contains(logs, "Throttled error: 1") {
		t.Errorf("Expected first throttled error log")
	}
	if strings.Contains(logs, "Throttled error: 2") {
		t.Errorf("Second throttled error log should not appear due to throttling")
	}
	if !strings.Contains(logs, "Throttled error: 3") {
		t.Errorf("Expected third throttled error log after interval")
	}
}
