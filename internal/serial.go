package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func mustInitSerialPort() serial.Port {
	portDetails, err := initializeSerialConnection()
	if err != nil {
		logError(err.Error())
		os.Exit(1)
	}

	mode := &serial.Mode{BaudRate: *baudRate}
	serialPort, err := serial.Open(portDetails.Name, mode)
	if err != nil {
		log.Fatal("Failed to open serial port:", err)
		os.Exit(1)
	}

	return serialPort
}

func initializeSerialConnection() (*enumerator.PortDetails, error) {
	logInfo("Initializing serial connection...")

	ports, err := enumerator.GetDetailedPortsList()
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

func readFromSerial(scanner *bufio.Scanner) (string, error) {
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading from serial: %w", err)
	}

	return "", nil
}
