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
	logger := logging.NewLogger(&cfg.Logging, serviceName)

	logger.Info("Starting blockchain service", nil)

	// Initialize database
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer db.Close()

	// Auto-migrate database schema
	if err := db.Migrate(
		&models.FabricTransaction{},
		&models.Channel{},
		&models.Chaincode{},
		&models.Peer{},
		&models.Block{},
		&models.Event{},
	); err != nil {
		logger.Fatal("Failed to migrate database", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize message bus
	messageBus, err := messaging.NewNATSMessageBus(&cfg.NATS, logger)
	if err != nil {
		logger.Fatal("Failed to create message bus", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer messageBus.Close()

	// Initialize monitoring
	monitor := monitoring.NewMonitor(&cfg.Monitoring, logger, serviceName, cfg.Service.Version)

	// Start metrics server
	monitor.StartMetricsServer()

	// Initialize repository
	repo := repository.NewRepository(db.DB, logger)

	// Initialize service
	blockchainService := service.NewService(repo, logger, messageBus, monitor)

	// Initialize HTTP handlers
	httpHandler := httpHandlers.NewHTTPHandler(blockchainService, logger, monitor)

	// Initialize gRPC handlers
	grpcHandler := grpcHandlers.NewGRPCHandler(blockchainService, logger, monitor)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Service.Port),
		Handler: httpHandler.SetupRoutes(),
	}

	go func() {
		logger.Info("HTTP server listening", map[string]interface{}{
			"port": cfg.Service.Port,
		})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler.RegisterServices(grpcServer)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to listen on gRPC port", map[string]interface{}{
			"error": err.Error(),
			"port":  cfg.Service.GRPCPort,
		})
	}

	go func() {
		logger.Info("gRPC server listening", map[string]interface{}{
			"port": cfg.Service.GRPCPort,
		})
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Monitoring server is already started above with StartMetricsServer()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...", nil)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Close database connection
	db.Close()

	logger.Info("Servers stopped", nil)
}
