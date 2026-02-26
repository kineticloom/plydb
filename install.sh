#!/bin/sh

# Copyright 2026 Paul Tzen
# SPDX-License-Identifier: Apache-2.0

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

      shell_name="$(basename "${SHELL:-}")"
      case "$shell_name" in
        zsh)  profile="$HOME/.zshrc" ;;
        bash) profile="$HOME/.bashrc" ;;
        fish) profile="${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish" ;;
        *)    profile="" ;;
      esac

      export_line="export PATH=\"${install_dir}:\$PATH\""
      fish_line="set -gx PATH \"${install_dir}\" \$PATH"

      if [ -n "$profile" ] && [ -c /dev/tty ]; then
        printf "Would you like to add it to %s? [Y/n] " "$profile"
        read -r answer </dev/tty
        case "${answer:-y}" in
          [Yy]*|"")
            if [ "$shell_name" = "fish" ]; then
              line_to_add="$fish_line"
            else
              line_to_add="$export_line"
            fi
            if ! grep -qF "$line_to_add" "$profile" 2>/dev/null; then
              printf '\n# Added by PlyDB installer\n%s\n' "$line_to_add" >> "$profile"
            fi
            echo "Added to ${profile}."
            echo "Restart your shell or run: . ${profile}"
            echo
            ;;
          *)
            echo "To add it manually, append the following to your shell profile:"
            echo
            echo "  $export_line"
            echo
            ;;
        esac
      else
        echo "Add it by appending the following to your shell profile"
        echo "(e.g. ~/.bashrc, ~/.zshrc, ~/.config/fish/config.fish):"
        echo
        echo "  $export_line"
        echo
      fi
      ;;
  esac
}

main

}
