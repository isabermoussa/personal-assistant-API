package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/isabermoussa/personal-assistant-API/internal/chat"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/assistant"
	"github.com/isabermoussa/personal-assistant-API/internal/chat/model"
	"github.com/isabermoussa/personal-assistant-API/internal/httpx"
	"github.com/isabermoussa/personal-assistant-API/internal/mongox"
	"github.com/isabermoussa/personal-assistant-API/internal/pb"
	"github.com/isabermoussa/personal-assistant-API/internal/telemetry"
	"github.com/twitchtv/twirp"
)

func main() {
	ctx := context.Background()

	// Initialize OpenTelemetry (metrics + tracing)
	shutdown, err := telemetry.InitTelemetry(ctx)
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
		panic(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown telemetry", "error", err)
		}
	}()

	// Initialize metrics instruments
	metrics, err := telemetry.NewMetrics()
	if err != nil {
		slog.Error("Failed to create metrics", "error", err)
		panic(err)
	}

	mongo := mongox.MustConnect()

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		telemetry.TracingMiddleware(),        // Add tracing first (outer layer)
		telemetry.MetricsMiddleware(metrics), // Add metrics
		httpx.Logger(),                       // Existing logger
		httpx.Recovery(),                     // Existing recovery
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		slog.Info("Starting the server on :8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			panic(err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
