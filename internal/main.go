package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.bug.st/serial"
)

var (
	portName       = flag.String("port", "/dev/ttyACM0", "Serial port name")
	baudRate       = flag.Int("baud", 9600, "Serial baud rate")
	dbFileName     = flag.String("db", "measurements.db", "SQLite database filename")
	exportCSV      = flag.String("export-csv", "", "Export measurements to CSV file and exit")
	serveDashboard = flag.Bool("dashboard", false, "Serve web dashboard at http://localhost:8080")
)

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

	portDetails, err := initializeSerialConnection()
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

	if *serveDashboard {
		serveAPI(db)
		http.Handle("/", http.FileServer(http.Dir("web-dashboard-static")))
		go func() {
			addr := "http://localhost:8080"
			logInfo("Web dashboard served at %s", addr)
			fmt.Printf("Web dashboard served at %s\n", addr)
			if err := http.ListenAndServe(":8080", nil); err != nil {
				logError("Dashboard server error: %v", err)
			}
		}()
	}

	scanner := bufio.NewScanner(serialPort)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Graceful shutdown requested. Exiting...")
			return
		default:
			line, err := readFromSerial(scanner)
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
