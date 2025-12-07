package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openai/openai-go/v2"
)

// Tool represents an assistant capability that can be called by the AI.
// Each tool defines its schema and execution logic independently.
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Definition returns the OpenAI function definition including
	// description and parameter schema
	Definition() openai.ChatCompletionToolUnionParam

	// Handle executes the tool with the given JSON arguments
	// and returns the result as a string
	Handle(ctx context.Context, args string) (string, error)
}

// Definitions converts a slice of tools into OpenAI tool parameters
func Definitions(tools []Tool) []openai.ChatCompletionToolUnionParam {
	defs := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, tool := range tools {
		defs[i] = tool.Definition()
	}
	return defs
}

// Dispatch finds and executes the appropriate tool for a given tool call.
// Returns an OpenAI tool message with the result or error.
func Dispatch(ctx context.Context, tools []Tool, call openai.ChatCompletionMessageToolCallUnion) openai.ChatCompletionMessageParamUnion {
	// Extract function name and arguments based on tool type
	var functionName, arguments string

	switch call.Type {
	case "function":
		functionName = call.Function.Name
		arguments = call.Function.Arguments
	case "custom":
		functionName = call.Custom.Name
		arguments = call.Custom.Input
	default:
		slog.WarnContext(ctx, "Unknown tool call type", "type", call.Type)
		return openai.ToolMessage(fmt.Sprintf("Unknown tool call type: %s", call.Type), call.ID)
	}

	for _, tool := range tools {
		if tool.Name() == functionName {
			result, err := tool.Handle(ctx, arguments)
			if err != nil {
				slog.ErrorContext(ctx, "Tool execution failed",
					"tool", tool.Name(),
					"error", err,
					"args", arguments,
				)
				return openai.ToolMessage(fmt.Sprintf("Tool failed: %v", err), call.ID)
			}
			return openai.ToolMessage(result, call.ID)
		}
	}

	slog.WarnContext(ctx, "Unknown tool called", "tool", functionName)
	return openai.ToolMessage(fmt.Sprintf("Unknown tool: %s", functionName), call.ID)
}
