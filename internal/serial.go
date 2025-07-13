package main

import (
	"bufio"
	"fmt"

	"go.bug.st/serial/enumerator"
)

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
