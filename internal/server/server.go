package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/light-bringer/cert-tasks/internal/handlers"
)

// Server represents the HTTP server
type Server struct {
	router *chi.Mux
	server *http.Server
}

// NewServer creates a new HTTP server with configured routes and middleware
func NewServer(handler *handlers.TaskHandler) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)                        // Log all requests
	r.Use(middleware.Recoverer)                     // Recover from panics
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Routes
	r.Post("/tasks", handler.CreateTask)
	r.Get("/tasks", handler.ListTasks)
	r.Get("/tasks/{id}", handler.GetTask)
	r.Put("/tasks/{id}", handler.UpdateTask)
	r.Delete("/tasks/{id}", handler.DeleteTask)

	return &Server{
		router: r,
	}
}

// Run starts the HTTP server and handles graceful shutdown
func (s *Server) Run(ctx context.Context, port string) error {
	s.server = &http.Server{
		Addr:         port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", port)
		serverErrors <- s.server.ListenAndServe()
	}()

	// Block until context is cancelled or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		log.Println("Shutting down server...")

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		log.Println("Server stopped gracefully")
	}

	return nil
}
