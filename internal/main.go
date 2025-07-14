package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.bug.st/serial"
)

const (
	weatherFetchInterval   = 1 * time.Minute
	weatherFetchRetryDelay = 500 * time.Millisecond
	serialRetryDelay       = 500 * time.Millisecond
)

var (
	portName       = flag.String("port", "/dev/ttyACM0", "Serial port name")
	baudRate       = flag.Int("baud", 9600, "Serial baud rate")
	dbFileName     = flag.String("db", "measurements.db", "SQLite database filename")
	exportCSV      = flag.String("export-csv", "", "Export measurements to CSV file and exit")
	serveDashboard = flag.Bool("dashboard", false, "Serve web dashboard at http://localhost:8080")
	enableWeather  = flag.Bool("weather", false, "Enable periodic weather data fetching")
	weatherCity    = flag.String("city", "", "City name for weather data")
)

func main() {
	flag.Parse()
	setupLogging()

	if *exportCSV != "" {
		exportCSVAndExit(dbFileName, exportCSV)
		return
	}

	logInfo("Skogsnet v2 started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serialPort := mustInitSerialPort()
	defer serialPort.Close()

	db := mustInitDatabase(dbFileName)
	defer db.Close()

	enableWALMode(db)

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	if *enableWeather {
		startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	}

	if *serveDashboard {
		startDashboardServer(ctx, db, &wg)
	}

	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
}

func mainLoop(ctx context.Context, serialPort serial.Port, db *sql.DB, latestWeather *Weather, wg *sync.WaitGroup) {
	scanner := bufio.NewScanner(serialPort)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Graceful shutdown requested. Exiting...")
			wg.Wait()
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
				time.Sleep(serialRetryDelay)
				continue
			}

			measurement, err := deserializeData(line)
			if err != nil {
				throttledLogError(&lastDeserializeErr, "Failed to deserialize data: %v", err)
				continue
			}

			currentTimestamp := time.Now().UnixMilli()
			if err := insertMeasurement(db, measurement, currentTimestamp); err != nil {
				throttledLogError(&lastInsertErr, "Failed to insert measurement into database: %v", err)
				continue
			}

			printToConsole(measurement, latestWeather)
		}
	}
}
