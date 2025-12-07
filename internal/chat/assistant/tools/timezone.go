package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go/v2"
)

// TimeZoneTool converts times between different time zones
type TimeZoneTool struct{}

// NewTimeZoneTool creates a new time zone converter tool
func NewTimeZoneTool() *TimeZoneTool {
	return &TimeZoneTool{}
}

func (t *TimeZoneTool) Name() string {
	return "convert_timezone"
}

func (t *TimeZoneTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "convert_timezone",
		Description: openai.String("Convert a time from one timezone to another. Useful for travelers scheduling across different locations. Supports IANA timezone names (e.g., 'America/New_York', 'Europe/Madrid', 'Asia/Tokyo')."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"time": map[string]string{
					"type":        "string",
					"description": "Time in RFC3339 format (e.g., '2025-12-15T14:00:00Z') or 'now' for current time",
				},
				"from_timezone": map[string]string{
					"type":        "string",
					"description": "Source timezone in IANA format (e.g., 'America/New_York', 'Europe/Madrid', 'UTC')",
				},
				"to_timezone": map[string]string{
					"type":        "string",
					"description": "Target timezone in IANA format (e.g., 'America/New_York', 'Europe/Madrid', 'Asia/Tokyo')",
				},
			},
			"required": []string{"from_timezone", "to_timezone"},
		},
	})
}

func (t *TimeZoneTool) Handle(ctx context.Context, args string) (string, error) {
	var params struct {
		Time         string `json:"time"`
		FromTimezone string `json:"from_timezone"`
		ToTimezone   string `json:"to_timezone"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid timezone parameters: %w", err)
	}

	// Load timezones
	fromLoc, err := time.LoadLocation(params.FromTimezone)
	if err != nil {
		return "", fmt.Errorf("invalid source timezone '%s': %w", params.FromTimezone, err)
	}

	toLoc, err := time.LoadLocation(params.ToTimezone)
	if err != nil {
		return "", fmt.Errorf("invalid target timezone '%s': %w", params.ToTimezone, err)
	}

	// Parse input time or use current time
	var inputTime time.Time
	if params.Time == "" || params.Time == "now" {
		inputTime = time.Now().In(fromLoc)
	} else {
		parsedTime, err := time.Parse(time.RFC3339, params.Time)
		if err != nil {
			return "", fmt.Errorf("invalid time format '%s', expected RFC3339 (e.g., '2025-12-15T14:00:00Z'): %w", params.Time, err)
		}
		inputTime = parsedTime.In(fromLoc)
	}

	// Convert to target timezone
	convertedTime := inputTime.In(toLoc)

	// Calculate time difference
	_, fromOffset := inputTime.Zone()
	_, toOffset := convertedTime.Zone()
	diffHours := float64(toOffset-fromOffset) / 3600.0

	var diffStr string
	if diffHours > 0 {
		diffStr = fmt.Sprintf("+%.1f hours", diffHours)
	} else if diffHours < 0 {
		diffStr = fmt.Sprintf("%.1f hours", diffHours)
	} else {
		diffStr = "same time"
	}

	return fmt.Sprintf(
		"Time Conversion:\n"+
			"From: %s (%s)\n"+
			"To:   %s (%s)\n"+
			"Time difference: %s",
		inputTime.Format("2006-01-02 15:04:05 MST"),
		params.FromTimezone,
		convertedTime.Format("2006-01-02 15:04:05 MST"),
		params.ToTimezone,
		diffStr,
	), nil
}
