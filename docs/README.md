# Documentation

This directory contains the operational and technical documentation for Shrike.

## Writing standard

Use the [documentation style](STYLE.md) for all new documents.

The style uses a practical subset of ASD-STE100 and George Orwell's writing
rules. It favors short, direct, and testable text.

## Operations

- [Cloudflare image delivery](operations/cloudflare-image-delivery.md):
  Deferred image cache and origin fallback design.
- [TLS](operations/tls.md): Local HTTPS and the planned Cloudflare origin TLS
  configuration.

## Deployment

- [Kubernetes](deployment/kubernetes.md): Production deployment gates and
  manifest requirements.
