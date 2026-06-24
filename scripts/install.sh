#!/usr/bin/env sh
# Install hass-cli by downloading the matching release tarball from GitHub.
#
#   curl -fsSL https://raw.githubusercontent.com/beclab/hass-cli/main/scripts/install.sh | sh
#
# Environment:
#   HASS_CLI_VERSION   release version to install (default: latest)
#   HASS_CLI_BIN_DIR   install dir (default: /usr/local/bin, else ~/.local/bin)
#   HASS_CLI_DOWNLOAD_MIRROR  base URL to fetch the tarball from instead of GitHub
set -eu

REPO="beclab/hass-cli"
GH_BASE="https://github.com/${REPO}/releases/download"

err() { printf '%s\n' "hass-cli install: $*" >&2; }
die() { err "$*"; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || die "required tool not found: $1"; }
need uname
need tar

# Prefer curl, fall back to wget.
if command -v curl >/dev/null 2>&1; then
  DL="curl -fsSL"
  DL_O="curl -fsSL -o"
elif command -v wget >/dev/null 2>&1; then
  DL="wget -qO-"
  DL_O="wget -qO"
else
  die "need curl or wget"
fi

# Map uname -> goreleaser os/arch (must match .goreleaser.yaml name_template).
os=$(uname -s)
case "$os" in
  Linux)  GOOS=linux ;;
  Darwin) GOOS=darwin ;;
  *) die "unsupported OS: $os (use the npm package or a release archive)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  armv7l|armv7|arm) GOARCH=arm ;;
  *) die "unsupported arch: $arch" ;;
esac

# Resolve version (strip a leading v for the archive, keep v for the tag).
VERSION="${HASS_CLI_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  need_api="https://api.github.com/repos/${REPO}/releases/latest"
  tag=$($DL "$need_api" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
  [ -n "$tag" ] || die "could not resolve latest release tag from GitHub API"
else
  case "$VERSION" in v*) tag="$VERSION" ;; *) tag="v$VERSION" ;; esac
fi
ver=${tag#v}

name="hass-cli-${tag}_${GOOS}_${GOARCH}.tar.gz"
if [ -n "${HASS_CLI_DOWNLOAD_MIRROR:-}" ]; then
  url="${HASS_CLI_DOWNLOAD_MIRROR%/}/${name}"
else
  url="${GH_BASE}/${tag}/${name}"
fi

# Pick an install dir we can actually write to.
if [ -n "${HASS_CLI_BIN_DIR:-}" ]; then
  BIN_DIR="$HASS_CLI_BIN_DIR"
elif [ -w /usr/local/bin ] 2>/dev/null; then
  BIN_DIR=/usr/local/bin
else
  BIN_DIR="$HOME/.local/bin"
fi
mkdir -p "$BIN_DIR"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

err "downloading $url"
$DL_O "$tmp/hass-cli.tar.gz" "$url" || die "download failed: $url"
tar -xzf "$tmp/hass-cli.tar.gz" -C "$tmp" || die "extract failed"
[ -f "$tmp/hass-cli" ] || die "archive did not contain a hass-cli binary"

install -m 0755 "$tmp/hass-cli" "$BIN_DIR/hass-cli" 2>/dev/null || {
  cp "$tmp/hass-cli" "$BIN_DIR/hass-cli" && chmod 0755 "$BIN_DIR/hass-cli";
}

err "installed hass-cli ${ver} to ${BIN_DIR}/hass-cli"
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) err "note: $BIN_DIR is not on your PATH; add it to use 'hass-cli' directly" ;;
esac
err "run 'hass-cli profile login' to configure your Home Assistant connection"
