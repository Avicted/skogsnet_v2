package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func serveAPI(db *sql.DB) {
	http.HandleFunc("/api/measurements", func(w http.ResponseWriter, r *http.Request) {
		rangeParam := r.URL.Query().Get("range")
		var since int64
		now := time.Now()

		switch rangeParam {
		case "1h":
			since = now.Add(-1 * time.Hour).UnixMilli()
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

		rows, err := db.Query("SELECT timestamp, temperature, humidity FROM measurements WHERE timestamp >= ? ORDER BY timestamp", since)
		if err != nil {
			http.Error(w, "DB error", 500)
			return
		}
		defer rows.Close()

		var data []Measurement
		for rows.Next() {
			var m Measurement
			if err := rows.Scan(&m.UnixTimestamp, &m.TemperatureCelsius, &m.HumidityPercentage); err == nil {
				data = append(data, m)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})
}
