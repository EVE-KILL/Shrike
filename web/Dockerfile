FROM oven/bun:1.3.14-alpine AS build

WORKDIR /src

# The local dogma package is the only workspace dependency retained by the
# renderer. Bun installs file dependencies as package snapshots, so its source
# and generated data must exist before installation rather than arriving in
# the later application-source copy.
COPY package.json bun.lock ./
COPY packages/dogma packages/dogma
RUN bun install --frozen-lockfile

COPY . .

ENV NITRO_PRESET=bun
RUN bun run build

FROM oven/bun:1.3.14-alpine

WORKDIR /app

COPY --from=build /src/.output ./

ARG GIT_SHA=unknown
ENV GIT_SHA=${GIT_SHA}
ENV NODE_ENV=production

EXPOSE 3000

CMD ["bun", "--bun", "run", "/app/server/index.mjs"]
