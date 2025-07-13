package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
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

var (
	portName   = flag.String("port", "/dev/ttyACM0", "Serial port name")
	baudRate   = flag.Int("baud", 9600, "Serial baud rate")
	dbFileName = flag.String("db", "measurements.db", "SQLite database filename")
	exportCSV  = flag.String("export-csv", "", "Export measurements to CSV file and exit")
	logFile    = flag.String("log-file", "", "Log output to file (optional)")
)

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
		if port.Name == *portName {
			fmt.Printf("Using port: %s\n", port.Name)
			return port, nil
		}
	}
	return nil, fmt.Errorf("specified port %s not found in available ports", *portName)
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
	if db == nil {
		return errors.New("db is nil")
	}

	_, err := db.Exec(
		"INSERT INTO measurements (timestamp, temperature, humidity) VALUES (?, ?, ?)",
		m.UnixTimestamp, m.TemperatureCelsius, m.HumidityPercentage,
	)

	return err
}

func exportToCSV(db *sql.DB, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	rows, err := db.Query("SELECT timestamp, temperature, humidity FROM measurements ORDER BY timestamp")
	if err != nil {
		return err
	}
	defer rows.Close()
	fmt.Fprintln(file, "timestamp,temperature,humidity")
	for rows.Next() {
		var ts int64
		var temp, hum float64
		if err := rows.Scan(&ts, &temp, &hum); err != nil {
			return err
		}
		fmt.Fprintf(file, "%d,%.2f,%.2f\n", ts, temp, hum)
	}
	return nil
}

func printToConsole(measurement Measurement) {
	t := time.UnixMilli(measurement.UnixTimestamp)

	// ANSI color codes
	const (
		green  = "\033[32m"
		cyan   = "\033[36m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	fmt.Printf("%sMeasurement at %s%s\n", cyan, t.Format("2006-01-02 15:04:05"), reset)
	fmt.Printf("    %sTemperature:%s %s%.2f °C%s\n", green, reset, reset, measurement.TemperatureCelsius, reset)
	fmt.Printf("    %sHumidity:   %s %s%.2f %%%s\n", green, reset, reset, measurement.HumidityPercentage, reset)
}

func setupLogging() {
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(f)
	}
}

func logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func logWarn(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

func logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

var (
	lastWarn           time.Time
	lastTimeoutWarn    time.Time
	lastDeserializeErr time.Time
	lastInsertErr      time.Time
	throttleInterval   = 5 * time.Second
)

func throttledLogWarn(last *time.Time, format string, v ...interface{}) {
	if time.Since(*last) > throttleInterval {
		logWarn(format, v...)
		*last = time.Now()
	}
}

func throttledLogError(last *time.Time, format string, v ...interface{}) {
	if time.Since(*last) > throttleInterval {
		logError(format, v...)
		*last = time.Now()
	}
}

func main() {
	flag.Parse()
	setupLogging()

	if *exportCSV != "" {
		db, err := openDatabase(*dbFileName)
		if err != nil {
			logError("Failed to open database: %v", err)
			return
		}
		defer db.Close()
		if err := exportToCSV(db, *exportCSV); err != nil {
			logError("Export to CSV failed: %v", err)
		}
		logInfo("Exported measurements to %s", *exportCSV)
		return
	}

	logInfo("Skogsnet v2 started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	portDetails, err := initialize_serial_connection()
	if err != nil {
		log.Fatal(err)
		return
	}

	mode := &serial.Mode{BaudRate: *baudRate}
	serialPort, err := serial.Open(portDetails.Name, mode)
	if err != nil {
		log.Fatal("Failed to open serial port:", err)
		return
	}
	defer serialPort.Close()

	db, err := openDatabase(*dbFileName)
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
				if err.Error() == "timeout" {
					throttledLogWarn(&lastTimeoutWarn, "Serial read timeout. Retrying...")
					continue
				}
			}
			if line == "" {
				throttledLogWarn(&lastWarn, "No data read from serial port. Retrying...")
				time.Sleep(500 * time.Millisecond)
				continue
			}

			measurement, err := deserializeData(line)
			if err != nil {
				throttledLogError(&lastDeserializeErr, "Failed to deserialize data: %v", err)
				continue
			}

			if err := insertMeasurement(db, measurement); err != nil {
				throttledLogError(&lastInsertErr, "Failed to insert measurement into database: %v", err)
				continue
			}

			printToConsole(measurement)
		}
	}
}
