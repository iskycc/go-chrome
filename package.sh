#!/bin/bash
set -e

echo "Packaging go-chrome..."

# Ensure executable exists
if [ ! -f go-chrome.exe ]; then
    echo "ERROR: go-chrome.exe not found. Run build first."
    exit 1
fi

PKGDIR="go-chrome-release"
ZIPNAME="go-chrome-release.zip"

rm -rf "$PKGDIR" "$ZIPNAME"
mkdir -p "$PKGDIR"

cp go-chrome.exe "$PKGDIR/"
cp README.md FAQ.md USER_GUIDE.md "$PKGDIR/" 2>/dev/null || true

mkdir -p "$PKGDIR/data/flows"
mkdir -p "$PKGDIR/logs"
mkdir -p "$PKGDIR/chrome"

zip -r "$ZIPNAME" "$PKGDIR"
rm -rf "$PKGDIR"

echo "Package created: $ZIPNAME"
