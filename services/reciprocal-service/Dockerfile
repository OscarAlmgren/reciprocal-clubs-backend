# Multi-stage Dockerfile for Reciprocal Service
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    build-base \
    protobuf \
    protobuf-dev

# Set working directory to the full project
WORKDIR /build

# Create non-root user for build
RUN adduser -D -g '' appuser

# Copy entire project for build (this ensures all dependencies are available)
COPY . .

# Change to service directory
WORKDIR /build/services/reciprocal-service

# Install protobuf tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Download service dependencies
RUN go mod download || echo "Some dependencies may be missing, continuing..."

# Generate protobuf code if proto files exist
RUN if [ -d "api/proto" ]; then \
        mkdir -p api/proto && \
        find api/proto -name "*.proto" -exec protoc \
            --go_out=. \
            --go_opt=paths=source_relative \
            --go-grpc_out=. \
            --go-grpc_opt=paths=source_relative \
            --proto_path=api/proto \
            {} + \
    ; fi

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o reciprocal-service \
    cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create working directory
WORKDIR /root/

# Import user and group files for security
RUN adduser -D -g '' appuser

# Copy the binary from builder stage
COPY --from=builder /build/services/reciprocal-service/reciprocal-service .

# Change ownership to non-root user
RUN chown appuser:appuser reciprocal-service

# Use non-root user
USER appuser

# Expose ports (HTTP and gRPC)
EXPOSE 8083 9083

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./reciprocal-service health || exit 1

# Set entrypoint
ENTRYPOINT ["./reciprocal-service"]

# Metadata labels
LABEL maintainer="Reciprocal Service Team"
LABEL version="1.0.0"
LABEL description="Reciprocal Service for Reciprocal Clubs Backend"
LABEL org.opencontainers.image.source="https://github.com/reciprocal-clubs/backend"
LABEL org.opencontainers.image.documentation="https://docs.reciprocal-clubs.com/reciprocal-service"
LABEL org.opencontainers.image.vendor="Reciprocal Clubs"
LABEL org.opencontainers.image.licenses="MIT"