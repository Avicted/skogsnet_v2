package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type GeoResult struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GeoResponse struct {
	Results []GeoResult `json:"results"`
}

type OpenMeteoWeather struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Current   struct {
		Time               string  `json:"time"`
		Interval           int     `json:"interval"`
		Temperature2m      float64 `json:"temperature_2m"`
		WeatherCode        int     `json:"weather_code"`
		Precipitation      float64 `json:"precipitation"`
		RelativeHumidity2m int     `json:"relative_humidity_2m"`
		WindSpeed10m       float64 `json:"wind_speed_10m"`
		WindDirection10m   int     `json:"wind_direction_10m"`
	} `json:"current"`
}

type Weather struct {
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Rain struct {
		OneHour float64 `json:"1h"`
	} `json:"rain"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Name string `json:"name"`
}

func startWeatherFetcher(ctx context.Context, db *sql.DB, latestWeather *Weather, latestWeatherTimestamp *int64, wg *sync.WaitGroup) {
	city := *weatherCity
	if city == "" {
		logError("No city specified for weather data")
		os.Exit(1)
	}

	weatherTicker := time.NewTicker(weatherFetchInterval)
	defer weatherTicker.Stop()

weatherInit:
	for {
		select {
		case <-ctx.Done():
			logInfo("Weather fetching loop stopped")
			return
		default:
			w, err := GetWeatherData(city)
			if err == nil {
				*latestWeather = w
				*latestWeatherTimestamp = time.Now().UnixMilli()
				insertWeather(db, *latestWeather, *latestWeatherTimestamp)
				break weatherInit
			}
			logError("Initial weather fetch failed, retrying in 5s: %v", err)
			time.Sleep(weatherFetchRetryDelay)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				logInfo("Weather update goroutine stopped")
				return
			case <-weatherTicker.C:
				w, err := GetWeatherData(city)
				ts := time.Now().UnixMilli()
				if err == nil {
					*latestWeather = w
					*latestWeatherTimestamp = ts
					insertWeather(db, *latestWeather, *latestWeatherTimestamp)
				} else {
					throttledLogError(&lastWeatherErr, "Failed to get weather data for city %s: %v", city, err)
				}
			}
		}
	}()
}

// https://open-meteo.com/en/docs#weather_variable_documentation
func WeatherCodeToSentence(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45:
		return "Fog"
	case 48:
		return "Depositing rime fog"
	case 51:
		return "Light drizzle"
	case 53:
		return "Moderate drizzle"
	case 55:
		return "Dense drizzle"
	case 56:
		return "Light freezing drizzle"
	case 57:
		return "Dense freezing drizzle"
	case 61:
		return "Slight rain"
	case 63:
		return "Moderate rain"
	case 65:
		return "Heavy rain"
	case 66:
		return "Light freezing rain"
	case 67:
		return "Heavy freezing rain"
	case 71:
		return "Slight snow fall"
	case 73:
		return "Moderate snow fall"
	case 75:
		return "Heavy snow fall"
	case 77:
		return "Snow grains"
	case 80:
		return "Slight rain showers"
	case 81:
		return "Moderate rain showers"
	case 82:
		return "Violent rain showers"
	case 85:
		return "Slight snow showers"
	case 86:
		return "Heavy snow showers"
	case 95:
		return "Thunderstorm"
	case 96:
		return "Thunderstorm with slight hail"
	case 99:
		return "Thunderstorm with heavy hail"
	default:
		return "Unknown weather code"
	}
}

func WindDirectionToCompass(deg int) string {
	if deg < 0 || deg > 359 {
		return ""
	}

	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}

	// Each direction covers 45 degrees, centered on its midpoint
	// Offset by 22.5 to align ranges: N = 337.5-22.5, NE = 22.5-67.5, etc.
	idx := int((float64(deg)+22.5)/45.0) % 8
	return directions[idx]
}

func ConvertOpenMeteoToWeather(om OpenMeteoWeather, cityName string) Weather {
	return Weather{
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		}{
			{
				ID:          om.Current.WeatherCode,
				Main:        WeatherCodeToSentence(om.Current.WeatherCode),
				Description: WeatherCodeToSentence(om.Current.WeatherCode),
			},
		},
		Main: struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		}{
			Temp:     om.Current.Temperature2m,
			Humidity: om.Current.RelativeHumidity2m,
		},
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		}{
			Speed: om.Current.WindSpeed10m,
			Deg:   om.Current.WindDirection10m,
		},
		Rain: struct {
			OneHour float64 `json:"1h"`
		}{
			OneHour: om.Current.Precipitation,
		},
		Clouds: struct {
			All int `json:"all"`
		}{
			All: 0,
		},
		Name: cityName,
	}
}

func GetCityLatLong(city string) (GeoResponse, error) {
	response, err := http.Get("https://geocoding-api.open-meteo.com/v1/search?name=" + city + "&count=1")
	if err != nil {
		return GeoResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return GeoResponse{}, fmt.Errorf("failed to get data: %s", response.Status)
	}

	var geoResponse GeoResponse
	if err := json.NewDecoder(response.Body).Decode(&geoResponse); err != nil {
		return GeoResponse{}, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(geoResponse.Results) == 0 {
		return GeoResponse{}, fmt.Errorf("no results found for city: %s", city)
	}

	return geoResponse, nil
}

func GetWeatherData(city string) (Weather, error) {
	geoResponse, err := GetCityLatLong(city)
	if err != nil {
		return Weather{}, err
	}

	lat := geoResponse.Results[0].Latitude
	long := geoResponse.Results[0].Longitude

	response, err := http.Get(fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,weather_code,precipitation,relative_humidity_2m,wind_speed_10m,wind_direction_10m&wind_speed_unit=ms&temperature_unit=celsius", lat, long))
	if err != nil {
		return Weather{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return Weather{}, fmt.Errorf("failed to get weather data: %s", response.Status)
	}

	body, _ := io.ReadAll(response.Body)
	response.Body = io.NopCloser(bytes.NewReader(body))

	var openMeteoWeather OpenMeteoWeather
	if err := json.NewDecoder(response.Body).Decode(&openMeteoWeather); err != nil {
		return Weather{}, fmt.Errorf("failed to decode weather data: %v", err)
	}

	return ConvertOpenMeteoToWeather(openMeteoWeather, geoResponse.Results[0].Name), nil
}
