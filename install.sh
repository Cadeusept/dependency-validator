#!/usr/bin/env bash
set -euo pipefail

# === CONFIGURATION ===
REPO="Cadeusept/dependency-validator"
PROJECT_NAME="dependency-validator"
VERSION="${1:-latest}"
INSTALL_DIR="${2:-/usr/local/bin}"

# Detect OS and architecture
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64) ARCH="arm64" ;;
  armv7l) ARCH="arm" ;;
  *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

ASSET_NAME="${PROJECT_NAME}-${OS}-${ARCH}.zip"
echo "➡️  Installing ${PROJECT_NAME} for ${OS}-${ARCH}..."

# Fetch release metadata
if [ "$VERSION" = "latest" ]; then
  API_URL="https://api.github.com/repos/${REPO}/releases/latest"
else
  API_URL="https://api.github.com/repos/${REPO}/releases/tags/${VERSION}"
fi

DOWNLOAD_URL=$(curl -sL "$API_URL" \
  | grep "browser_download_url" \
  | grep "$ASSET_NAME" \
  | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "❌ Could not find asset: $ASSET_NAME in release $VERSION"
  exit 1
fi

TMP_DIR="$(mktemp -d)"
ZIP_PATH="${TMP_DIR}/${ASSET_NAME}"

echo "⬇️  Downloading $ASSET_NAME..."
curl -sL "$DOWNLOAD_URL" -o "$ZIP_PATH"

echo "📦 Extracting..."
unzip -q "$ZIP_PATH" -d "$TMP_DIR"

# Find the binary file (first non-zip file with exec permission or no extension)

BIN_CANDIDATE=$(find "$TMP_DIR" -maxdepth 1 -type f ! -name "*.zip" | head -n 1)

if [ ! -x "$BIN_CANDIDATE" ]; then
  chmod +x "$BIN_CANDIDATE" 2>/dev/null || true
fi

if [ ! -f "$BIN_CANDIDATE" ]; then
  echo "❌ Binary not found inside zip file"
  exit 1
fi

echo "📁 Installing $BIN_CANDIDATE to $INSTALL_DIR..."
mv "$BIN_CANDIDATE" "$INSTALL_DIR/${PROJECT_NAME}"
chmod +x "$INSTALL_DIR/${PROJECT_NAME}"

echo "✅ Installed to ${INSTALL_DIR}/${PROJECT_NAME}"