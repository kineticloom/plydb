#!/bin/sh
# PlyDB installer — downloads the latest release binary for your platform.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kineticloom/plydb/main/install.sh | sh
#
# Environment variables:
#   PLYDB_INSTALL_DIR  — where to place the binary (default: ~/.local/bin)
#   PLYDB_VERSION      — version tag to install (default: latest)

{

set -e

REPO="kineticloom/plydb"
BINARY="plydb"

# ── helpers ──────────────────────────────────────────────────────────────────

die() {
  echo "error: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

# ── detect platform ─────────────────────────────────────────────────────────

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux"  ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) die "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) die "unsupported architecture: $arch" ;;
  esac
}

# On Apple Silicon running under Rosetta 2, uname -m reports x86_64.
# Detect the true architecture so we download the native arm64 binary.
detect_arch_darwin() {
  arch="$(detect_arch)"
  if [ "$arch" = "amd64" ]; then
    if sysctl -n sysctl.proc_translated 2>/dev/null | grep -q 1; then
      echo "arm64"
      return
    fi
  fi
  echo "$arch"
}

# ── resolve version ─────────────────────────────────────────────────────────

resolve_version() {
  if [ -n "$PLYDB_VERSION" ]; then
    echo "$PLYDB_VERSION"
    return
  fi

  need curl
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')

  [ -n "$tag" ] || die "could not determine latest release (GitHub API rate-limited?)"
  echo "$tag"
}

# ── main ─────────────────────────────────────────────────────────────────────

main() {
  need curl
  need tar

  os="$(detect_os)"

  if [ "$os" = "darwin" ]; then
    arch="$(detect_arch_darwin)"
  else
    arch="$(detect_arch)"
  fi

  version="$(resolve_version)"
  install_dir="${PLYDB_INSTALL_DIR:-$HOME/.local/bin}"

  asset="${BINARY}_${os}_${arch}.tar.gz"

  if [ "$os" = "windows" ]; then
    dest="${install_dir}/${BINARY}.exe"
  else
    dest="${install_dir}/${BINARY}"
  fi

  url="https://github.com/${REPO}/releases/download/${version}/${asset}"

  echo "Installing PlyDB ${version} (${os}/${arch})..."
  echo "  from: ${url}"
  echo "  to:   ${dest}"
  echo

  mkdir -p "$install_dir"

  tmpfile="$(mktemp)"
  trap 'rm -f "$tmpfile"' EXIT
  curl -fsSL -o "$tmpfile" "$url" || die "download failed — check that release ${version} has asset ${asset}"
  tar xzf "$tmpfile" -C "$install_dir"

  echo "PlyDB ${version} installed successfully."
  echo

  # ── PATH check ───────────────────────────────────────────────────────────

  case ":$PATH:" in
    *":${install_dir}:"*) ;;
    *)
      echo "NOTE: ${install_dir} is not in your PATH."
      echo
      echo "Add it by appending the following to your shell profile"
      echo "(e.g. ~/.bashrc, ~/.zshrc, ~/.config/fish/config.fish):"
      echo
      echo "  export PATH=\"${install_dir}:\$PATH\""
      echo
      ;;
  esac
}

main

}
