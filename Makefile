BINARY  := shrike
# Short alias for interactive use. A symlink rather than a second build, so the
# two can never drift and the image carries only one binary.
ALIAS   := ek
PKG     := ./cmd/shrike
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT)

.DEFAULT_GOAL := build

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)
	@ln -sf $(BINARY) bin/$(ALIAS)

.PHONY: gen-api-client
gen-api-client:
	go run $(PKG) openapi-spec --format json -o web/shared/api.openapi.json
	cd web/tools/api-codegen && bun install --frozen-lockfile
	cd web/tools/api-codegen && bun run generate

.PHONY: check-api-client
check-api-client: gen-api-client
	git diff --exit-code -- web/shared/api.openapi.json web/shared/api/
	@test -z "$$(git ls-files --others --exclude-standard -- web/shared/api.openapi.json web/shared/api/)" || \
		(echo "Untracked generated API contract files found; stage them." >&2; exit 1)

.PHONY: run
run:
	go run $(PKG) $(ARGS)

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -l -w .

.PHONY: check
check: fmt vet test

# The cluster runs linux; both arches are built because nodes are mixed.
.PHONY: dist
dist:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64 $(PKG)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-arm64 $(PKG)

.PHONY: hooks
hooks:
	lefthook install

.PHONY: generate
generate:
	sqlc generate

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/
