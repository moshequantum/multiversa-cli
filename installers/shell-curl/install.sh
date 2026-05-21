#!/usr/bin/env sh
# Multiversa CLI installer
# Usage: curl -sSL https://raw.githubusercontent.com/moshequantum/multiversa-cli/main/installers/shell-curl/install.sh | sh
#
# Honors $MULTIVERSA_VERSION, $MULTIVERSA_INSTALL_DIR.

set -eu

REPO="moshequantum/multiversa-cli"
INSTALL_DIR="${MULTIVERSA_INSTALL_DIR:-/usr/local/bin}"
VERSION="${MULTIVERSA_VERSION:-latest}"

CHARTREUSE="\033[38;5;191m"
IVORY="\033[38;5;230m"
DIM="\033[2m"
RESET="\033[0m"

say() { printf "%b%s%b\n" "$IVORY" "$1" "$RESET"; }
accent() { printf "%b%s%b\n" "$CHARTREUSE" "$1" "$RESET"; }
dim() { printf "%b%s%b\n" "$DIM" "$1" "$RESET"; }

detect_os() {
  case "$(uname -s)" in
    Darwin) echo darwin ;;
    Linux)  echo linux ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) echo "Unsupported arch: $(uname -m)" >&2; exit 1 ;;
  esac
}

accent "Multiversa CLI installer"
dim "  Orchestrates the curated agentic stack. Attribution at https://github.com/$REPO/blob/main/CREDITS.md"
echo

OS=$(detect_os)
ARCH=$(detect_arch)
say "Detected: $OS/$ARCH"

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | head -n1 | cut -d'"' -f4)
fi
[ -n "$VERSION" ] || { echo "Could not resolve latest version." >&2; exit 1; }
say "Version: $VERSION"

ASSET="multiversa_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

say "Downloading $ASSET..."
curl -fL "$URL" -o "$TMP/$ASSET"

say "Extracting..."
tar -xzf "$TMP/$ASSET" -C "$TMP"

if [ ! -w "$INSTALL_DIR" ]; then
  say "Installing to $INSTALL_DIR (sudo)..."
  sudo install -m 0755 "$TMP/multiversa" "$INSTALL_DIR/multiversa"
else
  say "Installing to $INSTALL_DIR..."
  install -m 0755 "$TMP/multiversa" "$INSTALL_DIR/multiversa"
fi

echo
accent "Installed: $(command -v multiversa)"
dim "  Run: multiversa init"
echo
