package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

func startDashboardServer(ctx context.Context, db *sql.DB, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
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

func serveAPI(db *sql.DB, mux *http.ServeMux) {
	mux.HandleFunc("/api/measurements", func(w http.ResponseWriter, r *http.Request) {
		rangeParam := r.URL.Query().Get("range")
		var since int64
		now := time.Now()

		switch rangeParam {
		case "1h":
			since = now.Add(-1 * time.Hour).UnixMilli()
		case "6h":
			since = now.Add(-6 * time.Hour).UnixMilli()
		case "12h":
			since = now.Add(-12 * time.Hour).UnixMilli()
		case "24h":
			since = now.Add(-24 * time.Hour).UnixMilli()
		case "today":
			since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
		case "week":
			since = now.AddDate(0, 0, -7).UnixMilli()
		case "month":
			since = now.AddDate(0, -1, 0).UnixMilli()
		case "year":
			since = now.AddDate(-1, 0, 0).UnixMilli()
		default:
			since = 0 // all data
		}

		const maxWeatherAgeMillis = int64(1 * 60 * 1000) // 1 minute

		rows, err := db.Query(`
			SELECT m.timestamp, m.temperature, m.humidity,
				w.city, w.temp, w.humidity, w.wind_speed, w.wind_deg, w.clouds, w.weather_code, w.description,
				w.timestamp as weather_ts
			FROM measurements m
			LEFT JOIN weather w ON m.weather_id = w.id
			WHERE m.timestamp >= ?
			ORDER BY m.timestamp
		`, since)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("DB error"))
			return
		}
		defer rows.Close()

		type Combined struct {
			Timestamp       int64   `json:"timestamp"`
			Temperature     float64 `json:"temperature"`
			Humidity        float64 `json:"humidity"`
			City            string  `json:"city"`
			WeatherTemp     float64 `json:"weather_temp"`
			WeatherHumidity int64   `json:"weather_humidity"`
			WindSpeed       float64 `json:"wind_speed"`
			WindDeg         int64   `json:"wind_deg"`
			Clouds          int64   `json:"clouds"`
			WeatherCode     int64   `json:"weather_code"`
			Description     string  `json:"weather_description"`
		}

		var data []Combined
		for rows.Next() {
			var c Combined
			var city, description sql.NullString
			var weatherTemp sql.NullFloat64
			var weatherHumidity, windDeg, clouds, weatherCode, weatherTs sql.NullInt64
			var windSpeed sql.NullFloat64

			if err := rows.Scan(
				&c.Timestamp, &c.Temperature, &c.Humidity,
				&city, &weatherTemp, &weatherHumidity, &windSpeed, &windDeg, &clouds, &weatherCode, &description,
				&weatherTs,
			); err == nil {
				// Only use weather data if it's not too old
				if weatherTs.Valid && (c.Timestamp-weatherTs.Int64) <= maxWeatherAgeMillis && (c.Timestamp-weatherTs.Int64) >= 0 {
					c.City = city.String
					c.WeatherTemp = weatherTemp.Float64
					c.WeatherHumidity = weatherHumidity.Int64
					c.WindSpeed = windSpeed.Float64
					c.WindDeg = windDeg.Int64
					c.Clouds = clouds.Int64
					c.WeatherCode = weatherCode.Int64
					c.Description = description.String
				} else {
					c.City = ""
					c.WeatherTemp = 0
					c.WeatherHumidity = 0
					c.WindSpeed = 0
					c.WindDeg = 0
					c.Clouds = 0
					c.WeatherCode = 0
					c.Description = ""
				}
				data = append(data, c)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})
}
