package assistant

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/isabermoussa/personal-assistant-API/internal/chat/assistant/tools"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/assistant/weather"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli           openai.Client
	weatherClient *weather.Client
	tools         []tools.Tool
}

// Option configures an Assistant
type Option func(*Assistant)

// WithWeatherClient sets a custom weather client
func WithWeatherClient(client *weather.Client) Option {
	return func(a *Assistant) {
		a.weatherClient = client
	}
}

// WithOpenAIClient sets a custom OpenAI client
func WithOpenAIClient(client openai.Client) Option {
	return func(a *Assistant) {
		a.cli = client
	}
}

// New creates a new Assistant with optional configuration
func New(opts ...Option) *Assistant {
	a := &Assistant{
		cli:           openai.NewClient(),
		weatherClient: weather.NewClient(),
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	// Initialize tools with dependencies
	a.tools = []tools.Tool{
		tools.NewWeatherTool(a.weatherClient),
		tools.NewDateTool(),
		tools.NewHolidaysTool(),
		tools.NewTimeZoneTool(),
	}

	return a
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
			Tools:    tools.Definitions(a.tools),
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
				msgs = append(msgs, tools.Dispatch(ctx, a.tools, call))
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
