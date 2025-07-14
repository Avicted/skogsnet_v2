package main

import (
	"testing"
)

func TestGetCityLatLong(t *testing.T) {
	city := "Helsingfors"
	geoResponse, err := GetCityLatLong(city)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(geoResponse.Results) == 0 {
		t.Fatalf("Expected results for city %s, got none", city)
	}

	result := geoResponse.Results[0]
	if result.Name != city {
		t.Errorf("Expected city name %s, got %s", city, result.Name)
	}
	if result.Latitude == 0 || result.Longitude == 0 {
		t.Error("Expected valid latitude and longitude")
	}
}

func TestGetCityLatLongInvalidCity(t *testing.T) {
	city := "InvalidCityName12345"
	_, err := GetCityLatLong(city)
	if err == nil {
		t.Fatalf("Expected error for invalid city, got none")
	}

	expectedError := "no results found for city: " + city
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetCityLatLongEmptyCity(t *testing.T) {
	_, err := GetCityLatLong("")
	if err == nil {
		t.Fatalf("Expected error for empty city name, got none")
	}

	expectedError := "no results found for city: "
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetWeatherData(t *testing.T) {
	city := "Helsingfors"
	weather, err := GetWeatherData(city)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if weather.Name != city {
		t.Errorf("Expected city name %s, got %s", city, weather.Name)
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

func TestWindDirectionToCompass(t *testing.T) {
	tests := []struct {
		degrees int
		compass string
	}{
		{0, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
	}

	for _, test := range tests {
		result := WindDirectionToCompass(test.degrees)
		if result != test.compass {
			t.Errorf("Expected %s for %d degrees, got %s", test.compass, test.degrees, result)
		}
	}
}

func TestWindDirectionToCompassInvalid(t *testing.T) {
	tests := []int{-1, 360, 400, -100}

	for _, deg := range tests {
		result := WindDirectionToCompass(deg)
		if result != "" {
			t.Errorf("Expected empty string for %d degrees, got %s", deg, result)
		}
	}
}
