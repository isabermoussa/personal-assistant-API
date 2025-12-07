package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/isabermoussa/personal-assistant-API/internal/chat/assistant/weather"
	"github.com/openai/openai-go/v2"
)

// WeatherTool provides current weather and forecast information
type WeatherTool struct {
	client *weather.Client
}

// NewWeatherTool creates a new weather tool with the provided weather client
func NewWeatherTool(client *weather.Client) *WeatherTool {
	return &WeatherTool{
		client: client,
	}
}

func (t *WeatherTool) Name() string {
	return "get_weather"
}

func (t *WeatherTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_weather",
		Description: openai.String("Get current weather or multi-day forecast for a given location. Use forecast_days for future weather predictions (1-10 days)."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]string{
					"type":        "string",
					"description": "City name, coordinates (lat,lon), or location query (e.g., 'Barcelona', 'Paris, France', '48.8567,2.3508')",
				},
				"forecast_days": map[string]any{
					"type":        "integer",
					"description": "Number of days of forecast (1-10). Omit or set to 0 for current weather only. Use this when user asks about future weather or multi-day forecasts.",
					"minimum":     1,
					"maximum":     10,
				},
			},
			"required": []string{"location"},
		},
	})
}

func (t *WeatherTool) Handle(ctx context.Context, args string) (string, error) {
	var params struct {
		Location     string `json:"location"`
		ForecastDays int    `json:"forecast_days,omitempty"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid weather parameters: %w", err)
	}

	// Get forecast if requested
	if params.ForecastDays > 0 {
		forecast, err := t.client.GetForecast(ctx, params.Location, params.ForecastDays)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to fetch weather forecast",
				"error", err,
				"location", params.Location,
				"days", params.ForecastDays,
			)
			return "", fmt.Errorf("failed to fetch weather forecast: %w", err)
		}
		return weather.FormatForecast(forecast), nil
	}

	// Get current weather
	w, err := t.client.GetCurrentWeather(ctx, params.Location)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch current weather",
			"error", err,
			"location", params.Location,
		)
		return "", fmt.Errorf("failed to fetch weather: %w", err)
	}

	return weather.FormatCurrentWeather(w), nil
}
