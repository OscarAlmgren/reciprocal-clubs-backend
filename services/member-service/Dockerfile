# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.work go.work.sum ./
COPY services/member-service/go.mod services/member-service/go.sum ./services/member-service/
COPY pkg/ ./pkg/

# Download dependencies
WORKDIR /app/services/member-service
RUN go mod download

# Copy source code
COPY services/member-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o member-service \
    ./cmd/main.go

# Final stage
FROM scratch

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/services/member-service/member-service /member-service

# Set environment
ENV TZ=UTC

# Expose ports
EXPOSE 8082 9092

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/member-service", "healthcheck"]

# Run the application
ENTRYPOINT ["/member-service"]