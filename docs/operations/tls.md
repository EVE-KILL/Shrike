# TLS

This document describes local HTTPS and the production Cloudflare Tunnel.

## Current behavior

| Environment | Client connection | Origin connection | Certificate |
| --- | --- | --- | --- |
| Development | HTTPS | Direct to Caddy | Caddy internal CA |
| Production | HTTPS to Cloudflare | HTTP over pod loopback to Caddy | Cloudflare edge certificate |

Development Caddy supports HTTP/1.1, HTTP/2, and HTTP/3.

Production runs a Cloudflared container beside Shrike in each serve pod. The
tunnel is encrypted between Cloudflare and Cloudflared; only the final
`http://127.0.0.1:4000` connection is plain HTTP/1.1. Browser-facing HTTP/2 and
HTTP/3 terminate at Cloudflare independently of that loopback connection.

No Nginx proxy or Bun-era render-concurrency limiter is required. Caddy invokes
the Go handlers directly, including the legacy `api`, `images`, `ws`, and `mcp`
subdomain routes. `LEGACY_DOMAIN` defaults to `eve-kill.com`; other tenant hosts
keep the normal page/API/WebSocket path routing. The original Host, query,
request body, cookies, Cloudflare client IP and HTTPS forwarding header survive
the path adaptation. Caddy replaces `X-Forwarded-Host` with the actual Host.

## Start local HTTPS

### Prerequisites

- Set `NODE_ENV=development`.
- Set `DATA_DIR` to a writable directory.
- Build Shrike and its native fitting worker with `make build`.

### Procedure

1. Start Shrike.

   ```sh
   go run ./cmd/shrike serve
   ```

2. When using `make dev` on macOS, install the generated authority from a
   separate foreground terminal:

   ```sh
   make dev-trust
   ```

   This command invokes `sudo` directly, so the password prompt remains attached
   to the terminal. The embedded Caddy process does not attempt to read a
   password through Air.

3. Open `https://localhost:4001`.

Use `localhost`. The certificate does not contain loopback IP addresses.

A custom killboard uses `https://<subdomain>.localhost:4001`. Caddy signs a
certificate for that name at the first request, from the same local authority.
The board names are database rows, so the configuration cannot list them.

A single `*.localhost` certificate would replace this, but Chromium and curl
reject a wildcard whose parent is one label.

The listener accepts HTTPS only. `http://localhost:4001` cannot use this
listener.

### Verification

Run this command after Shrike reports `site listening`:

```sh
curl --fail --show-error https://localhost:4001/health
```

The command must return a successful health response.

Caddy stores the local certificate authority under
`${DATA_DIR}/caddy/pki/authorities/local`.

### Recovery

Read the Caddy startup error when the browser rejects the certificate.

Run `make dev-trust` if the local root certificate is not trusted on macOS.

Do not commit or copy `${DATA_DIR}/caddy`. This directory contains private keys.

## Optional production origin TLS

Status: Not enabled. Plain HTTP over pod loopback is the chosen production
transport. The following is an optional future change, not a deployment gate.

Use this connection path:

```text
Browser -- HTTPS --> Cloudflare -- encrypted tunnel --> Cloudflared -- HTTPS/HTTP2 --> Caddy
```

TLS would encrypt the final loopback connection and permit HTTP/2 multiplexing
there. Measure an actual benefit before adding certificate management.

Use a Cloudflare Origin CA certificate for the origin. Cloudflared must validate
its issuer and expected server name. Configure `caPool` with the issuing root,
keep `noTLSVerify` false, and set `http2Origin` true. The tunnel's origin TLS
settings are separate from the zone's encryption mode for conventional proxied
origins.

Store the certificate and private key in a Kubernetes TLS secret. Mount the
secret read-only in the `serve` container.

Do not give Shrike a Cloudflare certificate-management token. The running
service does not need certificate creation rights.

Cloudflare Origin CA certificates are not public browser certificates. Direct
browser access to the origin will show a trust error.

### Implementation tasks

1. Add configuration fields for the certificate and key file paths.
2. Make Caddy load both files when production origin TLS is enabled.
3. Reject incomplete certificate configuration during startup.
4. Expose the TLS listener to Cloudflare.
5. Confirm the origin Server Name Indication contract for tenant domains.
6. Enable Cloudflare `Full (strict)` after the origin test succeeds.

The [Kubernetes deployment guide](../deployment/kubernetes.md) defines the
manifest gate and verification sequence.

### Verification

Confirm that Cloudflare can load the health endpoint without error `526`.

Confirm that Caddy negotiates HTTP/2 for Cloudflare origin requests.

Confirm that tenant custom domains use the expected origin certificate name.

### References

- [Cloudflare Full (strict)](https://developers.cloudflare.com/ssl/origin-configuration/ssl-modes/full-strict/)
- [Cloudflare Origin CA](https://developers.cloudflare.com/ssl/origin-configuration/origin-ca/)
- [Cloudflare HTTP/2 to Origin](https://developers.cloudflare.com/speed/optimization/protocol/http2-to-origin/)
