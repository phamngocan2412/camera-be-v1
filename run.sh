#!/bin/bash

# Script to run the Camera Security API backend

echo "🚀 Starting Camera Security API Backend..."

# Check if swagger docs exist, if not generate them
if [ ! -f "docs/docs.go" ]; then
    echo "📝 Generating Swagger documentation..."
    go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs
fi

# Run the API server
echo "🌐 Starting server..."
go run cmd/api/main.go
