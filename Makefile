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

dist/plydb_linux_amd64: $(go_files)
	@mkdir -p $(@D)
	docker run --rm --platform linux/amd64 -v $(PWD):/src -w /src golang:1.26 go build -o $@ .

dist/plydb_linux_arm64: $(go_files)
	@mkdir -p $(@D)
	docker run --rm --platform linux/arm64 -v $(PWD):/src -w /src golang:1.26 go build -o $@ .

dist/plydb_darwin_arm64: $(go_files)
	@mkdir -p $(@D)
# 	docker run --rm --platform darwin/arm64 -v $(PWD):/src -w /src golang:1.26 go build -o $@ .
	GOOS=darwin GOARCH=arm64 go build -o $@ .

# Not working
# dist/plydb_windows_amd64: $(go_files)
# 	@mkdir -p $(@D)
# 	GOOS=windows GOARCH=amd64 go build -o dist/plydb_windows_amd64 .

dist/plydb: $(go_files)
	@mkdir -p $(@D)
	go build -o $@ .

# TODO: add other architectures
dist/skills/.plydb-skill-built.sentinel: $(shell find skills/plydb -type f) dist/plydb_darwin_arm64
	@mkdir -p $(@D)
	cp -r skills/plydb dist/skills/
	cp dist/plydb_darwin_arm64 dist/skills/plydb/assets/
# 	cp dist/plydb_windows_amd64 dist/skills/plydb/assets/
	touch $@

dist/skills/plydb-skill.zip: dist/skills/.plydb-skill-built.sentinel
	@mkdir -p $(@D)
	cd dist/skills; zip -X -r plydb-skill.zip plydb
