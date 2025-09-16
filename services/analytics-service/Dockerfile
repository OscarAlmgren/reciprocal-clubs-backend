# =========================================================================
# Stage 1: Build Stage
# =========================================================================
# Use the requested Go version on a minimal Alpine base.
FROM golang:1.25-alpine AS builder

# Set the working directory within the container.
WORKDIR /app

# Install build dependencies.
# 'git' is often needed to fetch Go modules from version control systems.
RUN apk add --no-cache git

# Copy only the necessary files to download dependencies.
# This step is cached and only re-runs if the module files change.
COPY go.mod go.sum go.work* ./
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Compile the application into a statically linked binary.
# - CGO_ENABLED=0 ensures a static build without C library dependencies.
# -ldflags="-w -s" strips debug symbols, significantly reducing the binary size.
RUN CGO_ENABLED=0 go build \
    -ldflags='-w -s' \
    -o /bin/analytics-service \
    ./services/analytics-service/cmd/main.go


# =========================================================================
# Stage 2: Final Stage
# =========================================================================
# Use a minimal, secure base image.
FROM alpine:latest

# Install essential packages.
# - ca-certificates: For making secure HTTPS connections.
# - tzdata: For time zone information.
# - curl: A lightweight tool for the health check.
RUN apk add --no-cache ca-certificates tzdata curl

# Create a dedicated, non-root user and group for the application.
# Using static IDs (e.g., 1001) is a good practice for predictable permissions.
RUN addgroup -S appgroup -g 1001 && \
    adduser -S appuser -u 1001 -G appgroup

# Set the application's working directory.
WORKDIR /app

# Copy the compiled binary from the builder stage and set ownership in one step.
COPY --from=builder --chown=appuser:appgroup /bin/analytics-service .

# Switch to the non-root user for security.
USER appuser

# Expose the application's ports.
EXPOSE 8086 9096

# Define metadata for the container image.
LABEL maintainer="Analytics Service Team"
LABEL version="1.0.0"
LABEL description="Analytics Service for Reciprocal Clubs Backend"
LABEL org.opencontainers.image.source="https://github.com/reciprocal-clubs/backend"
LABEL org.opencontainers.image.documentation="https://docs.reciprocal-clubs.com/analytics-service"
LABEL org.opencontainers.image.vendor="Reciprocal Clubs"
LABEL org.opencontainers.image.licenses="MIT"

# Add a health check to ensure the service is responsive.
# The '-f' or '--fail' flag in curl causes it to return a non-zero exit code on server errors (HTTP 4xx or 5xx).
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8086/health || exit 1

# Set the command to run when the container starts.
ENTRYPOINT ["./analytics-service"]
