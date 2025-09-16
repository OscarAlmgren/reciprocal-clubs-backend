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
	
	grpcHandlers "reciprocal-clubs-backend/services/blockchain-service/internal/handlers/grpc"
	httpHandlers "reciprocal-clubs-backend/services/blockchain-service/internal/handlers/http"
	"reciprocal-clubs-backend/services/blockchain-service/internal/models"
	"reciprocal-clubs-backend/services/blockchain-service/internal/repository"
	"reciprocal-clubs-backend/services/blockchain-service/internal/service"
)

const serviceName = "blockchain-service"

func main() {
	// Load configuration
	cfg, err := config.Load(serviceName)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := logging.NewLogger(logging.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		Output:     cfg.Logging.Output,
		TimeFormat: cfg.Logging.TimeFormat,
	}, serviceName)

	logger.Info("Starting blockchain-service")

	// Initialize database
	db, err := database.NewConnection(cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	// Auto-migrate database schema
	if err := db.AutoMigrate(
		&models.Example{},
	); err != nil {
		logger.Fatal(fmt.Sprintf("Failed to migrate database: %v", err))
	}

	// Initialize NATS client
	natsConfig := messaging.Config{
		URL:            cfg.NATS.URL,
		ClusterID:      cfg.NATS.ClusterID,
		ClientID:       cfg.NATS.ClientID + "-" + serviceName,
		ConnectTimeout: time.Duration(cfg.NATS.ConnectTimeout) * time.Second,
		RequestTimeout: time.Duration(cfg.NATS.RequestTimeout) * time.Second,
		EnableJetStream: true,
	}

	natsClient, err := messaging.NewClient(natsConfig, serviceName)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to NATS: %v", err))
	}
	defer natsClient.Close()

	// Initialize monitoring
	monitoringService := monitoring.NewService(cfg.Monitoring, serviceName)

	// Initialize repository
	repo := repository.NewRepository(db, logger)

	// Initialize service
	blockchainserviceService := service.NewService(repo, logger, natsClient, monitoringService)

	// Initialize handlers
	httpHandler := httpHandlers.NewHTTPHandler(blockchainserviceService, logger, monitoringService)
	grpcHandler := grpcHandlers.NewGRPCHandler(blockchainserviceService, logger, monitoringService)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Service.Port),
		Handler: httpHandler.SetupRoutes(),
	}

	go func() {
		logger.Info(fmt.Sprintf("HTTP server listening on port %d", cfg.Service.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(fmt.Sprintf("HTTP server failed: %v", err))
		}
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler.RegisterServices(grpcServer)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to listen on gRPC port: %v", err))
	}

	go func() {
		logger.Info(fmt.Sprintf("gRPC server listening on port %d", cfg.Service.GRPCPort))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal(fmt.Sprintf("gRPC server failed: %v", err))
		}
	}()

	// Start monitoring server
	go func() {
		logger.Info(fmt.Sprintf("Metrics server listening on port %d", cfg.Monitoring.MetricsPort))
		if err := monitoringService.Start(); err != nil {
			logger.Error(fmt.Sprintf("Metrics server failed: %v", err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("HTTP server shutdown error: %v", err))
	}

	grpcServer.GracefulStop()

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	logger.Info("Servers stopped")
}
