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
	"sync"
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
	enableWeather  = flag.Bool("weather", false, "Enable periodic weather data fetching")
	weatherCity    = flag.String("city", "", "City name for weather data")
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
		logError(err.Error())
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

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		logError("Failed to enable WAL mode: %v", err)
	}

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var waitGroup sync.WaitGroup

	if *enableWeather {
		const weatherFetchInterval = 1 * time.Minute
		weatherTicker := time.NewTicker(weatherFetchInterval)
		defer weatherTicker.Stop()

		city := *weatherCity
		if city == "" {
			logError("No city specified for weather data")
			return
		}

	weatherInit:
		for {
			select {
			case <-ctx.Done():
				logInfo("Weather fetching loop stopped")
				return
			default:
				latestWeather, err = GetWeatherData(city)
				if err == nil {
					latestWeatherTimestamp = time.Now().UnixMilli()
					insertWeather(db, latestWeather, latestWeatherTimestamp)
					break weatherInit
				}
				logError("Initial weather fetch failed, retrying in 5s: %v", err)

				const weatherFetchRetryDelay = 500 * time.Millisecond
				time.Sleep(weatherFetchRetryDelay)
			}
		}

		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for {
				select {
				case <-ctx.Done():
					logInfo("Weather update goroutine stopped")
					return
				case <-weatherTicker.C:
					w, err := GetWeatherData(city)
					ts := time.Now().UnixMilli()
					if err == nil {
						latestWeather = w
						latestWeatherTimestamp = ts
						insertWeather(db, latestWeather, latestWeatherTimestamp)
					} else {
						throttledLogError(&lastWeatherErr, "Failed to get weather data for city %s: %v", city, err)
					}
				}
			}
		}()
	}

	if *serveDashboard {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			mux := http.NewServeMux()
			serveAPI(db, mux)
			mux.Handle("/", http.FileServer(http.Dir("web-dashboard-static")))
			server := &http.Server{Addr: ":8080", Handler: mux}
			logInfo("Web dashboard served at http://localhost:8080")
			go func() {
				<-ctx.Done()
				logInfo("Shutting down dashboard server...")
				server.Shutdown(context.Background())
			}()
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logError("Dashboard server error: %v", err)
			}
		}()
	}

	scanner := bufio.NewScanner(serialPort)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Graceful shutdown requested. Exiting...")
			waitGroup.Wait()
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

				const serialRetryDelay = 500 * time.Millisecond
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

			printToConsole(measurement, &latestWeather)
		}
	}
}
