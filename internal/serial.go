package main

import (
	"fmt"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// Allow both *bufio.Scanner and your *errorScanner can be used.
type Scanner interface {
	Scan() bool
	Text() string
	Err() error
}

var serialOpen = serial.Open
var getSerialPort = getSerialPortImpl
var readFromSerial = readFromSerialImpl
var enumeratorGetDetailedPortsList = enumerator.GetDetailedPortsList

func initSerialPort() serial.Port {
	logInfo("Initializing serial connection...")

	portDetails, err := getSerialPort()
	if err != nil {
		logError(err.Error())
		osExit(1)
		return nil
	}

	mode := &serial.Mode{BaudRate: *baudRate}
	serialPort, err := serialOpen(portDetails.Name, mode)
	if err != nil {
		logError(fmt.Sprintf("Failed to open serial port: %v", err))
		osExit(1)
		return nil
	}

	return serialPort
}

func getSerialPortImpl() (*enumerator.PortDetails, error) {
	ports, err := enumeratorGetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerator error: %w", err)
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no serial ports found")
	}

	logInfo("Available ports:")
	for _, port := range ports {
		logInfo("- %s", port.Name)
	}

	for _, port := range ports {
		if port.Name == *portName {
			logInfo("Using port: %s", port.Name)
			return port, nil
		}
	}

	return nil, fmt.Errorf("specified port %s not found in available ports", *portName)
}

func readFromSerialImpl(scanner Scanner) (string, error) {
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading from serial: %w", err)
	}

	return "", nil
}
