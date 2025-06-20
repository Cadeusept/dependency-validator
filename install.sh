#!/usr/bin/env bash
set -euo pipefail

# === CONFIGURATION ===
VERSION="${1:-latest}"
INSTALL_DIR="${2:-/usr/local/bin}"
PROJECT_NAME="dependency-validator"
REPO="Cadeusept/dependency-validator"

# Determine OS and Arch
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  armv7*) ARCH="arm" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Construct asset name
ASSET_NAME="${PROJECT_NAME}-${OS}-${ARCH}.zip"
echo "Looking for release asset: $ASSET_NAME"

# GitHub API URL
API="https://api.github.com/repos/$REPO/releases"
if [ "$VERSION" = "latest" ]; then
  URL="$API/latest"
else
  URL="$API/tags/$VERSION"
fi

# Fetch download URL
DOWNLOAD_URL=$(curl -sL "$URL" \
  | grep "browser_download_url" \
  | grep "\"$ASSET_NAME\"" \
  | cut -d '"' -f4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "❌ Error: asset $ASSET_NAME not found for release $VERSION"
  exit 1
fi

echo "Downloading $DOWNLOAD_URL..."
TMP="$(mktemp -d)"
ZIP_PATH="$TMP/$ASSET_NAME"
curl -sL "$DOWNLOAD_URL" -o "$ZIP_PATH"

echo "Unzipping..."
unzip -q "$ZIP_PATH" -d "$TMP"

BIN_PATH="$TMP/$PROJECT_NAME"
if [ ! -x "$BIN_PATH" ]; then
  echo "❌ Error: $PROJECT_NAME binary not found in zip"
  exit 1
fi

echo "Installing to $INSTALL_DIR..."
sudo install -m 0755 "$BIN_PATH" "$INSTALL_DIR/$PROJECT_NAME"

echo "✅ Installed $PROJECT_NAME $("$INSTALL_DIR/$PROJECT_NAME" version)"