#!/bin/bash

# Script to test all Docker builds

set -e

PROJECT_ROOT=$(pwd)
BUILD_SUCCESS=()
BUILD_FAILED=()

# Use Docker or Podman based on environment variable
DOCKER_CMD=${DOCKER_CMD:-podman}

echo "Testing Docker builds for all services using $DOCKER_CMD..."

# Find all services with Dockerfiles
services=$(find services -name "Dockerfile" -exec dirname {} \; | sort)

for service_dir in $services; do
    service_name=$(basename "$service_dir")
    echo ""
    echo "=================================================="
    echo "Building $service_name with $DOCKER_CMD..."
    echo "=================================================="
    
    if $DOCKER_CMD build -f "$service_dir/Dockerfile" -t "reciprocal-$service_name:test" . ; then
        echo "✅ SUCCESS: $service_name built successfully"
        BUILD_SUCCESS+=("$service_name")
    else
        echo "❌ FAILED: $service_name failed to build"
        BUILD_FAILED+=("$service_name")
    fi
done

echo ""
echo "=================================================="
echo "BUILD SUMMARY"
echo "=================================================="

echo "✅ Successful builds (${#BUILD_SUCCESS[@]}):"
for service in "${BUILD_SUCCESS[@]}"; do
    echo "  - $service"
done

if [ ${#BUILD_FAILED[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed builds (${#BUILD_FAILED[@]}):"
    for service in "${BUILD_FAILED[@]}"; do
        echo "  - $service"
    done
    echo ""
    echo "Some builds failed. Check the logs above for details."
    exit 1
else
    echo ""
    echo "🎉 All builds completed successfully!"
    exit 0
fi