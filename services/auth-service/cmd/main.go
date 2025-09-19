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

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/messaging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/services/auth-service/internal/handlers"
	"reciprocal-clubs-backend/services/auth-service/internal/models"
	"reciprocal-clubs-backend/services/auth-service/internal/repository"
	"reciprocal-clubs-backend/services/auth-service/internal/service"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const serviceName = "auth-service"

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

	// Initialize database
	db, err := database.NewConnection(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer db.Close()

	// Register database health check
	monitor.RegisterHealthCheck(db)

	// Run migrations
	if err := runMigrations(db); err != nil {
		logger.Fatal("Failed to run database migrations", map[string]interface{}{
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

	// Register message bus health check
	monitor.RegisterHealthCheck(&messageBusHealthChecker{messageBus})

	// Initialize repository
	repo := repository.NewAuthRepository(db, logger)

	// Initialize service
	authService := service.NewAuthService(repo, messageBus, cfg, logger)

	// Initialize handlers
	httpHandler := handlers.NewHTTPHandler(authService, logger, monitor)
	grpcHandler := handlers.NewAuthGRPCServer(authService, logger, monitor)

	// Start HTTP server
	httpServer := startHTTPServer(cfg, httpHandler, logger)
	defer httpServer.Shutdown(context.Background())

	// Start gRPC server
	grpcServer, grpcListener := startGRPCServer(cfg, grpcHandler, logger)
	defer grpcServer.GracefulStop()

	logger.Info("Auth service started successfully", map[string]interface{}{
		"http_port": cfg.Service.Port,
		"grpc_port": cfg.Service.GRPCPort,
	})

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down auth service...", map[string]interface{}{})

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()
	grpcListener.Close()

	logger.Info("Auth service stopped", map[string]interface{}{})
}

func runMigrations(db *database.Database) error {
	return db.Migrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.Club{},
		&models.UserSession{},
		&models.AuditLog{},
	)
}

func startHTTPServer(cfg *config.Config, handler *handlers.HTTPHandler, logger logging.Logger) *http.Server {
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Service.Host, cfg.Service.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Service.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Service.Timeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Starting HTTP server", map[string]interface{}{
			"address": server.Addr,
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return server
}

func startGRPCServer(cfg *config.Config, handler *handlers.AuthGRPCServer, logger logging.Logger) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Service.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to listen on gRPC port", map[string]interface{}{
			"error": err.Error(),
			"port":  cfg.Service.GRPCPort,
		})
	}

	server := grpc.NewServer()
	handler.RegisterServer(server)

	// Enable reflection for development
	if cfg.Service.Environment != "production" {
		reflection.Register(server)
	}

	go func() {
		logger.Info("Starting gRPC server", map[string]interface{}{
			"address": lis.Addr().String(),
		})

		if err := server.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return server, lis
}

type messageBusHealthChecker struct {
	messageBus messaging.MessageBus
}

func (h *messageBusHealthChecker) Name() string {
	return "message_bus"
}

func (h *messageBusHealthChecker) HealthCheck(ctx context.Context) error {
	return h.messageBus.HealthCheck(ctx)
}
