#!/bin/bash

# Script to create a basic microservice structure
# Usage: ./create-service.sh <service-name> <port> <grpc-port>

SERVICE_NAME=$1
HTTP_PORT=$2
GRPC_PORT=$3

if [ -z "$SERVICE_NAME" ] || [ -z "$HTTP_PORT" ] || [ -z "$GRPC_PORT" ]; then
    echo "Usage: ./create-service.sh <service-name> <port> <grpc-port>"
    echo "Example: ./create-service.sh blockchain-service 8084 9094"
    exit 1
fi

SERVICE_DIR="services/$SERVICE_NAME"
echo "Creating service structure for $SERVICE_NAME..."

# Create directory structure
mkdir -p "$SERVICE_DIR"/{cmd,internal/{handlers/{http,grpc},models,repository,service},api/proto,config,migrations,k8s,tests/{unit,integration,e2e}}

# Create go.mod
cat > "$SERVICE_DIR/go.mod" << EOF
module reciprocal-clubs-backend/services/$SERVICE_NAME

go 1.25

require (
	github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.4.0
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.31.0
	github.com/prometheus/client_golang v1.17.0
	github.com/rs/zerolog v1.31.0
	github.com/spf13/viper v1.17.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
	reciprocal-clubs-backend/pkg/shared/auth v0.0.0
	reciprocal-clubs-backend/pkg/shared/config v0.0.0
	reciprocal-clubs-backend/pkg/shared/database v0.0.0
	reciprocal-clubs-backend/pkg/shared/errors v0.0.0
	reciprocal-clubs-backend/pkg/shared/logging v0.0.0
	reciprocal-clubs-backend/pkg/shared/messaging v0.0.0
	reciprocal-clubs-backend/pkg/shared/monitoring v0.0.0
	reciprocal-clubs-backend/pkg/shared/utils v0.0.0
)

replace reciprocal-clubs-backend/pkg/shared/auth => ../../pkg/shared/auth
replace reciprocal-clubs-backend/pkg/shared/config => ../../pkg/shared/config
replace reciprocal-clubs-backend/pkg/shared/database => ../../pkg/shared/database
replace reciprocal-clubs-backend/pkg/shared/errors => ../../pkg/shared/errors
replace reciprocal-clubs-backend/pkg/shared/logging => ../../pkg/shared/logging
replace reciprocal-clubs-backend/pkg/shared/messaging => ../../pkg/shared/messaging
replace reciprocal-clubs-backend/pkg/shared/monitoring => ../../pkg/shared/monitoring
replace reciprocal-clubs-backend/pkg/shared/utils => ../../pkg/shared/utils
EOF

# Create main.go
MAIN_GO=$(echo "$SERVICE_NAME" | sed 's/-//g')
cat > "$SERVICE_DIR/cmd/main.go" << EOF
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

	grpcHandlers "reciprocal-clubs-backend/services/$SERVICE_NAME/internal/handlers/grpc"
	httpHandlers "reciprocal-clubs-backend/services/$SERVICE_NAME/internal/handlers/http"
	"reciprocal-clubs-backend/services/$SERVICE_NAME/internal/models"
	"reciprocal-clubs-backend/services/$SERVICE_NAME/internal/repository"
	"reciprocal-clubs-backend/services/$SERVICE_NAME/internal/service"
)

const serviceName = "$SERVICE_NAME"

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

	logger.Info("Starting $SERVICE_NAME")

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
	${MAIN_GO}Service := service.NewService(repo, logger, natsClient, monitoringService)

	// Initialize handlers
	httpHandler := httpHandlers.NewHTTPHandler(${MAIN_GO}Service, logger, monitoringService)
	grpcHandler := grpcHandlers.NewGRPCHandler(${MAIN_GO}Service, logger, monitoringService)

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
EOF

# Create basic models
cat > "$SERVICE_DIR/internal/models/models.go" << EOF
package models

import (
	"time"
	"gorm.io/gorm"
)

// Example model - replace with actual models
type Example struct {
	ID        uint           \`json:"id" gorm:"primaryKey"\`
	Name      string         \`json:"name" gorm:"size:255;not null"\`
	Status    string         \`json:"status" gorm:"size:50;default:'active'"\`
	CreatedAt time.Time      \`json:"created_at"\`
	UpdatedAt time.Time      \`json:"updated_at"\`
	DeletedAt gorm.DeletedAt \`json:"-" gorm:"index"\`
}

func (Example) TableName() string {
	return "${SERVICE_NAME//-/_}_examples"
}
EOF

# Create Dockerfile
cat > "$SERVICE_DIR/Dockerfile" << EOF
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY pkg/ ./pkg/
COPY services/$SERVICE_NAME/ ./services/$SERVICE_NAME/

WORKDIR /app/services/$SERVICE_NAME
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

FROM scratch
COPY --from=builder /app/services/$SERVICE_NAME/main /main
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE $HTTP_PORT $GRPC_PORT
CMD ["/main"]
EOF

echo "Service $SERVICE_NAME created successfully!"
echo "HTTP Port: $HTTP_PORT"
echo "gRPC Port: $GRPC_PORT"
echo ""
echo "Next steps:"
echo "1. Implement business logic in internal/service/"
echo "2. Add HTTP handlers in internal/handlers/http/"
echo "3. Add gRPC handlers in internal/handlers/grpc/"
echo "4. Define proper models in internal/models/"
echo "5. Implement repository in internal/repository/"
EOF

chmod +x scripts/create-service.sh
