package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createMeasurementTable := `
    CREATE TABLE IF NOT EXISTS measurements (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
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

func insertMeasurement(db *sql.DB, m Measurement, timestamp int64) error {
	if db == nil {
		return errors.New("db is nil")
	}
	_, err := db.Exec(
		"INSERT INTO measurements (timestamp, temperature, humidity) VALUES (?, ?, ?)",
		timestamp, m.TemperatureCelsius, m.HumidityPercentage,
	)
	return err
}

func insertWeather(db *sql.DB, w Weather, timestamp int64) error {
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

	// Write header
	header := "timestamp,temperature,humidity,city,weather_temp,weather_humidity,wind_speed,wind_deg,clouds,weather_code,weather_description\n"
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	rows, err := db.Query(`
        SELECT m.timestamp, m.temperature, m.humidity,
               w.city, w.temp, w.humidity, w.wind_speed, w.wind_deg, w.clouds, w.weather_code, w.description
        FROM measurements m
        LEFT JOIN weather w ON m.timestamp = w.timestamp
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
			nullString(city),
			nullFloat1(wTemp),
			nullInt(wHum),
			nullFloat1(windSpeed),
			nullInt(windDeg),
			nullInt(clouds),
			nullInt(weatherCode),
			nullString(description),
		)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
func nullFloat1(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return float64(int(nf.Float64*10)) / 10 // one decimal
	}
	return 0
}
func nullInt(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}
