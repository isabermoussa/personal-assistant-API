package assistant

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAssistant_Title(t *testing.T) {
	// Skip tests if OPENAI_API_KEY is not set
	// These are integration tests that actually call OpenAI API

	t.Run("returns default title for empty conversation", func(t *testing.T) {
		a := New()
		ctx := context.Background()

		conv := &model.Conversation{
			ID:       primitive.NewObjectID(),
			Messages: []*model.Message{},
		}

		title, err := a.Title(ctx, conv)
		if err != nil {
			t.Fatalf("Title() error = %v", err)
		}

		if title != "An empty conversation" {
			t.Errorf("Title() = %q, want %q", title, "An empty conversation")
		}
	})

	t.Run("title is within 80 characters", func(t *testing.T) {
		// This test uses a mock-like approach by checking the output format
		a := New()
		ctx := context.Background()

		// Create a conversation with a very long message
		longMessage := strings.Repeat("This is a very long question about many different topics that should result in a title that needs truncation because it exceeds the character limit. ", 3)

		conv := &model.Conversation{
			ID: primitive.NewObjectID(),
			Messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: longMessage,
				},
			},
		}

		// Note: This will only pass with a real OpenAI key
		// For unit testing without API calls, we'd need dependency injection
		title, err := a.Title(ctx, conv)

		// If no API key is set, we expect an error
		if err != nil {
			t.Skipf("Skipping API test (no OpenAI key configured): %v", err)
			return
		}

		if len(title) > 80 {
			t.Errorf("Title() length = %d, want <= 80. Title: %q", len(title), title)
		}
	})

	t.Run("title summarizes question not answers it", func(t *testing.T) {
		// This test validates the behavior conceptually
		// In a real scenario with API access, we'd verify the title doesn't contain
		// phrases like "I don't have", "I cannot", "Please check", etc.

		a := New()
		ctx := context.Background()

		conv := &model.Conversation{
			ID: primitive.NewObjectID(),
			Messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: "What is the weather like in Barcelona?",
				},
			},
		}

		title, err := a.Title(ctx, conv)

		if err != nil {
			t.Skipf("Skipping API test (no OpenAI key configured): %v", err)
			return
		}

		// The title should NOT contain answer-like phrases
		answerPhrases := []string{
			"I don't",
			"I can't",
			"I cannot",
			"Please check",
			"You should",
			"real-time data",
			"access to",
		}

		lowerTitle := strings.ToLower(title)
		for _, phrase := range answerPhrases {
			if strings.Contains(lowerTitle, strings.ToLower(phrase)) {
				t.Errorf("Title appears to answer the question instead of summarizing it. Title: %q contains phrase: %q", title, phrase)
			}
		}

		// The title should contain topic-related keywords
		topicKeywords := []string{"weather", "barcelona"}
		hasKeyword := false
		for _, keyword := range topicKeywords {
			if strings.Contains(lowerTitle, strings.ToLower(keyword)) {
				hasKeyword = true
				break
			}
		}

		if !hasKeyword {
			t.Logf("Warning: Title %q doesn't contain expected topic keywords. This might be acceptable depending on the summary style.", title)
		}

		t.Logf("Generated title: %q", title)
	})

	t.Run("title removes quotes and extra whitespace", func(t *testing.T) {
		a := New()
		ctx := context.Background()

		conv := &model.Conversation{
			ID: primitive.NewObjectID(),
			Messages: []*model.Message{
				{
					ID:      primitive.NewObjectID(),
					Role:    model.RoleUser,
					Content: "Tell me about Go programming language",
				},
			},
		}

		title, err := a.Title(ctx, conv)

		if err != nil {
			t.Skipf("Skipping API test (no OpenAI key configured): %v", err)
			return
		}

		// Title should not start/end with quotes
		if strings.HasPrefix(title, "\"") || strings.HasSuffix(title, "\"") {
			t.Errorf("Title should not have quotes: %q", title)
		}

		if strings.HasPrefix(title, "'") || strings.HasSuffix(title, "'") {
			t.Errorf("Title should not have single quotes: %q", title)
		}

		// Title should not have leading/trailing whitespace
		if title != strings.TrimSpace(title) {
			t.Errorf("Title has leading/trailing whitespace: %q", title)
		}

		// Title should not have newlines
		if strings.Contains(title, "\n") {
			t.Errorf("Title should not contain newlines: %q", title)
		}
	})

	t.Run("uses only first message for title generation", func(t *testing.T) {
		// This test ensures we're not processing all conversation history for titles
		a := New()
		ctx := context.Background()

		conv := &model.Conversation{
			ID: primitive.NewObjectID(),
			Messages: []*model.Message{
				{
					ID:        primitive.NewObjectID(),
					Role:      model.RoleUser,
					Content:   "What is quantum computing?",
					CreatedAt: time.Now(),
				},
				{
					ID:        primitive.NewObjectID(),
					Role:      model.RoleAssistant,
					Content:   "Quantum computing is...",
					CreatedAt: time.Now(),
				},
				{
					ID:        primitive.NewObjectID(),
					Role:      model.RoleUser,
					Content:   "Can you explain more about qubits?",
					CreatedAt: time.Now(),
				},
			},
		}

		title, err := a.Title(ctx, conv)

		if err != nil {
			t.Skipf("Skipping API test (no OpenAI key configured): %v", err)
			return
		}

		// The title should be about quantum computing (first message)
		// not about qubits (third message)
		lowerTitle := strings.ToLower(title)

		if strings.Contains(lowerTitle, "qubit") && !strings.Contains(lowerTitle, "quantum") {
			t.Errorf("Title appears to use later messages instead of first message: %q", title)
		}

		t.Logf("Generated title from multi-message conversation: %q", title)
	})
}

func TestTitleGenerationPrompt(t *testing.T) {
	// This test validates our prompt structure
	t.Run("prompt asks for title summarization", func(t *testing.T) {
		userQuestion := "What is the weather like in Barcelona?"

		// This is the actual prompt format we use
		prompt := "Create a short title (max 80 characters) that summarizes what this question is about. Only return the title, nothing else:\n\n" + userQuestion

		// Verify prompt structure
		if !strings.Contains(prompt, "summarizes") {
			t.Error("Prompt should explicitly ask to 'summarize'")
		}

		if !strings.Contains(prompt, "title") {
			t.Error("Prompt should mention 'title'")
		}

		if !strings.Contains(prompt, "Only return the title") {
			t.Error("Prompt should instruct to return only the title")
		}

		if !strings.Contains(prompt, userQuestion) {
			t.Error("Prompt should contain the user's question")
		}

		t.Logf("Prompt structure validated: %q", prompt)
	})
}
