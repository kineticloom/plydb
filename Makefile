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

LICENSE_SRCS = $(shell find . \
  -type f \( -name "*.go" -o -name "*.sh" -o -name "*.ps1" \) \
  -not -path "./dist/*" \
  -not -path "./demo_sandbox/*" \
  -not -path "./.git/*" \
  -not -path "./.vscode/*" \
  -not -path "./.claude/*")

# -----------------------------------------------------------------------------
# Top level commands
# -----------------------------------------------------------------------------

.PHONY: clean
clean:
	rm -rf dist

.PHONY: check
check: build lint test integration-test license-check notices-check vuln-check

.PHONY: test
test:
	go test ./...

.PHONY: integration-test
integration-test:
	go test -tags=integration -v -timeout 300s ./...

.PHONEY: lint
lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

.PHONY: vuln-check
vuln-check:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: license-check
license-check:
# NOTE: license detection is failing for github.com/segmentio/asm@v1.2.1
# Manually validated that it is MIT-0 no attribution
	go run github.com/google/addlicense@latest \
	  -check -c "Paul Tzen" -l apache -s=only \
	  $(LICENSE_SRCS)
	go run github.com/google/go-licenses@latest check \
	  --ignore github.com/segmentio/asm \
	  --allowed_licenses=Apache-2.0,MIT,MIT-0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0 \
	  ./...

.PHONY: license-fix
license-fix:
	go run github.com/google/addlicense@latest \
	  -c "Paul Tzen" -l apache -s=only \
	  $(LICENSE_SRCS)

.PHONY: notices-generate
notices-generate:
# NOTE: license detection is failing for github.com/segmentio/asm@v1.2.1
# Manually validated that it is MIT-0 no attribution
	go run github.com/google/go-licenses@latest report \
	  --ignore github.com/kineticloom/plydb \
	  --ignore github.com/segmentio/asm \
	  --template=scripts/notices.tpl \
	  ./... > THIRD_PARTY_NOTICES.md
	cat scripts/notices_manual.md >> THIRD_PARTY_NOTICES.md

.PHONY: notices-check
notices-check:
# NOTE: license detection is failing for github.com/segmentio/asm@v1.2.1
# Manually validated that it is MIT-0 no attribution
	go run github.com/google/go-licenses@latest report \
	  --ignore github.com/segmentio/asm \
	  --ignore github.com/kineticloom/plydb \
	  --template=scripts/notices.tpl \
	  ./... > /tmp/plydb_notices_check.md
	cat scripts/notices_manual.md >> /tmp/plydb_notices_check.md
	diff THIRD_PARTY_NOTICES.md /tmp/plydb_notices_check.md \
	  || (echo "THIRD_PARTY_NOTICES.md is out of date — run 'make notices-generate'" && exit 1)

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
