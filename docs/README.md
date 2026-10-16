# Documentation

This directory contains the operational and technical documentation for Shrike.

## Writing standard

Use the [documentation style](STYLE.md) for all new documents.

The style uses a practical subset of ASD-STE100 and George Orwell's writing
rules. It favors short, direct, and testable text.

## Operations

- [Development](operations/development.md): Run the site locally with `make dev`,
  open a custom killboard, and how it differs from production.
- [Frontend UI review](operations/ui-review.md): Shared usability improvements,
  validation coverage, and remaining work.
- [Cloudflare image delivery](operations/cloudflare-image-delivery.md):
  Deferred image cache and origin fallback design.
- [TLS](operations/tls.md): Local HTTPS and the planned Cloudflare origin TLS
  configuration.
- [Sentry performance investigation](operations/sentry-performance.md):
  Production query measurements, regression coverage and rejected experiments.

## Deployment

- [Kubernetes](deployment/kubernetes.md): Production deployment gates and
  manifest requirements.
- [Release pipeline](deployment/releases.md): CI, version tags, container
  publication, and recovery.
- [Sentry](deployment/observability.md): Runtime telemetry, configuration,
  delivery verification, and recovery.

## Architecture

- [Browser assets](architecture/browser-assets.md): Source ownership, TypeScript, shared modules and automatic rebuilds.
- [Go frontend](architecture/frontend.md): Page rendering, metadata, assets, and migration conventions.
- [Homepage rendering](architecture/homepage-rendering.md): Data loading and delivery for the Go homepage.
- [Kill category pages](architecture/kills-pages.md): Shared `/kills/{type}` rendering and deferred top lists.

- [Cache architecture](architecture/caching.md): L1, Valkey L2, response
  directives, and queue storage boundaries.
- [API contract](architecture/api-contract.md): Where each part of the OpenAPI
  document comes from, and how to document a new query parameter.
