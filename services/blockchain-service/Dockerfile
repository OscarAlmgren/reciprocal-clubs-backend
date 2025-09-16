# Multi-stage Dockerfile for Blockchain Service
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    build-base

# Set working directory to the full project
WORKDIR /build

# Create non-root user for build
RUN adduser -D -g '' appuser

# Copy entire project for build (this ensures all dependencies are available)
COPY . .

# Build from project root using Go workspace
WORKDIR /build

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o blockchain-service \
    ./services/blockchain-service/cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create working directory
WORKDIR /root/

# Import user and group files for security
RUN adduser -D -g '' appuser

# Copy the binary from builder stage
COPY --from=builder /build/blockchain-service .

# Change ownership to non-root user
RUN chown appuser:appuser blockchain-service

# Use non-root user
USER appuser

# Expose ports (HTTP and gRPC)
EXPOSE 8087 9097

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./blockchain-service health || exit 1

# Set entrypoint
ENTRYPOINT ["./blockchain-service"]

# Metadata labels
LABEL maintainer="Blockchain Service Team"
LABEL version="1.0.0"
LABEL description="Blockchain Service for Reciprocal Clubs Backend"
LABEL org.opencontainers.image.source="https://github.com/reciprocal-clubs/backend"
LABEL org.opencontainers.image.documentation="https://docs.reciprocal-clubs.com/blockchain-service"
LABEL org.opencontainers.image.vendor="Reciprocal Clubs"
LABEL org.opencontainers.image.licenses="MIT"
