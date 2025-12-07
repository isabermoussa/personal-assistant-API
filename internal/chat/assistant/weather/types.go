package weather

import "fmt"

// CurrentWeatherResponse represents the response from WeatherAPI current weather endpoint
type CurrentWeatherResponse struct {
	Location struct {
		Name      string  `json:"name"`
		Region    string  `json:"region"`
		Country   string  `json:"country"`
		Lat       float64 `json:"lat"`
		Lon       float64 `json:"lon"`
		LocalTime string  `json:"localtime"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
		} `json:"condition"`
		WindKph    float64 `json:"wind_kph"`
		WindMph    float64 `json:"wind_mph"`
		WindDir    string  `json:"wind_dir"`
		Humidity   int     `json:"humidity"`
		FeelsLikeC float64 `json:"feelslike_c"`
		FeelsLikeF float64 `json:"feelslike_f"`
		VisKm      float64 `json:"vis_km"`
		UV         float64 `json:"uv"`
	} `json:"current"`
}

// ForecastWeatherResponse represents the response from WeatherAPI forecast endpoint
type ForecastWeatherResponse struct {
	Location struct {
		Name      string `json:"name"`
		Region    string `json:"region"`
		Country   string `json:"country"`
		LocalTime string `json:"localtime"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
		WindKph  float64 `json:"wind_kph"`
		Humidity int     `json:"humidity"`
	} `json:"current"`
	Forecast struct {
		ForecastDay []struct {
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
		} `json:"forecastday"`
	} `json:"forecast"`
}

// FormatCurrentWeather formats current weather data into a human-readable string
func FormatCurrentWeather(w *CurrentWeatherResponse) string {
	return fmt.Sprintf(
		"Current weather in %s, %s:\n"+
			"Condition: %s\n"+
			"Temperature: %.1f°C (%.1f°F)\n"+
			"Feels like: %.1f°C (%.1f°F)\n"+
			"Wind: %.1f km/h %s\n"+
			"Humidity: %d%%\n"+
			"Visibility: %.1f km\n"+
			"UV Index: %.1f\n"+
			"Local time: %s",
		w.Location.Name,
		w.Location.Country,
		w.Current.Condition.Text,
		w.Current.TempC,
		w.Current.TempF,
		w.Current.FeelsLikeC,
		w.Current.FeelsLikeF,
		w.Current.WindKph,
		w.Current.WindDir,
		w.Current.Humidity,
		w.Current.VisKm,
		w.Current.UV,
		w.Location.LocalTime,
	)
}

// FormatForecast formats forecast data into a human-readable string
func FormatForecast(f *ForecastWeatherResponse) string {
	result := fmt.Sprintf("Weather forecast for %s, %s:\n\n", f.Location.Name, f.Location.Country)

	// Add current weather
	result += fmt.Sprintf("Current: %s, %.1f°C, Wind: %.1f km/h, Humidity: %d%%\n\n",
		f.Current.Condition.Text,
		f.Current.TempC,
		f.Current.WindKph,
		f.Current.Humidity,
	)

	// Add forecast days
	for i, day := range f.Forecast.ForecastDay {
		result += fmt.Sprintf("Day %d (%s):\n", i+1, day.Date)
		result += fmt.Sprintf("  Condition: %s\n", day.Day.Condition.Text)
		result += fmt.Sprintf("  Temperature: %.1f°C to %.1f°C (avg: %.1f°C)\n",
			day.Day.MinTempC,
			day.Day.MaxTempC,
			day.Day.AvgTempC,
		)
		result += fmt.Sprintf("  Max wind: %.1f km/h\n", day.Day.MaxWindKph)
		result += fmt.Sprintf("  Avg humidity: %.0f%%\n", day.Day.AvgHumidity)
		result += fmt.Sprintf("  Chance of rain: %d%%\n", day.Day.ChanceOfRain)
		if i < len(f.Forecast.ForecastDay)-1 {
			result += "\n"
		}
	}

	return result
}
