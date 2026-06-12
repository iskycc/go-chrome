#!/bin/bash
set -e

echo "Building go-chrome..."

export GO111MODULE=on

# Ensure JetBrains-style CJK UI fonts are present. The .ttf files are not
# committed to the repo; they are fetched on first build via a China mirror.
FONT_DIR="assets/fonts"
FONT_REGULAR="$FONT_DIR/MapleMono-CN-Regular.ttf"
FONT_MEDIUM="$FONT_DIR/MapleMono-CN-Medium.ttf"
if [[ ! -f "$FONT_REGULAR" || ! -f "$FONT_MEDIUM" ]]; then
    echo "Maple Mono CN fonts missing; downloading from China mirror..."
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    ZIP_URL="https://gh-proxy.com/https://github.com/subframe7536/maple-font/releases/download/v7.9/MapleMono-CN.zip"
    ZIP_FILE="$TMP_DIR/MapleMono-CN.zip"
    if ! curl -fsSL -o "$ZIP_FILE" "$ZIP_URL"; then
        echo "ERROR: failed to download Maple Mono CN fonts from $ZIP_URL"
        echo "Please place the following files manually in $FONT_DIR:"
        echo "  MapleMono-CN-Regular.ttf"
        echo "  MapleMono-CN-Medium.ttf"
        exit 1
    fi
    unzip -q "$ZIP_FILE" -d "$TMP_DIR"
    cp "$TMP_DIR/MapleMono-CN-Regular.ttf" "$FONT_REGULAR"
    cp "$TMP_DIR/MapleMono-CN-Medium.ttf" "$FONT_MEDIUM"
    echo "Fonts downloaded to $FONT_DIR"
fi

echo "Building executable..."
# Offline-friendly: do not run go mod tidy/download here.
# Prepare dependencies ahead of time with: go mod download
go build -mod=readonly -ldflags "-s -w" -o go-chrome ./cmd/go-chrome

echo "Build complete: go-chrome"
