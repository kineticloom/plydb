# Releases

Releases are created by pushing a version tag to `main`. GitHub Actions then
builds platform-specific binaries and skill zips, and publishes them as assets
on a GitHub Release.

---

## 1. Release Assets

Each release contains 8 assets — a standalone binary and a Claude Code skill zip
for each supported platform:

| Asset                           | Description                               |
| ------------------------------- | ----------------------------------------- |
| `plydb_linux_amd64`             | Standalone CLI binary                     |
| `plydb_linux_arm64`             | Standalone CLI binary                     |
| `plydb_darwin_arm64`            | Standalone CLI binary                     |
| `plydb_windows_amd64.exe`       | Standalone CLI binary                     |
| `plydb-skill-linux-amd64.zip`   | Claude Code skill (binary bundled inside) |
| `plydb-skill-linux-arm64.zip`   | Claude Code skill (binary bundled inside) |
| `plydb-skill-darwin-arm64.zip`  | Claude Code skill (binary bundled inside) |
| `plydb-skill-windows-amd64.zip` | Claude Code skill (binary bundled inside) |

> **Note:** `windows-arm64` is omitted — not yet supported by
> duckdb-go-bindings.

---

## 2. How to Cut a Release

```sh
git tag v1.0.0
git push origin v1.0.0
```

This triggers `.github/workflows/release.yml`, which:

1. Builds each platform binary natively on a GitHub-hosted runner
2. Packages each binary into a platform-specific skill zip
3. Creates a GitHub Release named after the tag with auto-generated notes
4. Attaches all 8 assets to the release

---

## 3. Build Matrix

Binaries are built natively (no cross-compilation) using the following runners:

| Platform        | Runner             |
| --------------- | ------------------ |
| `linux/amd64`   | `ubuntu-latest`    |
| `linux/arm64`   | `ubuntu-24.04-arm` |
| `darwin/arm64`  | `macos-latest`     |
| `windows/amd64` | `windows-latest`   |

---

## 4. Local Build

`make build-skills` builds a skill zip for the local OS/architecture only,
useful for development and testing. It does not produce the platform-named
binaries used in releases.

---

## 5. Version Embedding

Release binaries have version metadata injected at build time via `-ldflags`:

| Variable         | Source                  | Example                |
| ---------------- | ----------------------- | ---------------------- |
| `main.Version`   | `github.ref_name`       | `v1.2.3`               |
| `main.Commit`    | `github.sha`            | `a1b2c3d4...`          |
| `main.BuildDate` | `date -u` at build time | `2026-01-15T10:30:00Z` |

Local builds without ldflags (e.g. `go build .` or `go run .`) will show `dev` /
`none` / `unknown` for these fields.

To inspect the version of a binary:

```sh
plydb version
plydb --version
plydb -v
```
