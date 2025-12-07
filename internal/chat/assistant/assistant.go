package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli           openai.Client
	weatherClient *WeatherClient
}

func New() *Assistant {
	return &Assistant{
		cli:           openai.NewClient(),
		weatherClient: NewWeatherClient(),
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)
	// Build messages array: system instruction first, then user messages
	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a title generator. Extract the main topic from the user's message and create a short, descriptive title. Do NOT answer the question. Examples: 'What is the weather like in Barcelona?' → 'Weather in Barcelona'. Maximum 80 characters, no quotes."),
	}
	for _, m := range conv.Messages {
		msgs = append(msgs, openai.UserMessage(m.Content))
	}

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4o,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
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
								"maximum":     3,
							},
						},
						"required": []string{"location"},
					},
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_today_date",
					Description: openai.String("Get today's date and time in RFC3339 format"),
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
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
				}),
			},
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				switch call.Function.Name {
				case "get_weather":
					// Parse tool call arguments
					var weatherArgs struct {
						Location     string `json:"location"`
						ForecastDays int    `json:"forecast_days,omitempty"`
					}

					if err := json.Unmarshal([]byte(call.Function.Arguments), &weatherArgs); err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to parse weather request: "+err.Error(), call.ID))
						break
					}

					// Determine if forecast is requested
					if weatherArgs.ForecastDays > 0 {
						// Get forecast
						forecast, err := a.weatherClient.GetForecast(ctx, weatherArgs.Location, weatherArgs.ForecastDays)
						if err != nil {
							slog.ErrorContext(ctx, "Failed to fetch weather forecast", "error", err, "location", weatherArgs.Location)
							msgs = append(msgs, openai.ToolMessage("Failed to fetch weather forecast: "+err.Error(), call.ID))
							break
						}
						msgs = append(msgs, openai.ToolMessage(FormatForecast(forecast), call.ID))
					} else {
						// Get current weather
						weather, err := a.weatherClient.GetCurrentWeather(ctx, weatherArgs.Location)
						if err != nil {
							slog.ErrorContext(ctx, "Failed to fetch current weather", "error", err, "location", weatherArgs.Location)
							msgs = append(msgs, openai.ToolMessage("Failed to fetch weather: "+err.Error(), call.ID))
							break
						}
						msgs = append(msgs, openai.ToolMessage(FormatCurrentWeather(weather), call.ID))
					}
				case "get_today_date":
					msgs = append(msgs, openai.ToolMessage(time.Now().Format(time.RFC3339), call.ID))
				case "get_holidays":
					link := "https://www.officeholidays.com/ics/spain/catalonia"
					if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
						link = v
					}

					events, err := LoadCalendar(ctx, link)
					if err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to load holiday events", call.ID))
						break
					}

					var payload struct {
						BeforeDate time.Time `json:"before_date,omitempty"`
						AfterDate  time.Time `json:"after_date,omitempty"`
						MaxCount   int       `json:"max_count,omitempty"`
					}

					if err := json.Unmarshal([]byte(call.Function.Arguments), &payload); err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to parse tool call arguments: "+err.Error(), call.ID))
						break
					}

					var holidays []string
					for _, event := range events {
						date, err := event.GetAllDayStartAt()
						if err != nil {
							continue
						}

						if payload.MaxCount > 0 && len(holidays) >= payload.MaxCount {
							break
						}

						if !payload.BeforeDate.IsZero() && date.After(payload.BeforeDate) {
							continue
						}

						if !payload.AfterDate.IsZero() && date.Before(payload.AfterDate) {
							continue
						}

						holidays = append(holidays, date.Format(time.DateOnly)+": "+event.GetProperty(ics.ComponentPropertySummary).Value)
					}

					msgs = append(msgs, openai.ToolMessage(strings.Join(holidays, "\n"), call.ID))
				default:
					return "", errors.New("unknown tool call: " + call.Function.Name)
				}
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
