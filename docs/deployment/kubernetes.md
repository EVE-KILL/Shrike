# Kubernetes deployment

This document records the requirements for the new production manifests.

Status: Draft.

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
