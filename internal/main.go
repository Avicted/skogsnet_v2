package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
const dbFileName = "measurements.db"

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

func deserializeData(data string) (Measurement, error) {
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

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createTable := `
    CREATE TABLE IF NOT EXISTS measurements (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER,
        temperature REAL,
        humidity REAL
    );`

	_, err = db.Exec(createTable)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func insertMeasurement(db *sql.DB, m Measurement) error {
	_, err := db.Exec(
		"INSERT INTO measurements (timestamp, temperature, humidity) VALUES (?, ?, ?)",
		m.UnixTimestamp, m.TemperatureCelsius, m.HumidityPercentage,
	)

	return err
}

func printToConsole(measurement Measurement) {
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

	db, err := openDatabase(dbFileName)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

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

			measurement, err := deserializeData(line)
			if err != nil {
				log.Println(err)
				continue
			}

			if err := insertMeasurement(db, measurement); err != nil {
				log.Printf("Error writing to database: %v", err)
				continue
			}
			printToConsole(measurement)
		}
	}
}
