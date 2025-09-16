package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/api-gateway/internal/server"

	"github.com/rs/cors"
)

const serviceName = "api-gateway"

func main() {
	// Load configuration
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger := logging.NewLogger(&cfg.Logging, serviceName)

	// Initialize monitor
	monitor := monitoring.NewMonitor(&cfg.Monitoring, logger, serviceName, cfg.Service.Version)

	// Start metrics server
	monitor.StartMetricsServer()

	// Create HTTP server
	httpServer, err := server.NewServer(cfg, logger, monitor)
	if err != nil {
		logger.Fatal("Failed to create server", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Configure CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001", 
			"https://app.reciprocalclubs.com",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"Accept",
			"Origin",
			"Cache-Control",
			"X-File-Name",
		},
		AllowCredentials: true,
		MaxAge:          300,
	})

	// Wrap server with CORS
	handler := corsHandler.Handler(httpServer.Handler())

	// Create HTTP server instance
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Service.Host, cfg.Service.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Service.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Service.Timeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting API Gateway server", map[string]interface{}{
			"address": srv.Addr,
		})

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...", nil)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close server resources
	if err := httpServer.Close(); err != nil {
		logger.Error("Error closing server resources", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Info("Server gracefully stopped", nil)
}