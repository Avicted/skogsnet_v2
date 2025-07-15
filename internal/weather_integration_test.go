package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestGetCityLatLongImpl_RealAPI(t *testing.T) {
	city := "Helsinki"
	resp, err := getCityLatLongImpl(city)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("Expected results for city %s, got none", city)
	}
	result := resp.Results[0]
	if result.Name != city && result.Name != "Helsingfors" {
		t.Errorf("Expected city name %s or Helsingfors, got %s", city, result.Name)
	}
	if result.Latitude == 0 || result.Longitude == 0 {
		t.Error("Expected valid latitude and longitude")
	}
}

func TestGetWeatherDataImpl_RealAPI(t *testing.T) {
	city := "Helsinki"
	weather, err := getWeatherDataImpl(city)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if weather.Name != city && weather.Name != "Helsingfors" {
		t.Errorf("Expected city name %s or Helsingfors, got %s", city, weather.Name)
	}
	if weather.Main.Temp == 0 {
		t.Error("Expected valid temperature")
	}
	if weather.Wind.Speed == 0 {
		t.Error("Expected valid wind speed")
	}
	if weather.Clouds.All < 0 {
		t.Error("Expected valid cloud cover percentage")
	}
	if weather.Main.Humidity < 0 {
		t.Error("Expected valid humidity percentage")
	}
}

func TestGetWeatherDataImplRealAPI_ErrorCases(t *testing.T) {
	originalClient := httpClient
	defer func() { httpClient = originalClient }()

	// 1. Network error from GetCityLatLong (first call)
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		}),
	}
	_, err := getWeatherDataImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "network error") {
		t.Errorf("Expected network error from GetCityLatLong, got %v", err)
	}

	// 2. Non-200 HTTP status from GetCityLatLong (first call)
	step := 0
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if step == 0 {
				step++
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Status:     "403 Forbidden",
					Body:       ioutil.NopCloser(bytes.NewBufferString("Forbidden")),
				}, nil
			}
			// Should not reach here for this test
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			}, nil
		}),
	}
	_, err = getWeatherDataImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "failed to get data") {
		t.Errorf("Expected HTTP status error from GetCityLatLong, got %v", err)
	}

	// 3. No results found for city from GetCityLatLong (first call)
	step = 0
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if step == 0 {
				step++
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewBufferString(`{"results":[]}`)),
				}, nil
			}
			// Should not reach here for this test
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
			}, nil
		}),
	}
	_, err = getWeatherDataImpl("NoCity")
	expected := fmt.Sprintf("no results found for city: %s", "NoCity")
	if err == nil || err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%v'", expected, err)
	}

	// 4. Network error from weather API (second call)
	step = 0
	cityResponse := `{"results":[{"name":"Helsinki","latitude":60.0,"longitude":25.0}]}`
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if step == 0 {
				step++
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewBufferString(cityResponse)),
				}, nil
			}
			return nil, fmt.Errorf("weather api network error")
		}),
	}
	_, err = getWeatherDataImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "weather api network error") {
		t.Errorf("Expected network error from weather API, got %v", err)
	}
}

func TestGetWeatherDataImpl_WeatherAPIStatusError(t *testing.T) {
	originalClient := httpClient
	defer func() { httpClient = originalClient }()

	// First call: geocoding returns valid city
	// Second call: weather API returns 403 Forbidden
	step := 0
	cityResponse := `{"results":[{"name":"Helsinki","latitude":60.0,"longitude":25.0}]}`
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if step == 0 {
				step++
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewBufferString(cityResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       ioutil.NopCloser(bytes.NewBufferString("Forbidden")),
			}, nil
		}),
	}
	_, err := getWeatherDataImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "failed to get weather data") {
		t.Errorf("Expected HTTP status error from weather API, got %v", err)
	}
}

func TestGetWeatherDataImpl_WeatherAPIDecodeError(t *testing.T) {
	originalClient := httpClient
	defer func() { httpClient = originalClient }()

	// First call: geocoding returns valid city
	// Second call: weather API returns invalid JSON
	step := 0
	cityResponse := `{"results":[{"name":"Helsinki","latitude":60.0,"longitude":25.0}]}`
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if step == 0 {
				step++
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewBufferString(cityResponse)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString("{invalid json")),
			}, nil
		}),
	}
	_, err := getWeatherDataImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "failed to decode weather data") {
		t.Errorf("Expected decode error from weather API, got %v", err)
	}
}

func TestGetCityLatLongImplRealAPI_InvalidCity(t *testing.T) {
	city := "InvalidCityName12345"
	_, err := getCityLatLongImpl(city)
	if err == nil {
		t.Fatalf("Expected error for invalid city, got none")
	}
	expected := fmt.Sprintf("no results found for city: %s", city)
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestGetCityLatLongImplRealAPI_ErrorCases(t *testing.T) {
	originalClient := httpClient
	defer func() { httpClient = originalClient }()

	// 1. Network error (simulate by returning error from RoundTrip)
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		}),
	}
	_, err := getCityLatLongImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "network error") {
		t.Errorf("Expected network error, got %v", err)
	}

	// 2. Non-200 HTTP status
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Body:       ioutil.NopCloser(bytes.NewBufferString("Forbidden")),
			}
			return resp, nil
		}),
	}
	_, err = getCityLatLongImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "failed to get data") {
		t.Errorf("Expected HTTP status error, got %v", err)
	}

	// 3. No results found for city
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"results":[]}`)),
			}
			return resp, nil
		}),
	}
	city := "NoCity"
	_, err = getCityLatLongImpl(city)
	expected := fmt.Sprintf("no results found for city: %s", city)
	if err == nil || err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%v'", expected, err)
	}

	// 4. Decoding error
	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{invalid json`)),
			}
			return resp, nil
		}),
	}
	_, err = getCityLatLongImpl("Helsinki")
	if err == nil || !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("Expected decode error, got %v", err)
	}
}
