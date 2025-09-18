#!/bin/bash

# Generate protobuf files for notification service

set -e

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "protoc could not be found. Please install Protocol Buffers compiler."
    echo "On macOS: brew install protobuf"
    echo "On Ubuntu: sudo apt install protobuf-compiler"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "protoc-gen-go could not be found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "protoc-gen-go-grpc could not be found. Installing..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Create output directory
mkdir -p proto

# Generate Go files from proto
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    proto/notification.proto

echo "Protobuf files generated successfully!"