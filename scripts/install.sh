#!/bin/sh
# Installs the latest WaveTrack release on macOS or Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/lucasrafaldini/wavetrack/main/scripts/install.sh | sh
#
# Options (after `sh -s --` when piping):
#   --version 3.1.0    install a specific release instead of the latest
#   --prefix DIR       install the binary into DIR (default: /usr/local/bin if writable,
#                      otherwise ~/.local/bin)
#
# The download is checked against the release's checksums.txt before anything is installed.
# Nothing runs as root: capture permission is explained at the end, not granted silently.
set -eu

REPO="lucasrafaldini/wavetrack"
VERSION=""
PREFIX=""

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2#v}"; shift 2 ;;
    --prefix) PREFIX="$2"; shift 2 ;;
    -h|--help) sed -n '2,13p' "$0"; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

say() { printf '%s\n' "$*"; }
fail() { printf 'wavetrack install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "needs $1"; }

need curl
need tar
need uname

case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux) OS=linux ;;
  *) fail "this script supports macOS and Linux; on Windows use install.ps1" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) fail "no prebuilt binary for $(uname -m); build from source with 'make build'" ;;
esac

if [ -z "$VERSION" ]; then
  # The latest release page redirects to .../releases/tag/vX.Y.Z.
  url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") \
    || fail "couldn't reach GitHub"
  VERSION="${url##*/v}"
  [ -n "$VERSION" ] && [ "$VERSION" != "$url" ] || fail "couldn't find the latest release"
fi

NAME="wavetrack_${VERSION}_${OS}_${ARCH}"
BASE="https://github.com/$REPO/releases/download/v$VERSION"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

say "Downloading WaveTrack $VERSION for $OS/$ARCH"
curl -fsSL -o "$TMP/$NAME.tar.gz" "$BASE/$NAME.tar.gz" || fail "no release asset $NAME.tar.gz"
curl -fsSL -o "$TMP/checksums.txt" "$BASE/checksums.txt" || fail "couldn't download checksums.txt"

expected=$(grep " $NAME.tar.gz\$" "$TMP/checksums.txt" | cut -d' ' -f1)
[ -n "$expected" ] || fail "$NAME.tar.gz isn't listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$TMP/$NAME.tar.gz" | cut -d' ' -f1)
else
  actual=$(shasum -a 256 "$TMP/$NAME.tar.gz" | cut -d' ' -f1)
fi
[ "$expected" = "$actual" ] || fail "checksum mismatch: expected $expected, got $actual"
say "Checksum verified"

tar -xzf "$TMP/$NAME.tar.gz" -C "$TMP"

if [ -z "$PREFIX" ]; then
  if [ -w /usr/local/bin ]; then PREFIX=/usr/local/bin; else PREFIX="$HOME/.local/bin"; fi
fi
mkdir -p "$PREFIX"
install -m 755 "$TMP/$NAME/wavetrack" "$PREFIX/wavetrack" 2>/dev/null \
  || { cp "$TMP/$NAME/wavetrack" "$PREFIX/wavetrack" && chmod 755 "$PREFIX/wavetrack"; } \
  || fail "couldn't write to $PREFIX; pass --prefix or run with sudo"
say "Installed $PREFIX/wavetrack"

case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) say ""; say "  $PREFIX isn't on your PATH. Add it, e.g.:"; say "    echo 'export PATH=\"$PREFIX:\$PATH\"' >> ~/.profile" ;;
esac

say ""
say "Next steps"
say "  wavetrack serve --demo     try it now with a simulated office, no permissions needed"
say "  wavetrack doctor           check capture permission and configuration"
if [ "$OS" = linux ]; then
  if ! (ldconfig -p 2>/dev/null | grep -q libpcap) && [ ! -e /usr/lib/libpcap.so.1 ]; then
    say "  sudo apt install libpcap0.8   (or your distribution's libpcap package)"
  fi
  say "  sudo setcap cap_net_raw,cap_net_admin=eip $PREFIX/wavetrack   capture without sudo"
else
  say "  brew install --cask wireshark-chmodbpf   capture without sudo (log out and back in)"
fi
