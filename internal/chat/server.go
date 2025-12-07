package chat

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"github.com/isabermoussa/personal-assistant-API/internal/pb"
	"github.com/twitchtv/twirp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ pb.ChatService = (*Server)(nil)

type Assistant interface {
	Title(ctx context.Context, conv *model.Conversation) (string, error)
	Reply(ctx context.Context, conv *model.Conversation) (string, error)
}

type Server struct {
	repo   *model.Repository
	assist Assistant
}

func NewServer(repo *model.Repository, assist Assistant) *Server {
	return &Server{repo: repo, assist: assist}
}

func (s *Server) StartConversation(ctx context.Context, req *pb.StartConversationRequest) (*pb.StartConversationResponse, error) {
	conversation := &model.Conversation{
		ID:        primitive.NewObjectID(),
		Title:     "Untitled conversation",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []*model.Message{{
			ID:        primitive.NewObjectID(),
			Role:      model.RoleUser,
			Content:   req.GetMessage(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}},
	}

	if strings.TrimSpace(req.GetMessage()) == "" {
		return nil, twirp.RequiredArgumentError("message")
	}

	// Generate title and reply concurrently for better performance
	var (
		title    string
		titleErr error
		reply    string
		replyErr error
		wg       sync.WaitGroup
	)

	wg.Add(2)

	// Generate title in background (non-critical)
	go func() {
		defer wg.Done()
		title, titleErr = s.assist.Title(ctx, conversation)
		if titleErr != nil {
			slog.ErrorContext(ctx, "Failed to generate conversation title", "error", titleErr)
		}
	}()

	// Generate reply in background (critical)
	go func() {
		defer wg.Done()
		reply, replyErr = s.assist.Reply(ctx, conversation)
	}()

	// Wait for both operations to complete
	wg.Wait()

	// Check if reply generation failed (critical error)
	if replyErr != nil {
		return nil, replyErr
	}

	// Use generated title if successful, otherwise keep default
	if titleErr == nil && title != "" {
		conversation.Title = title
	}

	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := s.repo.CreateConversation(ctx, conversation); err != nil {
		return nil, err
	}

	return &pb.StartConversationResponse{
		ConversationId: conversation.ID.Hex(),
		Title:          conversation.Title,
		Reply:          reply,
	}, nil
}

func (s *Server) ContinueConversation(ctx context.Context, req *pb.ContinueConversationRequest) (*pb.ContinueConversationResponse, error) {
	if req.GetConversationId() == "" {
		return nil, twirp.RequiredArgumentError("conversation_id")
	}

	if strings.TrimSpace(req.GetMessage()) == "" {
		return nil, twirp.RequiredArgumentError("message")
	}

	conversation, err := s.repo.DescribeConversation(ctx, req.GetConversationId())
	if err != nil {
		return nil, err
	}

	conversation.UpdatedAt = time.Now()
	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleUser,
		Content:   req.GetMessage(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	reply, err := s.assist.Reply(ctx, conversation)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	conversation.Messages = append(conversation.Messages, &model.Message{
		ID:        primitive.NewObjectID(),
		Role:      model.RoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := s.repo.UpdateConversation(ctx, conversation); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.ContinueConversationResponse{Reply: reply}, nil
}

func (s *Server) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	conversations, err := s.repo.ListConversations(ctx)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	resp := &pb.ListConversationsResponse{}
	for _, conv := range conversations {
		conv.Messages = nil // Clear messages to avoid sending large data
		resp.Conversations = append(resp.Conversations, conv.Proto())
	}

	return resp, nil
}

func (s *Server) DescribeConversation(ctx context.Context, req *pb.DescribeConversationRequest) (*pb.DescribeConversationResponse, error) {
	if req.GetConversationId() == "" {
		return nil, twirp.RequiredArgumentError("conversation_id")
	}

	conversation, err := s.repo.DescribeConversation(ctx, req.GetConversationId())
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, twirp.NotFoundError("conversation not found")
	}

	return &pb.DescribeConversationResponse{Conversation: conversation.Proto()}, nil
}
