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

ASSET_NAME="${PROJECT_NAME}-${OS}-${ARCH}.tar.gz"
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
TAR_PATH="${TMP_DIR}/${ASSET_NAME}"

echo "⬇️  Downloading $ASSET_NAME..."
curl -sL "$DOWNLOAD_URL" -o "$TAR_PATH"

echo "📦 Extracting..."
tar -xzf "$TAR_PATH" -C "$TMP_DIR"

# Find the binary file
BIN_CANDIDATE=$(find "$TMP_DIR" -name "$PROJECT_NAME" -type f | head -n 1)

if [ -z "$BIN_CANDIDATE" ]; then
  echo "❌ Binary not found inside archive"
  exit 1
fi

# Verify binary is executable
if [ ! -x "$BIN_CANDIDATE" ]; then
  chmod +x "$BIN_CANDIDATE"
fi

# Verify binary compatibility (especially important for Docker)
if ! ldd "$BIN_CANDIDATE" >/dev/null 2>&1; then
  echo "⚠️  Warning: Binary might not be compatible with this system"
  echo "ℹ️  If you see 'not found' errors, you may need to install dependencies"
fi

echo "📁 Installing $BIN_CANDIDATE to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
mv -f "$BIN_CANDIDATE" "$INSTALL_DIR/${PROJECT_NAME}"

# Verify installation
if command -v "$INSTALL_DIR/${PROJECT_NAME}" >/dev/null; then
  echo "✅ Successfully installed to ${INSTALL_DIR}/${PROJECT_NAME}"
  echo "ℹ️  Version: $($INSTALL_DIR/${PROJECT_NAME} --version 2>/dev/null || echo "unknown")"
else
  echo "❌ Installation failed - binary not found in PATH"
  exit 1
fi