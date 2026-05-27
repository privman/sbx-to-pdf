#!/usr/bin/env bash
set -euo pipefail

REPO="privman/sbx-to-pdf"
INSTALL_DIR="/usr/local/bin"
BINARY="sbx2pdf"

# Detect OS.
OS="$(uname -s)"
case "$OS" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture.
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)       arch="amd64" ;;
  amd64)        arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

asset="${BINARY}-${os}-${arch}"

# Fetch latest release tag from GitHub API.
echo "Fetching latest release..."
tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)"

if [ -z "$tag" ]; then
  echo "Error: could not determine latest release." >&2
  exit 1
fi

url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
echo "Downloading ${BINARY} ${tag} (${os}/${arch})..."

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

curl -fsSL -o "${tmpdir}/${BINARY}" "$url"
chmod +x "${tmpdir}/${BINARY}"

# Install — use sudo only if needed.
if [ -w "$INSTALL_DIR" ]; then
  mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} ${tag} to ${INSTALL_DIR}/${BINARY}"
