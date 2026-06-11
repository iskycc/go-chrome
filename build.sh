#!/bin/bash
set -e

echo "Building go-chrome..."

export GO111MODULE=on

echo "Building executable..."
# Offline-friendly: do not run go mod tidy/download here.
# Prepare dependencies ahead of time with: go mod download
go build -mod=readonly -ldflags "-s -w" -o go-chrome ./cmd/go-chrome

echo "Build complete: go-chrome"
