// Package weather provides weather API client functionality using WeatherAPI.com
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Client handles communication with WeatherAPI.com
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new weather API client
// API key is read from WEATHER_API_KEY environment variable
func NewClient() *Client {
	return NewClientWithKey(os.Getenv("WEATHER_API_KEY"))
}

// NewClientWithKey creates a new weather API client with a specific API key
// Useful for testing and custom configurations
func NewClientWithKey(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.weatherapi.com/v1",
	}
}

// GetCurrentWeather retrieves current weather for a location
func (c *Client) GetCurrentWeather(ctx context.Context, location string) (*CurrentWeatherResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no",
		c.baseURL,
		url.QueryEscape(c.apiKey),
		url.QueryEscape(location),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("weather API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var weatherResp CurrentWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &weatherResp, nil
}

// GetForecast retrieves weather forecast for a location
// days parameter specifies number of days (1-10)
func (c *Client) GetForecast(ctx context.Context, location string, days int) (*ForecastWeatherResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}

	// Validate days parameter (WeatherAPI supports 1-10 days for free tier)
	if days < 1 {
		days = 1
	}
	if days > 10 {
		days = 10
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/forecast.json?key=%s&q=%s&days=%d&aqi=no&alerts=no",
		c.baseURL,
		url.QueryEscape(c.apiKey),
		url.QueryEscape(location),
		days,
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("weather API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var forecastResp ForecastWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &forecastResp, nil
}
