# Kubernetes deployment

This document records the requirements for the new production manifests.

Status: Draft.

## PostgreSQL read/write split

`DATABASE_URL` is the primary connection. Every worker, River client,
migration, importer, and API mutation uses it. Point it at the CloudNativePG
`rw` service, or at a session-mode Pooler whose type is `rw`.

`DATABASE_READ_URL` is used only by the HTTP service for public API, MCP, and
other eventually-consistent reads. When it is absent, Shrike reuses
`DATABASE_URL`; deploy that fallback configuration before changing database
topology.

For a three-instance CloudNativePG cluster:

- the `r` service includes the primary and both replicas;
- the `ro` service includes only the two replicas.

Use `r` when the intended read capacity is all three instances. Its credential
must belong to a PostgreSQL role with no write privileges, because `r` can
connect to the primary. Set `default_transaction_read_only=on` on that role as
defense in depth. Do not reuse the primary credential in
`DATABASE_READ_URL`.

Configure `DB_MAX_CONNS` and `DB_READ_MAX_CONNS` against separate connection
budgets. Kubernetes balances TCP connections, not individual SQL statements;
verify backend distribution from established application connections.

The live feed, authentication/account state, administration, domain management,
wallet flows, campaign mutations, killmail submissions, and every River client
remain primary-backed to preserve read-after-write consistency.

### Cutover

1. Deploy dual-pool Shrike with `DATABASE_READ_URL` unset.
2. Point `DATABASE_URL` at the restored cluster's `rw` endpoint and keep reads
   on that same endpoint.
3. Verify migrations, queues, ingestion, authentication, feed delivery, and a
   controlled primary switchover.
4. Set `DATABASE_READ_URL` to the `r` endpoint using the read-only credential.
5. Verify `pg_is_in_recovery()`, backend distribution, replica lag, public API
   behavior, and primary load.

Rollback the read split first by making `DATABASE_READ_URL` equal
`DATABASE_URL`. The database cutover can then be rolled back independently.

## Valkey

Deploy one Valkey service.

Use Valkey for shared caches, coordination, and pub/sub.

Use Postgres for River queue state.

Set a memory limit for Valkey.

Set the Valkey eviction policy to `allkeys-lru`.

Set `API_CACHE_BYTES` for each `serve` replica.

Copy the old cache Valkey address into `REDIS_HOST` and `REDIS_PORT` during the
cutover.

Do not configure `REDIS_CACHE_HOST`, `REDIS_CACHE_PORT`, `VALKEY_QUEUE`, or
`VALKEY_CACHE`.

## Cloudflare Tunnel sidecar

Each `shrike-serve` pod runs Shrike and Cloudflared. Cloudflared forwards to
`http://127.0.0.1:4000`; the ClusterIP Service remains available for internal
clients and authenticated port-forwarding. The tunnel's credentials come from
Infisical and are mounted only in the connector container. Generated tunnel
configuration names trigger rolling updates when routing changes.

Both containers have independent probes. Keep two replicas with
`maxUnavailable: 0` and `maxSurge: 1`. Verify page, legacy API/image/MCP,
WebSocket and custom-board routes before retiring the old connector Deployment.
Caddy owns legacy subdomain adaptation; there is no Nginx proxy or Nuxt render
limit in this path.

Production origin TLS is optional and is not a deployment prerequisite. Adding
it requires explicit certificate/key file configuration in Shrike and verified
trust/server-name configuration in Cloudflared. `http2Origin` requires HTTPS;
the Cloudflare zone's conventional `Full (strict)` setting does not configure
the connector's local service URL. See [TLS](../operations/tls.md).

## Related documents

- [TLS](../operations/tls.md)
- [Cloudflare image delivery](../operations/cloudflare-image-delivery.md)
