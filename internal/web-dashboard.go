package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Result struct {
	AggregatedTimestamp int64
	AvgTemperature      float64
	AvgHumidity         float64
	City                string
	AvgWeatherTemp      float64
	AvgWeatherHumidity  float64
	AvgWindSpeed        float64
	AvgWindDeg          float64
	AvgClouds           float64
	AvgWeatherCode      float64
	Description         string
}

var startDashboardServer = startDashboardServerImpl

func startDashboardServerImpl(ctx context.Context, db *sql.DB, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		mux := http.NewServeMux()

		var stdDB *sql.DB = db
		gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
			Conn: stdDB, // reuse existing connection
		}), &gorm.Config{})
		if err != nil {
			log.Fatal(err)
		}

		serveAPI(gormDB, mux)
		mux.Handle("/", http.FileServer(http.Dir("skogsnet-frontend/dist")))
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

func serveAPI(db *gorm.DB, mux *http.ServeMux) {
	mux.HandleFunc("/api/measurements/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		derivativeLastMeasurementCount := 10

		var results []Result
		err := db.Model(&Measurement{}).
			Select(`measurements.timestamp AS aggregated_timestamp,
            measurements.temperature AS avg_temperature,
            measurements.humidity AS avg_humidity,
            weather.city AS city,
            weather.temp AS avg_weather_temp,
            weather.humidity AS avg_weather_humidity,
            weather.wind_speed AS avg_wind_speed,
            weather.wind_deg AS avg_wind_deg,
            weather.clouds AS avg_clouds,
            weather.weather_code AS avg_weather_code,
            weather.description AS description`).
			Joins("LEFT JOIN weather ON measurements.weather_id = weather.id").
			Order("measurements.timestamp DESC").
			Limit(derivativeLastMeasurementCount).
			Scan(&results).Error
		if err != nil || len(results) == 0 {
			http.Error(w, "DB query error", 500)
			logError("DB query error: %v", err)
			return
		}

		// Calculate trajectory (delta over last lastMeasurementCount measurements)
		var tempTrajectory *float64
		if len(results) >= 2 {
			diff := results[0].AvgTemperature - results[len(results)-1].AvgTemperature
			tempTrajectory = &diff
		}

		response := map[string]any{
			"latest":     results[0],
			"trajectory": tempTrajectory,
		}

		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/measurements", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		rangeParam := r.URL.Query().Get("range")
		now := time.Now()
		var since int64

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
			since = 0
		}

		// Set interval
		var intervalSeconds int
		switch rangeParam {
		case "1h":
			intervalSeconds = 60
		case "6h":
			intervalSeconds = 60
		case "12h", "24h", "today":
			intervalSeconds = 60
		case "week":
			intervalSeconds = 3600
		case "month", "year":
			intervalSeconds = 86400
		default:
			intervalSeconds = 86400
		}

		end := now.UnixMilli()

		var results []Result

		if rangeParam == "week" || rangeParam == "month" || rangeParam == "year" {
			// Daily bucket
			err := db.Model(&Measurement{}).
				Select(`CAST((measurements.timestamp / 1000) / 86400 AS INTEGER) * 86400 * 1000 AS aggregated_timestamp,
                AVG(measurements.temperature) AS avg_temperature,
                AVG(measurements.humidity) AS avg_humidity,
                MAX(weather.city) AS city,
                AVG(weather.temp) AS avg_weather_temp,
                AVG(weather.humidity) AS avg_weather_humidity,
                AVG(weather.wind_speed) AS avg_wind_speed,
                AVG(weather.wind_deg) AS avg_wind_deg,
                AVG(weather.clouds) AS avg_clouds,
                AVG(weather.weather_code) AS avg_weather_code,
                MAX(weather.description) AS description`).
				Joins("LEFT JOIN weather ON measurements.weather_id = weather.id").
				Where("measurements.timestamp >= ? AND measurements.timestamp <= ?", since, end).
				Group("aggregated_timestamp").
				Having("COUNT(temperature) > 0").
				Scan(&results).Error
			if err != nil {
				http.Error(w, "DB query error", 500)
				logError("DB query error: %v", err)
				return
			}
		} else {
			// Flexible bucket using intervalSeconds
			err := db.Model(&Measurement{}).
				Select(`(strftime('%s', datetime(measurements.timestamp / 1000, 'unixepoch')) / ? ) * ? * 1000 AS aggregated_timestamp,
                AVG(measurements.temperature) AS avg_temperature,
                AVG(measurements.humidity) AS avg_humidity,
                MAX(weather.city) AS city,
                AVG(weather.temp) AS avg_weather_temp,
                AVG(weather.humidity) AS avg_weather_humidity,
                AVG(weather.wind_speed) AS avg_wind_speed,
                AVG(weather.wind_deg) AS avg_wind_deg,
                AVG(weather.clouds) AS avg_clouds,
                AVG(weather.weather_code) AS avg_weather_code,
                MAX(weather.description) AS description`, intervalSeconds, intervalSeconds).
				Joins("LEFT JOIN weather ON measurements.weather_id = weather.id").
				Where("measurements.timestamp >= ? AND measurements.timestamp <= ?", since, end).
				Group("aggregated_timestamp").
				Having("COUNT(temperature) > 0").
				Scan(&results).Error
			if err != nil {
				http.Error(w, "DB query error", 500)
				logError("DB query error: %v", err)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
}
