package chat

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	. "github.com/isabermoussa/personal-assistant-API/internal/chat/testing"
	"github.com/isabermoussa/personal-assistant-API/internal/pb"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new conversation with title and reply", WithFixture(func(t *testing.T, f *Fixture) {
		assist := newMockAssistant().
			withTitleFunc(titleSummarizer).
			withReplyFunc(func(ctx context.Context, conv *model.Conversation) (string, error) {
				return "Barcelona has beautiful weather today!", nil
			})

		srv := NewServer(f.Repository, assist)

		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "What is the weather like in Barcelona?",
		})

		if err != nil {
			t.Fatalf("StartConversation failed: %v", err)
		}

		// Verify response structure
		if resp.GetConversationId() == "" {
			t.Error("expected conversation ID, got empty string")
		}

		if resp.GetTitle() == "" {
			t.Error("expected title to be populated")
		}

		if resp.GetTitle() == "Untitled conversation" {
			t.Error("expected generated title, still has default title")
		}

		// Title should summarize the question, not answer it
		if resp.GetTitle() == "What is the weather like" {
			t.Logf("Title correctly summarizes question: %s", resp.GetTitle())
		}

		if resp.GetReply() == "" {
			t.Error("expected reply to be populated")
		}

		if resp.GetReply() != "Barcelona has beautiful weather today!" {
			t.Errorf("expected reply to match, got: %s", resp.GetReply())
		}

		// Verify conversation was saved to database
		saved, err := f.Repository.DescribeConversation(ctx, resp.GetConversationId())
		if err != nil {
			t.Fatalf("failed to retrieve saved conversation: %v", err)
		}

		if saved.Title != resp.GetTitle() {
			t.Errorf("saved title mismatch: got %q, want %q", saved.Title, resp.GetTitle())
		}

		if len(saved.Messages) != 2 {
			t.Errorf("expected 2 messages (user + assistant), got %d", len(saved.Messages))
		}

		// Clean up
		f.Repository.DeleteConversation(ctx, resp.GetConversationId())
	}))

	t.Run("handles title generation failure gracefully", WithFixture(func(t *testing.T, f *Fixture) {
		assist := newMockAssistant().
			withTitleFunc(titleError).
			withReplyFunc(func(ctx context.Context, conv *model.Conversation) (string, error) {
				return "Assistant reply works fine", nil
			})

		srv := NewServer(f.Repository, assist)

		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "Test message",
		})

		if err != nil {
			t.Fatalf("StartConversation should not fail when only title fails: %v", err)
		}

		// Should use default title when generation fails
		if resp.GetTitle() != "Untitled conversation" {
			t.Errorf("expected default title on error, got: %s", resp.GetTitle())
		}

		// Reply should still work
		if resp.GetReply() != "Assistant reply works fine" {
			t.Errorf("expected reply to succeed, got: %s", resp.GetReply())
		}

		// Clean up
		f.Repository.DeleteConversation(ctx, resp.GetConversationId())
	}))

	t.Run("fails when reply generation fails", WithFixture(func(t *testing.T, f *Fixture) {
		assist := newMockAssistant().
			withTitleFunc(titleSummarizer).
			withReplyFunc(replyError)

		srv := NewServer(f.Repository, assist)

		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "Test message",
		})

		if err == nil {
			t.Fatal("expected error when reply generation fails, got nil")
		}

		if err.Error() != "simulated reply generation error" {
			t.Errorf("expected reply error to propagate, got: %v", err)
		}
	}))

	t.Run("rejects empty message", WithFixture(func(t *testing.T, f *Fixture) {
		srv := NewServer(f.Repository, newMockAssistant())

		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "   ",
		})

		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		te, ok := err.(twirp.Error)
		if !ok {
			t.Fatalf("expected twirp error, got: %T", err)
		}

		if te.Code() != twirp.InvalidArgument {
			t.Errorf("expected InvalidArgument error, got: %s", te.Code())
		}
	}))

	t.Run("stores conversation with correct message structure", WithFixture(func(t *testing.T, f *Fixture) {
		assist := newMockAssistant()
		srv := NewServer(f.Repository, assist)

		userMsg := "Tell me about Go programming"
		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: userMsg,
		})

		if err != nil {
			t.Fatalf("StartConversation failed: %v", err)
		}

		// Retrieve and verify conversation structure
		saved, err := f.Repository.DescribeConversation(ctx, resp.GetConversationId())
		if err != nil {
			t.Fatalf("failed to retrieve conversation: %v", err)
		}

		// Check first message (user)
		if saved.Messages[0].Role != model.RoleUser {
			t.Errorf("first message should be user role, got: %s", saved.Messages[0].Role)
		}
		if saved.Messages[0].Content != userMsg {
			t.Errorf("first message content mismatch: got %q, want %q", saved.Messages[0].Content, userMsg)
		}

		// Check second message (assistant)
		if saved.Messages[1].Role != model.RoleAssistant {
			t.Errorf("second message should be assistant role, got: %s", saved.Messages[1].Role)
		}
		if saved.Messages[1].Content != resp.GetReply() {
			t.Errorf("second message content mismatch: got %q, want %q", saved.Messages[1].Content, resp.GetReply())
		}

		// Clean up
		f.Repository.DeleteConversation(ctx, resp.GetConversationId())
	}))

	t.Run("concurrent title and reply generation both complete", WithFixture(func(t *testing.T, f *Fixture) {
		// This test verifies the concurrent optimization works correctly
		titleCalled := false
		replyCalled := false

		assist := newMockAssistant().
			withTitleFunc(func(ctx context.Context, conv *model.Conversation) (string, error) {
				titleCalled = true
				return "Concurrent Title Test", nil
			}).
			withReplyFunc(func(ctx context.Context, conv *model.Conversation) (string, error) {
				replyCalled = true
				return "Concurrent reply test", nil
			})

		srv := NewServer(f.Repository, assist)

		resp, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
			Message: "Test concurrent execution",
		})

		if err != nil {
			t.Fatalf("StartConversation failed: %v", err)
		}

		if !titleCalled {
			t.Error("title generation was not called")
		}

		if !replyCalled {
			t.Error("reply generation was not called")
		}

		if resp.GetTitle() != "Concurrent Title Test" {
			t.Errorf("title not properly set: got %q", resp.GetTitle())
		}

		if resp.GetReply() != "Concurrent reply test" {
			t.Errorf("reply not properly set: got %q", resp.GetReply())
		}

		// Clean up
		f.Repository.DeleteConversation(ctx, resp.GetConversationId())
	}))
}

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}
