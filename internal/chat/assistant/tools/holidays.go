package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/openai/openai-go/v2"
)

// HolidaysTool provides information about local bank and public holidays
type HolidaysTool struct{}

// NewHolidaysTool creates a new holidays tool
func NewHolidaysTool() *HolidaysTool {
	return &HolidaysTool{}
}

func (t *HolidaysTool) Name() string {
	return "get_holidays"
}

func (t *HolidaysTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_holidays",
		Description: openai.String("Gets local bank and public holidays. Each line is a single holiday in the format 'YYYY-MM-DD: Holiday Name'."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"before_date": map[string]string{
					"type":        "string",
					"description": "Optional date in RFC3339 format to get holidays before this date. If not provided, all holidays will be returned.",
				},
				"after_date": map[string]string{
					"type":        "string",
					"description": "Optional date in RFC3339 format to get holidays after this date. If not provided, all holidays will be returned.",
				},
				"max_count": map[string]string{
					"type":        "integer",
					"description": "Optional maximum number of holidays to return. If not provided, all holidays will be returned.",
				},
			},
		},
	})
}

func (t *HolidaysTool) Handle(ctx context.Context, args string) (string, error) {
	var params struct {
		BeforeDate time.Time `json:"before_date,omitempty"`
		AfterDate  time.Time `json:"after_date,omitempty"`
		MaxCount   int       `json:"max_count,omitempty"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid holiday parameters: %w", err)
	}

	// Get calendar URL from environment or use default
	link := "https://www.officeholidays.com/ics/spain/catalonia"
	if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
		link = v
	}

	// Load calendar events
	events, err := loadCalendar(ctx, link)
	if err != nil {
		return "", fmt.Errorf("failed to load holiday calendar: %w", err)
	}

	// Filter and format holidays
	var holidays []string
	for _, event := range events {
		date, err := event.GetAllDayStartAt()
		if err != nil {
			continue
		}

		// Check max count limit
		if params.MaxCount > 0 && len(holidays) >= params.MaxCount {
			break
		}

		// Filter by before date
		if !params.BeforeDate.IsZero() && date.After(params.BeforeDate) {
			continue
		}

		// Filter by after date
		if !params.AfterDate.IsZero() && date.Before(params.AfterDate) {
			continue
		}

		// Format holiday
		holidayName := event.GetProperty(ics.ComponentPropertySummary).Value
		holidays = append(holidays, date.Format(time.DateOnly)+": "+holidayName)
	}

	if len(holidays) == 0 {
		return "No holidays found matching the criteria.", nil
	}

	return strings.Join(holidays, "\n"), nil
}

// loadCalendar loads calendar events from a given ICS URL
func loadCalendar(ctx context.Context, link string) ([]*ics.VEvent, error) {
	slog.InfoContext(ctx, "Loading calendar", "link", link)

	cal, err := ics.ParseCalendarFromUrl(link, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse calendar: %w", err)
	}

	return cal.Events(), nil
}
