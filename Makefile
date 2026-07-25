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

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/
