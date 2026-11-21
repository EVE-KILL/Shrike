# TLS

## Production

Cloudflare handles public HTTPS, HTTP/2 and HTTP/3. The tunnel from Cloudflare
to Cloudflared is encrypted. In each serve pod, Cloudflared forwards requests
over plain HTTP to Shrike on `127.0.0.1:4000`.

Shrike routes requests by host and path. It adapts the legacy `api`, `images`,
`ws` and `mcp` subdomain paths and keeps normal routing for tenant hosts.
`LEGACY_DOMAIN` defaults to `eve-kill.com`. Shrike sets `X-Forwarded-Host` from
the request's Host and preserves the forwarding scheme and client address
provided by Cloudflared.

## Local HTTPS

Local development uses HTTP by default. For HTTPS, provision a certificate
from a development CA trusted by the browser. Include `localhost` and the
specific tenant names you intend to test in the certificate's SANs.

```sh
go run ./cmd/shrike serve --port 4000 --tls-cert cert.pem --tls-key key.pem
curl --fail --show-error https://localhost:4000/health
```

Pass both PEM files to start the HTTPS listener. The listener supports HTTP/1.1
and HTTP/2. Keep the private key outside version control.

## Optional production origin TLS

To enable HTTPS on the pod loopback connection, mount a certificate and key
read-only and pass both file paths to `shrike serve`. Set the origin service
URL to HTTPS in Cloudflared, configure the trusted CA and origin server name,
and enable `http2Origin` if the origin should use HTTP/2. Configure Kubernetes
probes for HTTPS on the same port.

The Cloudflare zone's encryption mode controls conventional proxied origins.
The tunnel uses its own origin connection settings.

- [Kubernetes deployment](../deployment/kubernetes.md)
- [Cloudflare Origin CA](https://developers.cloudflare.com/ssl/origin-configuration/origin-ca/)
- [Cloudflare tunnel origin parameters](https://developers.cloudflare.com/tunnel/reference/origin-parameters/)
