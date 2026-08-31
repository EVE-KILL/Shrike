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
build: build-frontend
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)
	@ln -sf $(BINARY) bin/$(ALIAS)

.PHONY: build-frontend
build-frontend:
	cd web && bun install --frozen-lockfile
	cd web && bun run build

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

# Development stack: air rebuilds Go, `nuxt dev` serves the frontend, and Caddy
# fronts both on DEV_PORT.
#
# The two run as siblings rather than parent and child. If shrike supervised the
# development server, every Go rebuild would kill Vite — losing hot module
# replacement state and paying a cold Nuxt start on each edit. Air restarts the
# Go binary alone and Caddy re-attaches to the renderer that stayed up.
#
# Server-side rendering reaches Go on DEV_API_PORT rather than the production
# Unix socket. `nuxt dev` runs under Node, and Node's fetch ignores the `unix`
# request option that Bun implements and that web/shared/utils/serverApi.ts
# uses, so a socket would resolve shrike.internal through DNS and fail. The
# listener is loopback-only and serves the same handler.
DEV_PORT     ?= 4001
DEV_API_PORT ?= 4002
DATA_DIR     ?= data

# Nuxt silently moves to another port when its requested port is occupied, but
# Shrike cannot discover that choice and would keep proxying to the old port.
# Resolve the first free development port ourselves so both processes receive
# the same value. An explicit DEV_NUXT_PORT still wins.
ifndef DEV_NUXT_PORT
DEV_NUXT_PORT := $(shell \
	port=3000; \
	while [ $$port -le 3009 ]; do \
		if ! lsof -nP -iTCP:$$port -sTCP:LISTEN -t 2>/dev/null | grep -q .; then \
			echo $$port; \
			break; \
		fi; \
		port=$$((port + 1)); \
	done)
endif

.PHONY: dev
dev:
	@command -v air >/dev/null || { \
		echo "air is required: go install github.com/air-verse/air@latest" >&2; exit 1; }
	@command -v bun >/dev/null || { echo "bun is required" >&2; exit 1; }
	@command -v lsof >/dev/null || { echo "lsof is required" >&2; exit 1; }
	@test -n "$(DEV_NUXT_PORT)" || { \
		echo "no free Nuxt port found in 3000-3009; set DEV_NUXT_PORT explicitly" >&2; exit 1; }
	@echo "dev: https://localhost:$(DEV_PORT)  (nuxt :$(DEV_NUXT_PORT), ssr api :$(DEV_API_PORT))"
	@NUXT_API_ORIGIN=http://127.0.0.1:$(DEV_API_PORT) \
	NUXT_DEV_PROXY_PORT=$(DEV_PORT) \
		bun --cwd=$(CURDIR)/web run dev \
			--host 127.0.0.1 --port $(DEV_NUXT_PORT) & \
	nuxt_pid=$$!; \
	trap 'kill $$nuxt_pid 2>/dev/null' INT TERM EXIT; \
	SHRIKE_DEV_RENDERER=127.0.0.1:$(DEV_NUXT_PORT) \
	SHRIKE_DEV_API_ADDR=127.0.0.1:$(DEV_API_PORT) \
	PORT=$(DEV_PORT) air

.PHONY: dev-trust
dev-trust:
	@case "$$(uname -s)" in \
		Darwin) ;; \
		*) echo "dev-trust currently supports macOS only" >&2; exit 1 ;; \
	esac
	@cert_path="$(DATA_DIR)/caddy/pki/authorities/local/root.crt"; \
	case "$$cert_path" in /*) ;; *) cert_path="$(CURDIR)/$$cert_path" ;; esac; \
	test -f "$$cert_path" || { \
		echo "local CA not found; start make dev once to generate it" >&2; exit 1; }; \
	echo "Installing Shrike's local development CA into the macOS system keychain."; \
	sudo security add-trusted-cert -d -r trustRoot \
		-k /Library/Keychains/System.keychain "$$cert_path"

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

.PHONY: image
image:
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		-t evekill:$(COMMIT) .

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
