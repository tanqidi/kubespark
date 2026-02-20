package apiserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	restful "github.com/emicklei/go-restful/v3"
)

// Server wraps HTTP server and configuration
type Server struct {
	httpServer *http.Server
	container  *restful.Container
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	container := restful.NewContainer()
	container.Filter(CORSFilter)
	container.Filter(AuthFilter)
	container.Filter(LoggingFilter)

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + port,
			Handler: container,
		},
		container: container,
	}
}

// Container returns the restful container
func (s *Server) Container() *restful.Container {
	return s.container
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🚀 Kubespark API server starting on %s", s.httpServer.Addr)
	log.Printf("📚 API Documentation available at http://localhost%s", s.httpServer.Addr)
	log.Printf("🔗 Kubernetes API available at /kapis/resources.kubespark.io/v1alpha1/*")

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// StartWithGracefulShutdown starts the server with graceful shutdown handling
func (s *Server) StartWithGracefulShutdown() {
	// Start server in goroutine
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
