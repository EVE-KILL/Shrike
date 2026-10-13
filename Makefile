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
build: build-browser-assets build-frontend build-dogma
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)
	@ln -sf $(BINARY) bin/$(ALIAS)

.PHONY: build-frontend
build-frontend:
	go tool templ generate -path internal/frontend

# Native fitting worker; the browser retains the matching WASM engine.
CARGO ?= cargo
DOGMA_SOURCES := $(shell rg --files engine/dogma -g '*.rs' -g 'Cargo.toml' -g 'Cargo.lock' -g 'compat.json')
.PHONY: build-dogma
build-dogma: bin/shrike-dogma

bin/shrike-dogma: $(DOGMA_SOURCES)
	$(CARGO) build --locked --release --manifest-path engine/dogma/Cargo.toml
	@mkdir -p bin
	@cp engine/dogma/target/release/shrike-dogma bin/shrike-dogma.next
	@mv bin/shrike-dogma.next bin/shrike-dogma

# Rebuild after changing the shared Rust adapter or upstream calculation source.
# Requires rustup target wasm32-unknown-unknown and wasm-bindgen-cli 0.2.128.
.PHONY: build-dogma-wasm
build-dogma-wasm:
	CARGO="$(CARGO)" bash engine/dogma/build-wasm.sh
	npm --prefix frontend run build -- dogma

# Browser bundles are checked in. Install tooling once with npm --prefix frontend ci.
.PHONY: build-browser-assets test-frontend
build-browser-assets:
	npm --prefix frontend run build

test-frontend:
	npm --prefix frontend test
	npm --prefix frontend run typecheck

.PHONY: build-dogma-frontend
build-dogma-frontend:
	npm --prefix frontend run build -- dogma

.PHONY: build-map-frontend
build-map-frontend:
	npm --prefix frontend run build -- map

.PHONY: build-fit-table-frontend
build-fit-table-frontend:
	npm --prefix frontend run build -- fit-table

.PHONY: build-fit-drag-frontend
build-fit-drag-frontend:
	npm --prefix frontend run build -- fit-drag

.PHONY: build-fit-document-frontend
build-fit-document-frontend:
	npm --prefix frontend run build -- fit-document

.PHONY: build-css-frontend
build-css-frontend:
	npm --prefix frontend run build -- css

.PHONY: build-formatters-frontend
build-formatters-frontend:
	npm --prefix frontend run build -- formatters

.PHONY: build-settings-frontend
build-settings-frontend:
	npm --prefix frontend run build -- settings

.PHONY: build-account-frontend
build-account-frontend:
	npm --prefix frontend run build -- account

.PHONY: build-esi-frontend
build-esi-frontend:
	npm --prefix frontend run build -- esi

.PHONY: build-admin-frontend
build-admin-frontend:
	npm --prefix frontend run build -- admin

.PHONY: build-announcement-frontend
build-announcement-frontend:
	npm --prefix frontend run build -- announcement

.PHONY: build-icons-frontend
build-icons-frontend:
	npm --prefix frontend run build -- icons

.PHONY: build-search-frontend
build-search-frontend:
	npm --prefix frontend run build -- search

.PHONY: build-campaign-frontend
build-campaign-frontend:
	npm --prefix frontend run build -- campaign

.PHONY: build-comment-frontend
build-comment-frontend:
	npm --prefix frontend run build -- comment

.PHONY: build-battle-frontend
build-battle-frontend:
	npm --prefix frontend run build -- battle

.PHONY: build-faction-war-frontend
build-faction-war-frontend:
	npm --prefix frontend run build -- faction-war

.PHONY: build-battle-generator-frontend
build-battle-generator-frontend:
	npm --prefix frontend run build -- battle-generator

.PHONY: fmt-frontend
fmt-frontend:
	go tool templ fmt internal/frontend

.PHONY: gen-api-client
gen-api-client:
	go run $(PKG) openapi-spec --format json -o frontend/shared/api.openapi.json
	cd frontend/tools/api-codegen && npm ci --ignore-scripts
	cd frontend/tools/api-codegen && npm run generate

.PHONY: check-api-client
check-api-client: gen-api-client
	git diff --exit-code -- frontend/shared/api.openapi.json frontend/shared/api/
	@test -z "$$(git ls-files --others --exclude-standard -- frontend/shared/api.openapi.json frontend/shared/api/)" || \
		(echo "Untracked generated API contract files found; stage them." >&2; exit 1)

.PHONY: run
run:
	go run $(PKG) $(ARGS)

# Development is one Go process. Air regenerates templates and rebuilds it.
DEV_PORT ?= 4000
DATA_DIR ?= data

.PHONY: dev
dev: build-dogma
	@command -v air >/dev/null || { \
		echo "air is required: go install github.com/air-verse/air@latest" >&2; exit 1; }
	@echo "dev: http://localhost:$(DEV_PORT) (Go frontend)"
	PORT=$(DEV_PORT) ./scripts/dev.sh

.PHONY: dev-trust
dev-trust:
	@case "$$(uname -s)" in \
		Darwin) ;; \
		*) echo "dev-trust currently supports macOS only" >&2; exit 1 ;; \
	esac
	@cert_path="$(DATA_DIR)/caddy/pki/authorities/local/root.crt"; \
	case "$$cert_path" in /*) ;; *) cert_path="$(CURDIR)/$$cert_path" ;; esac; \
	test -f "$$cert_path" || { \
		echo "local CA not found; start shrike serve once to generate it" >&2; exit 1; }; \
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
