package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
)

// mockAssistant provides a test double for Assistant interface
type mockAssistant struct {
	titleFunc func(ctx context.Context, conv *model.Conversation) (string, error)
	replyFunc func(ctx context.Context, conv *model.Conversation) (string, error)
}

func (m *mockAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if m.titleFunc != nil {
		return m.titleFunc(ctx, conv)
	}
	// Default: generate a simple title by truncating the first message
	if len(conv.Messages) == 0 {
		return "Empty Conversation", nil
	}
	title := conv.Messages[0].Content
	if len(title) > 50 {
		title = title[:50]
	}
	return title, nil
}

func (m *mockAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if m.replyFunc != nil {
		return m.replyFunc(ctx, conv)
	}
	// Default: return a canned response
	return "This is a test reply from the assistant.", nil
}

// newMockAssistant creates a mock assistant with default behavior
func newMockAssistant() *mockAssistant {
	return &mockAssistant{}
}

// withTitleFunc allows customizing the Title behavior
func (m *mockAssistant) withTitleFunc(fn func(ctx context.Context, conv *model.Conversation) (string, error)) *mockAssistant {
	m.titleFunc = fn
	return m
}

// withReplyFunc allows customizing the Reply behavior
func (m *mockAssistant) withReplyFunc(fn func(ctx context.Context, conv *model.Conversation) (string, error)) *mockAssistant {
	m.replyFunc = fn
	return m
}

// titleSummarizer creates a title by summarizing the first user message
func titleSummarizer(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "Empty Conversation", nil
	}

	msg := conv.Messages[0].Content
	words := strings.Fields(msg)

	// Simulate creating a summary title (take first 5 words)
	if len(words) > 5 {
		return strings.Join(words[:5], " "), nil
	}
	return msg, nil
}

// titleError simulates a title generation error
func titleError(ctx context.Context, conv *model.Conversation) (string, error) {
	return "", errors.New("simulated title generation error")
}

// replyError simulates a reply generation error
func replyError(ctx context.Context, conv *model.Conversation) (string, error) {
	return "", errors.New("simulated reply generation error")
}
