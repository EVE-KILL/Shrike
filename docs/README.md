# Documentation

This directory contains the operational and technical documentation for Shrike.

## Writing standard

Use the [documentation style](STYLE.md) for all new documents.

The style uses a practical subset of ASD-STE100 and George Orwell's writing
rules. It favors short, direct, and testable text.

## Operations

- [Development](operations/development.md): Run the site locally with `make dev`,
  open a custom killboard, and how it differs from production.
- [Cloudflare image delivery](operations/cloudflare-image-delivery.md):
  Deferred image cache and origin fallback design.
- [TLS](operations/tls.md): Local HTTPS and the planned Cloudflare origin TLS
  configuration.

## Deployment

- [Kubernetes](deployment/kubernetes.md): Production deployment gates and
  manifest requirements.
- [Release pipeline](deployment/releases.md): CI, version tags, container
  publication, and recovery.

## Architecture

- [Cache architecture](architecture/caching.md): L1, Valkey L2, response
  directives, and queue storage boundaries.
- [API contract](architecture/api-contract.md): Where each part of the OpenAPI
  document comes from, and how to document a new query parameter.
