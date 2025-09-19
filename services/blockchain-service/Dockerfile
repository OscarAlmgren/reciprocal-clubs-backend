# =========================================================================
# Multi-stage Dockerfile for Blockchain Service
# =========================================================================

# =========================================================================
# Stage 1: Build Stage
# =========================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    build-base \
    protobuf \
    protobuf-dev

# Set working directory to project root
WORKDIR /build

# Create non-root user for build
RUN adduser -D -g '' appuser

# Copy entire project for workspace support
COPY . .

# Install protobuf tools if needed
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Change to service directory
WORKDIR /build/services/blockchain-service

# Download dependencies (workspace-aware)
RUN go mod download || echo "Some dependencies may be missing, continuing..."

# Generate protobuf code if proto files exist
RUN if [ -d "api/proto" ] || [ -d "proto" ]; then \
        find . -name "*.proto" -exec protoc \
            --go_out=. \
            --go_opt=paths=source_relative \
            --go-grpc_out=. \
            --go-grpc_opt=paths=source_relative \
            --proto_path=. \
            {} + 2>/dev/null || true \
    ; fi

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o blockchain-service \
    cmd/main.go

# =========================================================================
# Stage 2: Final Stage
# =========================================================================
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# Create non-root user with consistent UID/GID
RUN addgroup -S appgroup -g 1001 && \
    adduser -S appuser -u 1001 -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage with proper ownership
COPY --from=builder --chown=appuser:appgroup /build/services/blockchain-service/blockchain-service ./

# Switch to non-root user
USER appuser

# Expose service ports (HTTP and gRPC)
EXPOSE 8084 9094

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8084/health || exit 1

# Metadata labels
LABEL maintainer="Blockchain Service Team"
LABEL version="1.0.0"
LABEL description="Blockchain Service for Reciprocal Clubs Backend"
LABEL org.opencontainers.image.source="https://github.com/reciprocal-clubs/backend"
LABEL org.opencontainers.image.documentation="https://docs.reciprocal-clubs.com/blockchain-service"
LABEL org.opencontainers.image.vendor="Reciprocal Clubs"
LABEL org.opencontainers.image.licenses="MIT"

# Set entrypoint
ENTRYPOINT ["./blockchain-service"]
