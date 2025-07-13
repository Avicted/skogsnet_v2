package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type Measurement struct {
	UnixTimestamp      int64
	TemperatureCelsius float64
	HumidityPercentage float64
}

const portName = "/dev/ttyACM0"
const baudRate = 9600
const measurementFileName = "measurements.dat"

func initialize_serial_connection() (*enumerator.PortDetails, error) {
	fmt.Println("Initializing serial connection...")

	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("enumerator error: %w", err)
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no serial ports found")
	}

	fmt.Println("Available ports:")
	for _, port := range ports {
		fmt.Printf("- %s\n", port.Name)
	}

	for _, port := range ports {
		if port.Name == portName {
			fmt.Printf("Using port: %s\n", port.Name)
			return port, nil
		}
	}
	return nil, fmt.Errorf("no suitable serial port found")
}

func read_from_serial(scanner *bufio.Scanner) (string, error) {
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading from serial: %w", err)
	}
	return "", nil
}

func deserialize_data(data string) (Measurement, error) {
	var measurement Measurement
	type raw struct {
		TemperatureCelsius float64 `json:"temperature_celcius"`
		HumidityPercentage float64 `json:"humidity"`
	}
	var r raw
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return Measurement{}, fmt.Errorf("failed to deserialize data: %w", err)
	}
	measurement.TemperatureCelsius = r.TemperatureCelsius
	measurement.HumidityPercentage = r.HumidityPercentage
	measurement.UnixTimestamp = time.Now().UnixMilli()
	return measurement, nil
}

func open_measurement_file(fileName string) (*os.File, error) {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
		header := "UnixTimestampInMilliseconds\tTemperatureCelcius\tHumidity\n"
		if _, err := file.WriteString(header); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write header to file: %w", err)
		}
		return file, nil
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for appending: %w", err)
	}

	return file, nil
}

func write_to_file(file *os.File, measurement Measurement) error {
	line := fmt.Sprintf("%d\t%.6f\t%.6f\n", measurement.UnixTimestamp, measurement.TemperatureCelsius, measurement.HumidityPercentage)
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write measurement to file: %w", err)
	}
	return nil
}

func print_to_console(measurement Measurement) {
	t := time.UnixMilli(measurement.UnixTimestamp)
	fmt.Printf("Measurement at %s: Temperature = %.2f °C, Humidity = %.2f%%\n",
		t.Format("2006-01-02 15:04:05"), measurement.TemperatureCelsius, measurement.HumidityPercentage)
}

func main() {
	fmt.Println("Skogsnet v2")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	portDetails, err := initialize_serial_connection()
	if err != nil {
		log.Fatal(err)
		return
	}

	mode := &serial.Mode{BaudRate: baudRate}
	serialPort, err := serial.Open(portDetails.Name, mode)
	if err != nil {
		log.Fatal("Failed to open serial port:", err)
		return
	}
	defer serialPort.Close()

	file, err := open_measurement_file(measurementFileName)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(serialPort)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Graceful shutdown requested. Exiting...")
			return
		default:
			line, err := read_from_serial(scanner)
			if err != nil {
				log.Println(err)
				continue
			}
			if line == "" {
				log.Println("No data read from serial port. Retrying...")
				continue
			}

			measurement, err := deserialize_data(line)
			if err != nil {
				log.Println(err)
				continue
			}

			if err := write_to_file(file, measurement); err != nil {
				log.Printf("Error writing to file: %v", err)
				continue
			}
			print_to_console(measurement)
		}
	}
}
