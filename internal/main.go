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

	var latestWeather Weather
	var latestWeatherTimestamp int64

	if *enableWeather {
		weatherTicker := time.NewTicker(1 * time.Minute)
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
				time.Sleep(5 * time.Second)
			}
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
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
		serveAPI(db)
		http.Handle("/", http.FileServer(http.Dir("web-dashboard-static")))
		go func() {
			addr := "http://localhost:8080"
			logInfo("Web dashboard served at %s", addr)
			fmt.Printf("Web dashboard served at %s\n", addr)
			if err := http.ListenAndServe(":8080", nil); err != nil {
				logError("Dashboard server error: %v", err)
				return
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

			currentTimestamp := time.Now().UnixMilli()
			if err := insertMeasurement(db, measurement, currentTimestamp); err != nil {
				throttledLogError(&lastInsertErr, "Failed to insert measurement into database: %v", err)
				continue
			}

			printToConsole(measurement, &latestWeather)
		}
	}
}
