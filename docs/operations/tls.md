# TLS

This document describes local HTTPS and the planned Cloudflare origin TLS
configuration.

## Current behavior

| Environment | Client connection | Origin connection | Certificate |
| --- | --- | --- | --- |
| Development | HTTPS | Direct to Caddy | Caddy internal CA |
| Production | HTTPS to Cloudflare | HTTP to Caddy | Cloudflare edge certificate |

Development Caddy supports HTTP/1.1, HTTP/2, and HTTP/3.

Production Cloudflare currently uses HTTP/2 or HTTP/1.1 for origin connections.
Cloudflare enables HTTP/2 origin connections by default.

## Start local HTTPS

### Prerequisites

- Set `NODE_ENV=development`.
- Set `DATA_DIR` to a writable directory.
- Build the Nuxt renderer.

### Procedure

1. Start Shrike.

   ```sh
   go run ./cmd/shrike serve
   ```

2. Approve the trust-store request when the operating system shows it.

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

Approve the trust-store request if Caddy could not install the local root
certificate.

Do not commit or copy `${DATA_DIR}/caddy`. This directory contains private keys.

## Add production origin TLS

Status: Planned.

Use this connection path:

```text
Browser -- HTTPS and HTTP/3 --> Cloudflare -- HTTPS and HTTP/2 --> Caddy
```

TLS protects the Cloudflare-to-origin connection. HTTP/2 multiplexing reduces
the number of origin connections.

Use a Cloudflare Origin CA certificate for the origin. Cloudflare accepts this
certificate with `Full (strict)` mode.

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
