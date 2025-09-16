#!/bin/bash

# Script to tidy all Go modules in the project

set -e

PROJECT_ROOT=$(pwd)

echo "Tidying shared packages..."
for pkg_dir in $(find pkg/shared -name "go.mod" -exec dirname {} \;); do
    echo "Tidying $pkg_dir..."
    cd "$PROJECT_ROOT/$pkg_dir"
    go mod tidy
    go mod download
done

cd "$PROJECT_ROOT"

echo "Tidying services..."
for service_dir in $(find services -name "go.mod" -exec dirname {} \;); do
    echo "Tidying $service_dir..."
    cd "$PROJECT_ROOT/$service_dir"
    go mod tidy
    go mod download
done

cd "$PROJECT_ROOT"

echo "Tidying root module..."
if [ -f "go.mod" ]; then
    go mod tidy
    go mod download
fi

echo "All modules tidied successfully!"