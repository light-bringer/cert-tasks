package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/light-bringer/cert-tasks/internal/handlers"
	"github.com/light-bringer/cert-tasks/internal/repository"
	"github.com/light-bringer/cert-tasks/internal/server"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	} else {
		port = ":" + port
	}

	// Initialize repository
	repo := repository.NewMemoryRepository()

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(repo)

	// Create server
	srv := server.NewServer(taskHandler)

	// Create context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run server
	if err := srv.Run(ctx, port); err != nil {
		log.Fatal(err)
	}
}
