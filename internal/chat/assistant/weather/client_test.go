package weather

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestWeatherClient_GetCurrentWeather(t *testing.T) {
	// Skip if no API key is set
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping weather API test: WEATHER_API_KEY not set")
	}

	client := NewClient()
	ctx := context.Background()

	t.Run("gets current weather for Barcelona", func(t *testing.T) {
		weather, err := client.GetCurrentWeather(ctx, "Barcelona")
		if err != nil {
			t.Fatalf("GetCurrentWeather failed: %v", err)
		}

		// Validate response structure
		if weather.Location.Name == "" {
			t.Error("Expected location name, got empty string")
		}

		if weather.Location.Country == "" {
			t.Error("Expected country, got empty string")
		}

		if weather.Current.Condition.Text == "" {
			t.Error("Expected condition text, got empty string")
		}

		// Temperature should be in reasonable range (-50 to 60°C)
		if weather.Current.TempC < -50 || weather.Current.TempC > 60 {
			t.Errorf("Temperature seems unreasonable: %.1f°C", weather.Current.TempC)
		}

		t.Logf("Weather in %s: %s, %.1f°C",
			weather.Location.Name,
			weather.Current.Condition.Text,
			weather.Current.TempC,
		)
	})

	t.Run("handles invalid location", func(t *testing.T) {
		_, err := client.GetCurrentWeather(ctx, "InvalidLocationNameThatDoesNotExist123456")
		if err == nil {
			t.Error("Expected error for invalid location, got nil")
		}

		if !strings.Contains(err.Error(), "weather API returned status") {
			t.Logf("Error message: %v", err)
		}
	})

	t.Run("supports coordinates", func(t *testing.T) {
		// Barcelona coordinates
		weather, err := client.GetCurrentWeather(ctx, "41.3874,2.1686")
		if err != nil {
			t.Fatalf("GetCurrentWeather with coordinates failed: %v", err)
		}

		if weather.Location.Name == "" {
			t.Error("Expected location name from coordinates")
		}

		t.Logf("Location from coordinates: %s", weather.Location.Name)
	})
}

func TestWeatherClient_GetForecast(t *testing.T) {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping weather API test: WEATHER_API_KEY not set")
	}

	client := NewClient()
	ctx := context.Background()

	t.Run("gets 3-day forecast for Barcelona", func(t *testing.T) {
		forecast, err := client.GetForecast(ctx, "Barcelona", 3)
		if err != nil {
			t.Fatalf("GetForecast failed: %v", err)
		}

		// Validate response
		if forecast.Location.Name == "" {
			t.Error("Expected location name, got empty string")
		}

		if len(forecast.Forecast.ForecastDay) != 3 {
			t.Errorf("Expected 3 forecast days, got %d", len(forecast.Forecast.ForecastDay))
		}

		// Check each forecast day
		for i, day := range forecast.Forecast.ForecastDay {
			if day.Date == "" {
				t.Errorf("Day %d: expected date, got empty string", i)
			}
			if day.Day.Condition.Text == "" {
				t.Errorf("Day %d: expected condition text, got empty string", i)
			}

			t.Logf("Day %d (%s): %s, %.1f-%.1f°C",
				i+1,
				day.Date,
				day.Day.Condition.Text,
				day.Day.MinTempC,
				day.Day.MaxTempC,
			)
		}
	})

	t.Run("limits forecast to 10 days maximum", func(t *testing.T) {
		// Request 15 days, should be capped at 10
		forecast, err := client.GetForecast(ctx, "London", 15)
		if err != nil {
			t.Fatalf("GetForecast failed: %v", err)
		}

		if len(forecast.Forecast.ForecastDay) > 10 {
			t.Errorf("Expected maximum 10 forecast days, got %d", len(forecast.Forecast.ForecastDay))
		}

		t.Logf("Requested 15 days, got %d days (correctly capped)", len(forecast.Forecast.ForecastDay))
	})

	t.Run("defaults to 1 day if invalid days parameter", func(t *testing.T) {
		forecast, err := client.GetForecast(ctx, "Paris", 0)
		if err != nil {
			t.Fatalf("GetForecast failed: %v", err)
		}

		if len(forecast.Forecast.ForecastDay) < 1 {
			t.Error("Expected at least 1 forecast day")
		}

		t.Logf("Requested 0 days, got %d day(s)", len(forecast.Forecast.ForecastDay))
	})
}

