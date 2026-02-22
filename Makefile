# -----------------------------------------------------------------------------
# Dependencies
# - docker (for integration-test)
# -----------------------------------------------------------------------------

.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# -----------------------------------------------------------------------------
# Custom vars
# -----------------------------------------------------------------------------

go_files = $(shell find . -type f -name "*.go")

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
build: dist/plydb

.PHONY: build-skills
build-skills: dist/skills/plydb-skill.zip

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------

# Multi-platform builds are handled by GitHub Actions using native runners.
# See .github/workflows/build-skills.yml for linux-amd64, linux-arm64,
# darwin-arm64, and windows-amd64 builds.
# Note: windows-arm64 is omitted - not yet supported by duckdb-go-bindings.

# NOTE: this builds for the local OS and architecture only
dist/plydb: $(go_files)
	@mkdir -p $(@D)
	go build -o $@ .

dist/skills/.plydb-skill-built.sentinel: $(shell find skills/plydb -type f)
	@mkdir -p $(@D)
	cp -r skills/plydb dist/skills/
	touch $@

dist/skills/plydb-skill.zip: dist/skills/.plydb-skill-built.sentinel
	@mkdir -p $(@D)
	cd dist/skills; zip -X -r plydb-skill.zip plydb
