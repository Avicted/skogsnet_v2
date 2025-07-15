package main

import (
	"bufio"
	"errors"
	"strings"
	"testing"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Mock serial.Port
type mockPort struct{}

// GetModemStatusBits implements serial.Port.
func (m *mockPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	panic("unimplemented")
}

// ResetInputBuffer implements serial.Port.
func (m *mockPort) ResetInputBuffer() error {
	panic("unimplemented")
}

// ResetOutputBuffer implements serial.Port.
func (m *mockPort) ResetOutputBuffer() error {
	panic("unimplemented")
}

// SetDTR implements serial.Port.
func (m *mockPort) SetDTR(dtr bool) error {
	panic("unimplemented")
}

// SetMode implements serial.Port.
func (m *mockPort) SetMode(mode *serial.Mode) error {
	panic("unimplemented")
}

// SetRTS implements serial.Port.
func (m *mockPort) SetRTS(rts bool) error {
	panic("unimplemented")
}

// SetReadTimeout implements serial.Port.
func (m *mockPort) SetReadTimeout(t time.Duration) error {
	panic("unimplemented")
}

func (m *mockPort) Read(p []byte) (int, error)  { return 0, nil }
func (m *mockPort) Write(p []byte) (int, error) { return 0, nil }
func (m *mockPort) Close() error                { return nil }
func (m *mockPort) Break(time.Duration) error   { return nil }
func (m *mockPort) Drain() error                { return nil }

type errorScanner struct {
	called bool
}

func (e *errorScanner) Scan() bool   { return false }
func (e *errorScanner) Text() string { return "" }
func (e *errorScanner) Err() error   { return errors.New("scan error") }

func TestInitSerialPort_Success(t *testing.T) {
	origGetSerialPort := getSerialPort
	origSerialOpen := serialOpen
	origLogInfo := logInfo
	origLogError := logError
	origOsExit := osExit

	getSerialPort = func() (*enumerator.PortDetails, error) {
		return &enumerator.PortDetails{Name: "/dev/ttyACM0"}, nil
	}
	serialOpen = func(name string, mode *serial.Mode) (serial.Port, error) {
		return &mockPort{}, nil
	}
	logInfo = func(format string, args ...interface{}) {}
	logError = func(format string, args ...interface{}) {}
	osExit = func(code int) { t.Fatalf("osExit called unexpectedly") }

	defer func() {
		getSerialPort = origGetSerialPort
		serialOpen = origSerialOpen
		logInfo = origLogInfo
		logError = origLogError
		osExit = origOsExit
	}()

	port := initSerialPort()
	if port == nil {
		t.Error("Expected serial port, got nil")
	}
}

func TestInitSerialPort_GetSerialPortError(t *testing.T) {
	origGetSerialPort := getSerialPort
	origLogError := logError
	origOsExit := osExit

	getSerialPort = func() (*enumerator.PortDetails, error) {
		return nil, errors.New("no ports")
	}
	logError = func(format string, args ...interface{}) {}
	osExit = func(code int) { panic("osExit called") }

	defer func() {
		getSerialPort = origGetSerialPort
		logError = origLogError
		osExit = origOsExit
	}()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected osExit to be called on getSerialPort error")
		}
	}()

	initSerialPort()
}

func TestInitSerialPort_SerialOpenError(t *testing.T) {
	origGetSerialPort := getSerialPort
	origSerialOpen := serialOpen
	origLogInfo := logInfo
	origLogError := logError
	origOsExit := osExit

	getSerialPort = func() (*enumerator.PortDetails, error) {
		return &enumerator.PortDetails{Name: "/dev/tty/AMC0"}, nil
	}
	serialOpen = func(name string, mode *serial.Mode) (serial.Port, error) {
		return nil, errors.New("open error")
	}
	logInfo = func(format string, args ...interface{}) {}
	logError = func(format string, args ...interface{}) {}
	osExit = func(code int) { panic("osExit called") }

	defer func() {
		getSerialPort = origGetSerialPort
		serialOpen = origSerialOpen
		logInfo = origLogInfo
		logError = origLogError
		osExit = origOsExit
	}()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected osExit to be called on serial.Open error")
		}
	}()

	initSerialPort()
}

func TestGetSerialPortImpl_EnumeratorError(t *testing.T) {
	origGetDetailedPortsList := enumeratorGetDetailedPortsList
	enumeratorGetDetailedPortsList = func() ([]*enumerator.PortDetails, error) {
		return nil, errors.New("enumerator error")
	}
	defer func() { enumeratorGetDetailedPortsList = origGetDetailedPortsList }()

	_, err := getSerialPortImpl()
	if err == nil || err.Error() != "enumerator error: enumerator error" {
		t.Errorf("Expected enumerator error, got: %v", err)
	}
}

func TestGetSerialPortImpl_NoPorts(t *testing.T) {
	origGetDetailedPortsList := enumeratorGetDetailedPortsList
	enumeratorGetDetailedPortsList = func() ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{}, nil
	}
	defer func() { enumeratorGetDetailedPortsList = origGetDetailedPortsList }()

	_, err := getSerialPortImpl()
	if err == nil || err.Error() != "no serial ports found" {
		t.Errorf("Expected no serial ports found error, got: %v", err)
	}
}

func TestGetSerialPortImpl_PortFound(t *testing.T) {
	origGetDetailedPortsList := enumeratorGetDetailedPortsList
	origPortName := portName
	testPort := "COM1"
	portName = &testPort
	enumeratorGetDetailedPortsList = func() ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{
			{Name: "COM1"},
			{Name: "COM2"},
		}, nil
	}
	defer func() {
		enumeratorGetDetailedPortsList = origGetDetailedPortsList
		portName = origPortName
	}()

	port, err := getSerialPortImpl()
	if err != nil {
		t.Fatalf("Expected port, got error: %v", err)
	}
	if port.Name != "COM1" {
		t.Errorf("Expected port COM1, got: %s", port.Name)
	}
}

func TestGetSerialPortImpl_PortNotFound(t *testing.T) {
	origGetDetailedPortsList := enumeratorGetDetailedPortsList
	origPortName := portName
	testPort := "COM3"
	portName = &testPort
	enumeratorGetDetailedPortsList = func() ([]*enumerator.PortDetails, error) {
		return []*enumerator.PortDetails{
			{Name: "COM1"},
			{Name: "COM2"},
		}, nil
	}
	defer func() {
		enumeratorGetDetailedPortsList = origGetDetailedPortsList
		portName = origPortName
	}()

	_, err := getSerialPortImpl()
	if err == nil || err.Error() != "specified port COM3 not found in available ports" {
		t.Errorf("Expected port not found error, got: %v", err)
	}
}

func TestReadFromSerial_Success(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("hello\n"))
	line, err := readFromSerial(scanner)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if line != "hello" {
		t.Errorf("Expected 'hello', got: %s", line)
	}
}

func TestReadFromSerial_Error(t *testing.T) {
	scanner := &errorScanner{}
	line, err := readFromSerial(scanner)
	if err == nil || err.Error() != "error reading from serial: scan error" {
		t.Errorf("Expected scan error, got: %v", err)
	}
	if line != "" {
		t.Errorf("Expected empty string, got: %s", line)
	}
}

func TestReadFromSerial_EOF(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	line, err := readFromSerial(scanner)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if line != "" {
		t.Errorf("Expected empty string at EOF, got: %s", line)
	}
}
