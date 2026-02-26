<#
 Copyright 2026 Paul Tzen
 SPDX-License-Identifier: Apache-2.0
#>

# PlyDB installer for Windows — downloads the latest release binary.
#
# Usage:
#   irm https://raw.githubusercontent.com/kineticloom/plydb/main/install.ps1 | iex
#
# Environment variables:
#   PLYDB_INSTALL_DIR  — where to place the binary (default: ~\.local\bin)
#   PLYDB_VERSION      — version tag to install (default: latest)

$ErrorActionPreference = "Stop"

$Repo = "kineticloom/plydb"
$Binary = "plydb"

# ── detect architecture ──────────────────────────────────────────────────────

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "x86"   { "amd64" }  # 32-bit process on 64-bit OS
    "ARM64" { "arm64" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

# ── resolve version ──────────────────────────────────────────────────────────

if ($env:PLYDB_VERSION) {
    $Version = $env:PLYDB_VERSION
} else {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name
    if (-not $Version) {
        throw "Could not determine latest release."
    }
}

# ── download ─────────────────────────────────────────────────────────────────

$InstallDir = if ($env:PLYDB_INSTALL_DIR) { $env:PLYDB_INSTALL_DIR } else { "$HOME\.local\bin" }
$Asset = "${Binary}_windows_${Arch}.tar.gz"
$Dest = Join-Path $InstallDir "${Binary}.exe"
$Url = "https://github.com/$Repo/releases/download/$Version/$Asset"

Write-Host "Installing PlyDB $Version (windows/$Arch)..."
Write-Host "  from: $Url"
Write-Host "  to:   $Dest"
Write-Host ""

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$TmpFile = [System.IO.Path]::GetTempFileName()
try {
    Invoke-WebRequest -Uri $Url -OutFile $TmpFile -UseBasicParsing
    tar xzf $TmpFile -C $InstallDir
} catch {
    throw "Download failed — check that release $Version has asset $Asset"
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue $TmpFile
}

Write-Host "PlyDB $Version installed successfully."
Write-Host ""

# ── PATH check ───────────────────────────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "NOTE: $InstallDir is not in your PATH."
    Write-Host ""

    $answer = Read-Host "Would you like to add it to your User PATH? [Y/n]"
    if ($answer -eq "" -or $answer -match "^[Yy]") {
        [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$UserPath", "User")
        $env:Path = "$InstallDir;$env:Path"
        Write-Host "Added $InstallDir to your User PATH."
        Write-Host "Restart your terminal for the change to take effect."
        Write-Host ""
    } else {
        Write-Host "To add it manually, run:"
        Write-Host ""
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `"$InstallDir;`$env:Path`", 'User')"
        Write-Host ""
    }
}
