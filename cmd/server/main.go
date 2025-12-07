package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/isabermoussa/personal-assistant-API/internal/chat"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/assistant"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"github.com/isabermoussa/personal-assistant-API/internal/httpx"
	"github.com/isabermoussa/personal-assistant-API/internal/mongox"
	"github.com/isabermoussa/personal-assistant-API/internal/pb"
	"github.com/twitchtv/twirp"
)

func main() {
	mongo := mongox.MustConnect()

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.Logger(),
		httpx.Recovery(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Start the server
	slog.Info("Starting the server...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
