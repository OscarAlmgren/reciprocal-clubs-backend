package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	grpchandler "reciprocal-clubs-backend/services/member-service/internal/handlers/grpc"
	httphandler "reciprocal-clubs-backend/services/member-service/internal/handlers/http"
	"reciprocal-clubs-backend/services/member-service/internal/models"
	"reciprocal-clubs-backend/services/member-service/internal/repository"
	"reciprocal-clubs-backend/services/member-service/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load("MEMBER_SERVICE")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger := logging.NewLogger(&cfg.Logging, "member-service")
	logger.Info("Starting Member Service", map[string]interface{}{
		"version":     cfg.Service.Version,
		"environment": cfg.Service.Environment,
		"port":        cfg.Service.Port,
		"grpc_port":   cfg.Service.GRPCPort,
	})

	// Initialize monitoring
	monitor := monitoring.NewMonitor(&cfg.Monitoring, logger, "member-service", cfg.Service.Version)

	// Initialize database
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to initialize database", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	defer db.Close()

	// Auto-migrate database schema
	if err := autoMigrate(db); err != nil {
		logger.Error("Failed to migrate database", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Initialize message bus
	messageBus, err := messaging.NewNATSMessageBus(&cfg.NATS, logger)
	if err != nil {
		logger.Error("Failed to initialize message bus", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	defer messageBus.Close()

	// Initialize repository
	repo := repository.NewRepository(db, logger)

	// Initialize service
	memberService := service.NewService(repo, logger, messageBus)

	// Initialize gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler := grpchandler.NewHandler(memberService, logger)
	// TODO: Register gRPC service when proto is generated
	// memberpb.RegisterMemberServiceServer(grpcServer, grpcHandler)
	_ = grpcHandler // Silence unused variable warning

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		logger.Error("Failed to create gRPC listener", map[string]interface{}{
			"error": err.Error(),
			"port":  cfg.Service.GRPCPort,
		})
		os.Exit(1)
	}

	go func() {
		logger.Info("Starting gRPC server", map[string]interface{}{
			"port": cfg.Service.GRPCPort,
		})
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("gRPC server failed", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	// Initialize HTTP server
	router := mux.NewRouter()
	httpHandler := httphandler.NewHandler(memberService, logger, monitor)
	httpHandler.RegisterRoutes(router)

	// Add monitoring endpoints
	router.HandleFunc("/health", monitor.HealthCheckHandler()).Methods("GET")
	router.HandleFunc("/ready", monitor.ReadinessCheckHandler()).Methods("GET")

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Service.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", map[string]interface{}{
			"port": cfg.Service.Port,
		})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	// Register health checks
	monitor.RegisterHealthCheck(&serviceHealthChecker{service: memberService})

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Member Service...", nil)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	logger.Info("Member Service stopped", nil)
}

// autoMigrate runs database migrations
func autoMigrate(db *database.Database) error {
	return db.AutoMigrate(
		&models.Member{},
		&models.MemberProfile{},
		&models.Address{},
		&models.EmergencyContact{},
		&models.MemberPreferences{},
	)
}

// serviceHealthChecker implements health check for the member service
type serviceHealthChecker struct {
	service service.Service
}

func (h *serviceHealthChecker) Name() string {
	return "member_service"
}

func (h *serviceHealthChecker) HealthCheck(ctx context.Context) error {
	return h.service.HealthCheck(ctx)
}