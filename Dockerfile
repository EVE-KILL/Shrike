# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.7
ARG BUN_VERSION=1.3.14

FROM oven/bun:${BUN_VERSION}-alpine AS web-build

WORKDIR /src/web

# Dogma is a local file dependency. It must be present when Bun resolves the
# lockfile, not copied afterwards with the rest of the renderer source.
COPY web/package.json web/bun.lock ./
COPY web/packages/dogma packages/dogma
RUN bun install --frozen-lockfile

COPY web/ ./
RUN bun run build
RUN mkdir -p /out/dogma/src \
    && bun build packages/dogma/src/bridge.ts --target=bun --outfile=/out/dogma/src/bridge.js \
    && cp -R packages/dogma/dist /out/dogma/dist

FROM golang:${GO_VERSION}-alpine AS go-build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY migrations/ migrations/

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/shrike ./cmd/shrike

FROM oven/bun:${BUN_VERSION}-alpine AS runtime

RUN apk add --no-cache ca-certificates tini tzdata

WORKDIR /app

COPY --from=go-build /out/shrike /usr/local/bin/shrike
RUN ln -s /usr/local/bin/shrike /usr/local/bin/ek

COPY --chown=bun:bun --from=web-build /src/web/.output /app/web
COPY --chown=bun:bun --from=web-build /out/dogma /app/dogma
COPY docker/entrypoint.sh /usr/local/bin/shrike-entrypoint
RUN chmod 0755 /usr/local/bin/shrike-entrypoint \
    && mkdir -p /app/data \
    && chown bun:bun /app/data

ARG COMMIT=unknown
ENV GIT_SHA=${COMMIT} \
    NODE_ENV=production \
    DATA_DIR=/app/data \
    NUXT_ENTRYPOINT=/app/web/server/index.mjs \
    DOGMA_BRIDGE_PATH=/app/dogma/src/bridge.js \
    NUXT_SOCKET=/tmp/shrike-nuxt.sock \
    SHRIKE_API_SOCKET=/tmp/shrike-api.sock

USER bun

EXPOSE 4000

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/shrike-entrypoint"]
CMD ["serve"]
