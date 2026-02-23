# -----------------------------------------------------------------------------
# Dependencies
# - docker (for integration-test)
# -----------------------------------------------------------------------------

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# -----------------------------------------------------------------------------
# Custom vars
# -----------------------------------------------------------------------------

GO_FILES = $(shell find . -type f -name "*.go")

# Build metadata — overridable from CI: make VERSION=v1.2.3 COMMIT=abc1234
VERSION    ?= dev
COMMIT     ?= none
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Platform — auto-detected from native runner; overridable if needed
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXE     = $(if $(filter windows,$(GOOS)),.exe,)

LDFLAGS = -X github.com/kineticloom/plydb/cmd.Version=$(VERSION) \
          -X github.com/kineticloom/plydb/cmd.Commit=$(COMMIT) \
          -X github.com/kineticloom/plydb/cmd.BuildDate=$(BUILD_DATE)

# -----------------------------------------------------------------------------
# Top level commands
# -----------------------------------------------------------------------------

.PHONY: clean
clean:
	rm -rf dist

.PHONY: test
test:
	go test ./...

.PHONY: integration-test
integration-test:
	go test -tags=integration -v -timeout 300s ./...

.PHONY: build
build: dist/plydb$(EXE)

.PHONY: build-skill
build-skill: dist/plydb_skill.zip

.PHONY: package-release
package-release: dist/plydb_$(GOOS)_$(GOARCH).tar.gz

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------

# Multi-platform builds are handled by GitHub Actions using native runners.
# See .github/workflows/release.yml for linux-amd64, linux-arm64,
# darwin-arm64, and windows-amd64 builds.
# Note: windows-arm64 is omitted - not yet supported by duckdb-go-bindings.

dist/plydb$(EXE): $(GO_FILES)
	@mkdir -p $(@D)
	go build -ldflags "$(LDFLAGS)" -o $@ .

dist/plydb_skill.zip: $(shell find skills/plydb -type f)
	@mkdir -p $(@D)
	cd skills; zip -X -r ../dist/plydb_skill.zip plydb

# Package the release binary as a tar.gz whose root contains only plydb (or plydb.exe),
# matching the layout expected by install.sh / install.ps1.
dist/plydb_$(GOOS)_$(GOARCH).tar.gz: dist/plydb$(EXE)
	tar czf $@ -C dist plydb$(EXE)
