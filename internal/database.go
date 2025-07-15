package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var mustInitDatabase = mustInitDatabaseImpl
var openDatabase = openDatabaseImpl
var insertWeather = insertWeatherImpl
var insertMeasurement = insertMeasurementImpl
var exportCSVAndExit = exportCSVAndExitImpl

func mustInitDatabaseImpl(dbFileName *string) (*sql.DB, error) {
	db, err := openDatabase(*dbFileName)
	if err != nil {
		logError("Failed to open database: %v", err)
		return nil, err
	}

	return db, nil
}

func enableWALMode(db *sql.DB) {
	if db == nil {
		logError("Database connection is nil")
		osExit(1)
		return
	}

	_, err := db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		logError("Failed to enable WAL mode: %v", err)
		osExit(1)
		return
	}

	var mode string
	row := db.QueryRow("PRAGMA journal_mode")
	if err := row.Scan(&mode); err != nil || mode != "wal" {
		logError("WAL mode verification failed: %v, mode=%s", err, mode)
		osExit(1)
		return
	}

	logInfo("WAL mode enabled")
}

func exportCSVAndExitImpl(dbFileName *string, exportCSV *string) {
	db, err := openDatabase(*dbFileName)
	if err != nil {
		logError("Failed to open database: %v", err)
		osExit(1)
		return
	}
	defer db.Close()
	if err := exportToCSV(db, *exportCSV); err != nil {
		logError("Export to CSV failed: %v", err)
		osExit(1)
		return
	} else {
		logInfo("Exported measurements to %s", *exportCSV)
	}
}

func openDatabaseImpl(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createMeasurementTable := `
	CREATE TABLE IF NOT EXISTS measurements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		weather_id INTEGER,
		timestamp INTEGER,
		temperature REAL,
		humidity REAL
	);`

	createWeatherTable := `
	CREATE TABLE IF NOT EXISTS weather (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER,
		city TEXT,
		temp REAL,
		humidity INTEGER,
		wind_speed REAL,
		wind_deg INTEGER,
		clouds INTEGER,
		weather_code INTEGER,
		description TEXT
	);`

	_, err = db.Exec(createMeasurementTable)
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(createWeatherTable)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func insertMeasurementImpl(db *sql.DB, m Measurement, timestamp int64) error {
	if db == nil {
		return errors.New("db is nil")
	}

	// Find nearest weather record within 10 minutes
	const weatherMatchWindowMillis = 600_000
	var weatherID sql.NullInt64

	err := db.QueryRow(`
		SELECT id FROM weather
		WHERE ABS(timestamp - ?) < ?
		ORDER BY ABS(timestamp - ?) ASC
		LIMIT 1
	`, timestamp, weatherMatchWindowMillis, timestamp).Scan(&weatherID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO measurements (timestamp, temperature, humidity, weather_id) VALUES (?, ?, ?, ?)",
		timestamp, m.TemperatureCelsius, m.HumidityPercentage, func() int64 {
			if weatherID.Valid {
				return weatherID.Int64
			} else {
				return 0
			}
		}(),
	)
	return err
}

func insertWeatherImpl(db *sql.DB, w Weather, timestamp int64) error {
	if db == nil {
		return errors.New("db is nil")
	}

	var weatherID int
	var weatherDesc string
	if len(w.Weather) > 0 {
		weatherID = w.Weather[0].ID
		weatherDesc = w.Weather[0].Description
	} else {
		weatherID = 0
		weatherDesc = ""
	}

	_, err := db.Exec(
		`INSERT INTO weather (timestamp, city, temp, humidity, wind_speed, wind_deg, clouds, weather_code, description)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		timestamp,
		w.Name,
		w.Main.Temp,
		w.Main.Humidity,
		w.Wind.Speed,
		w.Wind.Deg,
		w.Clouds.All,
		weatherID,
		weatherDesc,
	)

	return err
}

func exportToCSV(db *sql.DB, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fields := []string{
		"timestamp",
		"temperature",
		"humidity",
		"city",
		"weather_temp",
		"weather_humidity",
		"wind_speed",
		"wind_deg",
		"clouds",
		"weather_code",
		"weather_description",
	}

	header := strings.Join(fields, ",") + "\n"
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	rows, err := db.Query(`
		SELECT m.timestamp, m.temperature, m.humidity,
			w.city, w.temp, w.humidity, w.wind_speed, w.wind_deg, w.clouds, w.weather_code, w.description
		FROM measurements m
		LEFT JOIN weather w ON m.weather_id = w.id
		ORDER BY m.timestamp ASC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var ts int64
		var temp, hum float64
		var city sql.NullString
		var wTemp sql.NullFloat64
		var wHum sql.NullInt64
		var windSpeed sql.NullFloat64
		var windDeg sql.NullInt64
		var clouds sql.NullInt64
		var weatherCode sql.NullInt64
		var description sql.NullString

		if err := rows.Scan(&ts, &temp, &hum, &city, &wTemp, &wHum, &windSpeed, &windDeg, &clouds, &weatherCode, &description); err != nil {
			return err
		}

		// Format floats with one decimal, ints as is, empty string for NULLs
		line := fmt.Sprintf("%d,%.1f,%.1f,%s,%.1f,%d,%.1f,%d,%d,%d,%s\n",
			ts,
			temp,
			hum,
			func() string {
				if city.Valid {
					return city.String
				} else {
					return ""
				}
			}(),
			func() float64 {
				if wTemp.Valid {
					return float64(int(wTemp.Float64*10)) / 10
				} else {
					return 0
				}
			}(),
			func() int64 {
				if wHum.Valid {
					return wHum.Int64
				} else {
					return 0
				}
			}(),
			func() float64 {
				if windSpeed.Valid {
					return float64(int(windSpeed.Float64*10)) / 10
				} else {
					return 0
				}
			}(),
			func() int64 {
				if windDeg.Valid {
					return windDeg.Int64
				} else {
					return 0
				}
			}(),
			func() int64 {
				if clouds.Valid {
					return clouds.Int64
				} else {
					return 0
				}
			}(),
			func() int64 {
				if weatherCode.Valid {
					return weatherCode.Int64
				} else {
					return 0
				}
			}(),
			func() string {
				if description.Valid {
					return description.String
				} else {
					return ""
				}
			}(),
		)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}
