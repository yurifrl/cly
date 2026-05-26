#!/usr/bin/env bash
# Build cly-notifier.app universal Swift binary, sign it, and tar it.
#
# Output: pkg/notify/assets/cly-notifier.app.tar.gz
#
# Codesign identity comes from the CLY_NOTIFIER_SIGN_ID env var, populated
# by `task envs:op` (which resolves .env.op via 1Password) or by
# `cly update` (which runs `op inject` to a temp file). When unset,
# falls back to ad-hoc signature `-` (local dev only, not distributable).
set -euo pipefail

if [[ "$(uname)" != "Darwin" ]]; then
    echo "build.sh: skipping (non-darwin host: $(uname))" >&2
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$SCRIPT_DIR/Sources"
BUILD_DIR="$SCRIPT_DIR/.build"
APP_DIR="$BUILD_DIR/cly-notifier.app"
ASSETS_DIR="$PKG_DIR/assets"
TARBALL="$ASSETS_DIR/cly-notifier.app.tar.gz"

mkdir -p "$BUILD_DIR" "$ASSETS_DIR"
rm -rf "$APP_DIR"
mkdir -p "$APP_DIR/Contents/MacOS"

SOURCES=( "$SRC_DIR/main.swift" "$SRC_DIR/Notifier.swift" "$SRC_DIR/Socket.swift" )

echo "build.sh: compiling arm64..."
swiftc -O -target arm64-apple-macos11 \
    -framework Foundation -framework UserNotifications \
    -o "$BUILD_DIR/cly-notifier-arm64" "${SOURCES[@]}"

echo "build.sh: compiling x86_64..."
swiftc -O -target x86_64-apple-macos11 \
    -framework Foundation -framework UserNotifications \
    -o "$BUILD_DIR/cly-notifier-x86_64" "${SOURCES[@]}"

echo "build.sh: lipo universal..."
lipo -create -output "$APP_DIR/Contents/MacOS/cly-notifier" \
    "$BUILD_DIR/cly-notifier-arm64" \
    "$BUILD_DIR/cly-notifier-x86_64"

cp "$SCRIPT_DIR/Info.plist" "$APP_DIR/Contents/Info.plist"

# Generate .icns icon from the embedded PNG so the notification has an icon.
ICON_PNG="$PKG_DIR/assets/icon.png"
if [[ -f "$ICON_PNG" ]]; then
    echo "build.sh: generating icon.icns..."
    mkdir -p "$APP_DIR/Contents/Resources"
    ICONSET="$BUILD_DIR/icon.iconset"
    rm -rf "$ICONSET"
    mkdir -p "$ICONSET"
    for size in 16 32 64 128 256 512; do
        sips -z $size $size "$ICON_PNG" --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
        # Retina @2x variants (skip for 512 which is the cap)
        if [[ $size -lt 512 ]]; then
            dbl=$((size * 2))
            sips -z $dbl $dbl "$ICON_PNG" --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
        fi
    done
    iconutil -c icns -o "$APP_DIR/Contents/Resources/icon.icns" "$ICONSET"
    cp "$ICON_PNG" "$APP_DIR/Contents/Resources/icon.png"
    rm -rf "$ICONSET"
fi

SIGN_ID="${CLY_NOTIFIER_SIGN_ID:-}"
if [[ -n "$SIGN_ID" ]]; then
    echo "build.sh: codesigning with Developer ID..."
    codesign --force --sign "$SIGN_ID" \
        --options runtime --timestamp \
        "$APP_DIR"
    codesign --verify --deep --strict "$APP_DIR"
else
    echo "build.sh: CLY_NOTIFIER_SIGN_ID not set; using ad-hoc signature (local dev only)"
    echo "         (run 'task envs:op' to populate .env from .env.op)"
    codesign --force --sign - "$APP_DIR"
    codesign --verify --deep --strict "$APP_DIR" || true
fi

echo "build.sh: creating tarball..."
tar -C "$BUILD_DIR" -czf "$TARBALL" "cly-notifier.app"

rm -f "$BUILD_DIR/cly-notifier-arm64" "$BUILD_DIR/cly-notifier-x86_64"

SIZE="$(du -h "$TARBALL" | cut -f1)"
echo "build.sh: wrote $TARBALL ($SIZE)"
