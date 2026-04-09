#!/bin/sh
# Install Haystack's fork of Entire CLI
# Usage: curl -fsSL https://raw.githubusercontent.com/haystackeditor/cli/main/install.sh | sh
set -e

VERSION="v0.5.3-haystack.1"
BASE_URL="https://github.com/haystackeditor/cli/releases/download/${VERSION}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

BINARY="entire_${OS}_${ARCH}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "Installing Entire CLI ${VERSION} (${OS}/${ARCH})..."

if [ ! -w "$INSTALL_DIR" ]; then
  echo "Need sudo to write to ${INSTALL_DIR}"
  sudo curl -fsSL "${BASE_URL}/${BINARY}" -o "${INSTALL_DIR}/entire"
  sudo chmod +x "${INSTALL_DIR}/entire"
else
  curl -fsSL "${BASE_URL}/${BINARY}" -o "${INSTALL_DIR}/entire"
  chmod +x "${INSTALL_DIR}/entire"
fi

echo "Installed: $(entire --version)"
echo ""
echo "Transcripts are automatically compacted before commit (full.jsonl ~30-200KB instead of ~7MB)."
