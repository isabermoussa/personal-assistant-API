package tools

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
)

// DateTool provides current date and time information
type DateTool struct{}

// NewDateTool creates a new date tool
func NewDateTool() *DateTool {
	return &DateTool{}
}

func (t *DateTool) Name() string {
	return "get_today_date"
}

func (t *DateTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_today_date",
		Description: openai.String("Get today's date and time in RFC3339 format"),
	})
}

func (t *DateTool) Handle(ctx context.Context, args string) (string, error) {
	// No parameters needed for this tool
	return time.Now().Format(time.RFC3339), nil
}
