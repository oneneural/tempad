#!/usr/bin/env bash
# TEMPAD installer — downloads the latest Go release from GitHub.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/oneneural/tempad/main/scripts/install.sh | bash
#   curl -sSL https://raw.githubusercontent.com/oneneural/tempad/main/scripts/install.sh | bash -s -- --version go/v1.0.0
#   curl -sSL https://raw.githubusercontent.com/oneneural/tempad/main/scripts/install.sh | bash -s -- --dir /usr/local/bin

set -euo pipefail

REPO="oneneural/tempad"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION=""

# Parse arguments.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    *)         echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# Detect OS and architecture.
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS (use Linux or macOS)" >&2; exit 1 ;;
esac

# Resolve latest version if not specified.
if [[ -z "$VERSION" ]]; then
  echo "Fetching latest release..."
  VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

  if [[ -z "$VERSION" ]]; then
    echo "Error: could not determine latest release." >&2
    echo "Specify a version with: --version go/v1.0.0" >&2
    exit 1
  fi
fi

# Strip go/ prefix for archive naming.
CLEAN_VERSION="${VERSION#go/}"
CLEAN_VERSION="${CLEAN_VERSION#v}"

ARCHIVE="tempad_${CLEAN_VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

echo "Installing TEMPAD ${VERSION} (${OS}/${ARCH})..."

# Create temp directory.
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# Download archive and checksums.
echo "Downloading ${URL}..."
curl -sSL -o "${TMP_DIR}/${ARCHIVE}" "$URL"
curl -sSL -o "${TMP_DIR}/checksums.txt" "$CHECKSUMS_URL"

# Verify checksum.
cd "$TMP_DIR"
if command -v sha256sum &>/dev/null; then
  grep "$ARCHIVE" checksums.txt | sha256sum --check --quiet
  echo "Checksum verified."
elif command -v shasum &>/dev/null; then
  grep "$ARCHIVE" checksums.txt | shasum -a 256 --check --quiet
  echo "Checksum verified."
else
  echo "Warning: sha256sum not found, skipping checksum verification."
fi

# Extract and install.
tar xzf "$ARCHIVE"
BINARY_PATH="${TMP_DIR}/tempad"

# Handle GoReleaser directory wrapping.
if [[ ! -f "$BINARY_PATH" ]]; then
  BINARY_PATH=$(find "$TMP_DIR" -name "tempad" -type f -perm +111 | head -1)
fi

if [[ ! -f "$BINARY_PATH" ]]; then
  echo "Error: tempad binary not found in archive." >&2
  exit 1
fi

# Install binary.
mkdir -p "$INSTALL_DIR"
if [[ -w "$INSTALL_DIR" ]]; then
  cp "$BINARY_PATH" "${INSTALL_DIR}/tempad"
  chmod +x "${INSTALL_DIR}/tempad"
else
  echo "Requires sudo to install to ${INSTALL_DIR}"
  sudo cp "$BINARY_PATH" "${INSTALL_DIR}/tempad"
  sudo chmod +x "${INSTALL_DIR}/tempad"
fi

echo ""
echo "TEMPAD ${VERSION} installed to ${INSTALL_DIR}/tempad"
echo ""
echo "Get started:"
echo "  tempad init              # Create config"
echo "  tempad --help            # Show usage"
echo "  tempad --version         # Show version"
