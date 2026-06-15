#!/bin/bash
set -e

echo "Building go-chrome..."

export GO111MODULE=on

# Ensure JetBrains-style CJK UI font is present. The .ttf file is not committed
# to the repo; it is fetched on first build via a China mirror. Only the
# Regular variant is embedded at runtime; Fyne synthesizes bold from it.
FONT_DIR="assets/fonts"
FONT_REGULAR="$FONT_DIR/MapleMono-CN-Regular.ttf"
if [[ ! -f "$FONT_REGULAR" ]]; then
    echo "Maple Mono CN Regular font missing; downloading from China mirror..."
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    ZIP_URL="https://gh-proxy.com/https://github.com/subframe7536/maple-font/releases/download/v7.9/MapleMono-CN.zip"
    ZIP_FILE="$TMP_DIR/MapleMono-CN.zip"
    if ! curl -fsSL -o "$ZIP_FILE" "$ZIP_URL"; then
        echo "ERROR: failed to download Maple Mono CN font from $ZIP_URL"
        echo "Please place the following file manually in $FONT_DIR:"
        echo "  MapleMono-CN-Regular.ttf"
        exit 1
    fi
    unzip -q "$ZIP_FILE" -d "$TMP_DIR"
    cp "$TMP_DIR/MapleMono-CN-Regular.ttf" "$FONT_REGULAR"
    echo "Font downloaded to $FONT_REGULAR"
fi

echo "Building executable..."
# Offline-friendly: do not run go mod tidy/download here.
# Prepare dependencies ahead of time with: go mod download
go build -mod=readonly -ldflags "-s -w" -o go-chrome ./cmd/go-chrome

echo "Build complete: go-chrome"
