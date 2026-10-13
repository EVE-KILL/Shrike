# syntax=docker/dockerfile:1
ARG GO_VERSION=1.27.2
ARG RUST_VERSION=1.99.0
ARG NODE_VERSION=24

# Compile browser sources during the image build; Node never reaches runtime.
FROM node:${NODE_VERSION}-alpine AS browser-build
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./frontend/
COPY frontend/dogma/package.json ./frontend/dogma/package.json
RUN npm --prefix frontend ci --ignore-scripts
COPY frontend/ frontend/
COPY engine/dogma/ engine/dogma/
COPY internal/frontend/ internal/frontend/
RUN node frontend/scripts/dogma-source.mjs --check \
    && npm --prefix frontend run typecheck \
    && npm --prefix frontend run build

# Native engine built for each target platform; no JavaScript runtime in the image.
FROM rust:${RUST_VERSION}-alpine3.24 AS dogma-build
RUN apk add --no-cache musl-dev
WORKDIR /src/engine/dogma
COPY engine/dogma/ ./
RUN cargo build --locked --release

FROM golang:${GO_VERSION}-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
COPY frontend/go.mod frontend/go.mod
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN find internal/frontend/static -type f \( -name '*.js' -o -name '*.js.map' \) -delete
COPY --from=browser-build /src/internal/frontend/static/ internal/frontend/static/
COPY --from=browser-build /src/internal/frontend/data/ internal/frontend/data/
COPY migrations/ migrations/
COPY frontend/dogma/ frontend/dogma/
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
RUN go tool templ generate -path internal/frontend
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/shrike ./cmd/shrike

FROM alpine:3.24 AS runtime
RUN apk add --no-cache ca-certificates tini tzdata libgcc \
    && addgroup -S -g 1000 shrike && adduser -S -u 1000 -G shrike shrike
WORKDIR /app
COPY --from=go-build /out/shrike /usr/local/bin/shrike
COPY --from=dogma-build /src/engine/dogma/target/release/shrike-dogma /usr/local/bin/shrike-dogma
COPY engine/dogma/vendor/esf-dogma-engine/LICENSE /app/licenses/dogma-engine/LICENSE
COPY engine/dogma/vendor/esf-data/LICENSE /app/licenses/dogma-data-reader/LICENSE
COPY frontend/dogma/dist/upstream/data/LICENSE \
     frontend/dogma/dist/upstream/data/LICENSE.EVE /app/licenses/dogma-data/
RUN ln -s /usr/local/bin/shrike /usr/local/bin/ek
COPY frontend/dogma/dist/upstream/data/dist/sde.dat /app/dogma/sde/sde.dat
COPY docker/entrypoint.sh /usr/local/bin/shrike-entrypoint
RUN chmod 0755 /usr/local/bin/shrike-entrypoint \
    && mkdir -p /app/data && chown shrike:shrike /app/data
ARG COMMIT=unknown
ENV GIT_SHA=${COMMIT} \
    NODE_ENV=production \
    DATA_DIR=/app/data \
    DOGMA_WORKER_PATH=/usr/local/bin/shrike-dogma \
    DOGMA_SDE_DIR=/app/dogma/sde
USER shrike
EXPOSE 4000
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/shrike-entrypoint"]
CMD ["serve"]