func TestFormatCurrentWeather(t *testing.T) {
	// Create sample weather data
	weather := &CurrentWeatherResponse{}
	weather.Location.Name = "Barcelona"
	weather.Location.Country = "Spain"
	weather.Location.LocalTime = "2025-12-07 15:30"
	weather.Current.TempC = 18.5
	weather.Current.TempF = 65.3
	weather.Current.FeelsLikeC = 17.2
	weather.Current.FeelsLikeF = 63.0
	weather.Current.Condition.Text = "Partly cloudy"
	weather.Current.WindKph = 15.8
	weather.Current.WindDir = "NW"
	weather.Current.Humidity = 65
	weather.Current.VisKm = 10.0
	weather.Current.UV = 4.0

	formatted := FormatCurrentWeather(weather)

	// Check that all key information is included
	requiredFields := []string{
		"Barcelona", "Spain", "Partly cloudy",
		"18.5°C", "65.3°F",
		"15.8 km/h", "NW",
		"65%",
		"10.0 km",
	}

	for _, field := range requiredFields {
		if !strings.Contains(formatted, field) {
			t.Errorf("Formatted weather missing expected field: %s", field)
		}
	}

	t.Logf("Formatted weather:\n%s", formatted)
}

func TestFormatForecast(t *testing.T) {
	// Create sample forecast data
	forecast := &ForecastWeatherResponse{}
	forecast.Location.Name = "Barcelona"
	forecast.Location.Country = "Spain"
	forecast.Current.TempC = 18.5
	forecast.Current.Condition.Text = "Sunny"
	forecast.Current.WindKph = 10.0
	forecast.Current.Humidity = 60

	// Add 2 forecast days
	forecast.Forecast.ForecastDay = make([]struct {
		Date string `json:"date"`
		Day  struct {
			MaxTempC  float64 `json:"maxtemp_c"`
			MinTempC  float64 `json:"mintemp_c"`
			AvgTempC  float64 `json:"avgtemp_c"`
			Condition struct {
				Text string `json:"text"`
			} `json:"condition"`
			MaxWindKph   float64 `json:"maxwind_kph"`
			AvgHumidity  float64 `json:"avghumidity"`
			ChanceOfRain int     `json:"daily_chance_of_rain"`
		} `json:"day"`
	}, 2)

	forecast.Forecast.ForecastDay[0].Date = "2025-12-08"
	forecast.Forecast.ForecastDay[0].Day.MaxTempC = 22.0
	forecast.Forecast.ForecastDay[0].Day.MinTempC = 15.0
	forecast.Forecast.ForecastDay[0].Day.AvgTempC = 18.5
	forecast.Forecast.ForecastDay[0].Day.Condition.Text = "Sunny"
	forecast.Forecast.ForecastDay[0].Day.MaxWindKph = 20.0
	forecast.Forecast.ForecastDay[0].Day.AvgHumidity = 55.0
	forecast.Forecast.ForecastDay[0].Day.ChanceOfRain = 10

	forecast.Forecast.ForecastDay[1].Date = "2025-12-09"
	forecast.Forecast.ForecastDay[1].Day.MaxTempC = 20.0
	forecast.Forecast.ForecastDay[1].Day.MinTempC = 14.0
	forecast.Forecast.ForecastDay[1].Day.AvgTempC = 17.0
	forecast.Forecast.ForecastDay[1].Day.Condition.Text = "Partly cloudy"
	forecast.Forecast.ForecastDay[1].Day.MaxWindKph = 25.0
	forecast.Forecast.ForecastDay[1].Day.AvgHumidity = 60.0
	forecast.Forecast.ForecastDay[1].Day.ChanceOfRain = 30

	formatted := FormatForecast(forecast)

	// Check structure
	requiredFields := []string{
		"Barcelona", "Spain",
		"Current:", "18.5°C",
		"Day 1", "2025-12-08", "Sunny",
		"Day 2", "2025-12-09", "Partly cloudy",
		"22.0°C", "15.0°C",
		"Chance of rain: 10%",
		"Chance of rain: 30%",
	}

	for _, field := range requiredFields {
		if !strings.Contains(formatted, field) {
			t.Errorf("Formatted forecast missing expected field: %s", field)
		}
	}

	t.Logf("Formatted forecast:\n%s", formatted)
}

func TestWeatherClient_NoAPIKey(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("WEATHER_API_KEY")
	os.Setenv("WEATHER_API_KEY", "")
	defer os.Setenv("WEATHER_API_KEY", originalKey)

	client := NewClient()
	ctx := context.Background()

	t.Run("returns error when API key not set for current weather", func(t *testing.T) {
		_, err := client.GetCurrentWeather(ctx, "Barcelona")
		if err == nil {
			t.Error("Expected error when API key not set, got nil")
		}

		if !strings.Contains(err.Error(), "WEATHER_API_KEY") {
			t.Errorf("Expected error message about WEATHER_API_KEY, got: %v", err)
		}
	})

	t.Run("returns error when API key not set for forecast", func(t *testing.T) {
		_, err := client.GetForecast(ctx, "Barcelona", 3)
		if err == nil {
			t.Error("Expected error when API key not set, got nil")
		}

		if !strings.Contains(err.Error(), "WEATHER_API_KEY") {
			t.Errorf("Expected error message about WEATHER_API_KEY, got: %v", err)
		}
	})
}
