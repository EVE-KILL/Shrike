# Sentry

Shrike uses sentry-go v0.50.0 in the shared Go runtime. `serve`, `dev`, queue
consumers, scheduled/manual crons, feed ingestion and operational commands use
the same configuration. An empty `SENTRY_DSN` disables telemetry.

## Configuration

Configure `SENTRY_DSN` in the deployment environment. Local overrides belong in
the ignored `.env`; never put an auth token or private MCP configuration in Git.

| Setting | Default | Meaning |
| --- | --- | --- |
| `SENTRY_ENVIRONMENT` | `NODE_ENV` | Separates production, development and verification |
| `SENTRY_TRACES_SAMPLE_RATE` | `0.1` | Fraction of traces retained; `1` records every trace |
| `SENTRY_LOGS` | `true` | Structured zerolog and River/slog logs |
| `SENTRY_LOG_LEVEL` | `info` | Minimum log severity sent to Sentry |
| `SENTRY_LOG_SAMPLE_RATE` | `1` | Fraction of info/debug logs retained; warnings/errors are retained |
| `SENTRY_METRICS` | `true` | Application metrics and periodic runtime gauges |

Error monitoring is independent of trace/log sampling. Errors are not sampled.
Events carry `release=shrike@VERSION`, the build commit in `dist`, environment,
pod hostname and the command's `service` tag (`serve`, `work queues`, `work cron`,
`work zkb`, etc.). Shutdown flushes pending events, logs and metrics for up to
two seconds. Sentry delivery happens asynchronously and does not control API
availability or job retries.

## Coverage

- HTTP requests: grouped route, method, status, response size, duration, cache
  outcome, panics and server errors. Response writer capabilities are preserved.
- Go pages/fragments: separate data-loading and template-rendering spans.
- PostgreSQL: query, batch, COPY and pool-acquire timing; normalized SQL, errors,
  bulk row counts and connection utilization. Development's performance panel
  continues using the same tracer.
- Valkey: command/pipeline/connect timing and failure counts, connection-pool
  utilization. Commands include their names, never keys, arguments or values.
- API cache: L1/L2 hits, misses and stale entries; byte/entry capacity; background
  refreshes have their own transactions linked to the initiating request.
- Image cache: hits/misses and byte/entry capacity.
- River: job kind/queue, attempts, waiting/execution time, enqueue/deduplication,
  success/retry/permanent-failure/snooze/cancellation/panic, and queue depth.
  Trace propagation is stored under `_sentry` in job metadata; arguments and
  deduplication behavior are unchanged.
- Crons: named transactions, results, timings and downtime skips.
- CCP ESI: overall pipeline timing, cached/network outcomes, actual HTTP
  attempts, retries, response status, pause/error-budget/rate-limit waits,
  remaining rate/error budgets and retry-after values. Authenticated calls
  are identified by a boolean, never by their access token.
- EVE SSO, SDE downloads, EVE Ref, zKillboard, image upstreams and object storage:
  HTTP timing/status/bytes without forwarding trace headers to third parties.
- Memgraph: read/ingest/purge timing and normalized read queries.
- Native dogma: calculation duration and worker-pool wait/capacity.
- WebSockets: connections, handshake results, deliveries and slow-client
  backpressure. WebSocket/SSE sessions do not retain a transaction for their
  entire lifetime.
- Go runtime: goroutines, heap bytes/objects and GC cycles, every 30 seconds.

Metrics are flushed every 15 seconds. Counters are summed exactly and gauges
retain the latest value per label set. Duration distributions use up to 64
uniformly sampled observations per series per interval; use traces for exact
individual requests. A 2,048-series limit bounds memory, and
`telemetry.metrics.dropped_observations` reports observations rejected by that
limit. Keep metric labels categorical: routes, operation names, queue, cache
tier and outcome. Never label metrics with a character, killmail, user or key.

## Data handling

HTTP headers/cookies/bodies/query parameters and automatic user information
are disabled at collection. The outbound transport observes timings and byte
counts without copying bodies or changing authorization/trace headers. Error,
log, breadcrumb and metric callbacks remove known credentials and sensitive
fields. SQL/Cypher literals and comments are removed and bind arguments are
never collected. Existing nginx-style access lines remain local; Sentry gets a
structured request outcome without IP addresses, referrers or raw query strings.

## Verification

This command opens no database connections. It sends one intentional error,
structured log, three metrics, and a sampled test transaction:

```sh
SENTRY_ENVIRONMENT=verification shrike sentry:test --json
```

Keep logs/metrics/tracing enabled while verifying. Search the returned event
and trace IDs in Sentry and confirm all four signal types are present. The error
is tagged `verification=true`, so it can be distinguished from real failures.

For Codex in this checkout, configure `.codex/config.toml` locally:

```toml
[mcp_servers.sentry]
url = "https://mcp.sentry.dev/mcp/eve-kill/shrike"
```

Run `codex mcp login sentry` once. `.codex/` is gitignored; OAuth credentials
remain in Codex's credential store. The Sentry CLI can independently inspect
the same `eve-kill/shrike` project.

## Recovery

If verification fails, check these stages in order. Keep production serving
requests while investigating; telemetry failures do not require a restart.

1. Check that `SENTRY_DSN` and the verification feature settings are configured in the running process.
2. Check the command's exit status and `delivered` result.
3. If configuration fails, correct the invalid setting before retrying the command.
4. If `delivered` is false, check DNS, TLS, egress rules, and connectivity to the DSN's ingest host.
5. If `delivered` is true but signals are absent, select the matching project, environment, and time range in Sentry.
6. Check Sentry's ingestion usage, quotas, rate limits, and service status for rejected or delayed envelopes.
7. If delivery remains unclear, enable SDK transport diagnostics temporarily in an isolated verification process.
8. Inspect transport errors and HTTP response codes without sharing credentials or unredacted diagnostic output.
9. If telemetry affects service health, set `SENTRY_DSN` to an empty value through deployment configuration and roll out the change.
10. Resolve intentional verification issues after all four signal types arrive.

`delivered=true` confirms the SDK drained its buffer; it does not confirm Sentry
accepted or indexed every signal. Verify ingestion by reading the returned IDs
and metrics from Sentry. Avoid repeated verification runs while a quota or
network failure remains unresolved.

## Sources

SDK references: [Go SDK](https://github.com/getsentry/sentry-go),
[logging](https://docs.sentry.io/platforms/go/logs/),
[metrics](https://docs.sentry.io/platforms/go/metrics/),
[Codex MCP](https://developers.openai.com/codex/mcp/).
