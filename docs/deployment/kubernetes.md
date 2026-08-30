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

Set the Valkey eviction policy to `allkeys-lfu`.

Set `API_CACHE_BYTES` for each `serve` replica.

Copy the old cache Valkey address into `REDIS_HOST` and `REDIS_PORT` during the
cutover.

Do not configure `REDIS_CACHE_HOST`, `REDIS_CACHE_PORT`, `VALKEY_QUEUE`, or
`VALKEY_CACHE`.

## Origin TLS gate

Do not enable Cloudflare `Full (strict)` until every item in this section is
complete.

### Application tasks

1. Add configuration fields for the origin certificate and key paths.
2. Make embedded Caddy load the certificate and key.
3. Reject a configuration that contains only one TLS file.
4. Wait for a successful TLS handshake before startup completes.
5. Report TLS status through the ingress status data.

### Secret tasks

1. Create an elliptic-curve Cloudflare Origin CA certificate.
2. Include `eve-kill.com` and the required wildcard names.
3. Store the certificate and key in a Kubernetes TLS secret.
4. Mount the secret read-only in the `serve` container.
5. Keep the secret outside Git and rendered manifest output.
6. Record the certificate expiration in operational monitoring.

Do not give the Shrike pod a Cloudflare certificate-management token.

### Service tasks

1. Expose the Caddy TLS port through the Kubernetes Service.
2. Configure the readiness and liveness probes for HTTPS.
3. Keep the private Nuxt and API Unix sockets inside the pod.
4. Do not expose User Datagram Protocol for Cloudflare origin traffic.

Cloudflare documents HTTP/2 and HTTP/1.1 for origin connections. It does not
document HTTP/3 for this origin path.

### Cloudflare tasks

1. Confirm the origin Server Name Indication value for tenant custom domains.
2. Confirm that the Origin CA certificate covers that name.
3. Confirm that HTTP/2 to Origin is enabled.
4. Test the origin certificate before changing the encryption mode.
5. Enable `Full (strict)`.

### Verification

1. Load `/health` through Cloudflare.
2. Confirm that Cloudflare does not return error `526`.
3. Load API, image, WebSocket, and Nuxt routes.
4. Load a tenant custom domain.
5. Confirm that Caddy negotiates HTTP/2 for origin requests.
6. Confirm that no public route uses the private Unix sockets.

### Recovery

1. Redeploy the prior Shrike manifest.
2. Verify the prior origin connection.
3. Restore the prior Cloudflare encryption mode.

Restore the prior Service and probe configuration with the application
manifest.

Do not delete the previous TLS secret until the new deployment passes
verification.

## Related documents

- [TLS](../operations/tls.md)
- [Cloudflare image delivery](../operations/cloudflare-image-delivery.md)
