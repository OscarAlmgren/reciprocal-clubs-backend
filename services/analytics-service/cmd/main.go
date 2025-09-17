package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"

	grpcHandlers "reciprocal-clubs-backend/services/analytics-service/internal/handlers/grpc"
	httpHandlers "reciprocal-clubs-backend/services/analytics-service/internal/handlers/http"
	"reciprocal-clubs-backend/services/analytics-service/internal/models"
	"reciprocal-clubs-backend/services/analytics-service/internal/repository"
	"reciprocal-clubs-backend/services/analytics-service/internal/service"
)

const serviceName = "analytics-service"

func main() {
	// Load configuration
	cfg, err := config.Load(serviceName)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger(&cfg.Logging, serviceName)

	logger.Info("Starting analytics-service", map[string]interface{}{})

	// Initialize database
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", map[string]interface{}{"error": err.Error()})
	}

	// Auto-migrate database schema
	if err := db.AutoMigrate(
		&models.Example{},
		&repository.AnalyticsEvent{},
		&repository.AnalyticsMetric{},
		&repository.AnalyticsReport{},
	); err != nil {
		logger.Fatal("Failed to migrate database", map[string]interface{}{"error": err.Error()})
	}

	// Initialize NATS client
	natsClient, err := messaging.NewNATSMessageBus(&cfg.NATS, logger)
	if err != nil {
		logger.Fatal("Failed to connect to NATS", map[string]interface{}{"error": err.Error()})
	}
	defer natsClient.Close()

	// Initialize monitoring
	monitoringService := monitoring.NewMonitor(&cfg.Monitoring, logger, serviceName, cfg.Service.Version)

	// Initialize repository
	repo := repository.NewRepository(db.DB, logger)

	// Initialize service
	analyticsService := service.NewService(repo, logger, natsClient, monitoringService)

	// Start event processor
	if err := analyticsService.StartEventProcessor(); err != nil {
		logger.Error("Failed to start event processor", map[string]interface{}{"error": err.Error()})
	}

	// Initialize handlers
	httpHandler := httpHandlers.NewHTTPHandler(analyticsService, logger, monitoringService)
	grpcHandler := grpcHandlers.NewGRPCHandler(analyticsService, logger, monitoringService)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Service.Port),
		Handler: httpHandler.SetupRoutes(),
	}

	go func() {
		logger.Info("HTTP server listening", map[string]interface{}{"port": cfg.Service.Port})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler.RegisterServices(grpcServer)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to listen on gRPC port", map[string]interface{}{"error": err.Error(), "port": cfg.Service.GRPCPort})
	}

	go func() {
		logger.Info("gRPC server listening", map[string]interface{}{"port": cfg.Service.GRPCPort})
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server failed", map[string]interface{}{"error": err.Error()})
		}
	}()

	// Start monitoring server
	go func() {
		logger.Info("Metrics server listening", map[string]interface{}{"port": cfg.Monitoring.MetricsPort})
		monitoringService.StartMetricsServer()
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...", map[string]interface{}{})

	// Stop event processor
	if err := analyticsService.StopEventProcessor(); err != nil {
		logger.Error("Failed to stop event processor", map[string]interface{}{"error": err.Error()})
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", map[string]interface{}{"error": err.Error()})
	}

	grpcServer.GracefulStop()

	if sqlDB, err := db.DB.DB(); err == nil {
		sqlDB.Close()
	}

	logger.Info("Servers stopped", map[string]interface{}{})
}
