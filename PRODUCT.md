# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Newcomers and experienced EVE Online players are equally primary audiences.
Both groups need to find, read, and understand combat activity.
Experienced players also need efficient comparison, filtering, fitting, and analysis workflows.
Support mobile browsing as a full part of the product.

## Product Purpose

EVE-KILL helps players explore combat reports, understand losses, and investigate activity across New Eden.
The site combines killmails, entity histories, search, maps, battle reports, and fitting tools.
Users should find relevant information quickly and understand what each result represents.

## Operating Context

Users browse live kill feeds and follow links to pilots, corporations, alliances, ships, and locations.
They inspect killmail participants, equipment, values, and related reports.
They refine searches, explore maps, and create or compare ship fittings.
Custom domains provide scoped killboards through the same application.

## Capabilities and Constraints

- Render pages in Go and templ, with reusable server-rendered components.
- Use HTML fragments for expensive sections and retained page interactions.
- Keep browser behavior in TypeScript and the existing Stimulus architecture.
- Build browser assets for development and production; require no JavaScript server at runtime.
- Preserve public APIs, API documentation, and the Model Context Protocol interface for external consumers.
- Keep fitting calculations compatible with the pinned Rust engine and its browser WebAssembly build.
- Preserve custom themes, domain themes, navigation history, responsive image geometry, and live feed reconnection.
- Show loading, empty, error, and retry states where data loads asynchronously.

## Brand Commitments

Keep the established EVE-KILL name, assets, terminology, and recognizable interface.
Use the original Nuxt source as the reference for faithful ports of existing behavior and appearance.
Preserve the fitting wheel instead of inventing a replacement.
The user welcomes cohesive improvements within that identity.

## Evidence on Hand

- Existing Go templates, CSS, TypeScript components, tests, and real EVE data.
- Original Nuxt source in Git history.
- Captured parity fixtures under `frontend/test/fixtures`.
- Development pages with database and request-performance diagnostics.
- Production telemetry through Sentry.

Do not invent performance measurements, endorsements, user counts, or product claims.
Record the environment and scope when reporting validation or timing results.

## Product Principles

1. Support recognition and explanation for newcomers without slowing experienced workflows.
2. Reuse working components and existing source before building replacements.
3. Preserve useful information density while keeping controls and essential labels readable.
4. Communicate progress and recovery so users can understand what the application is doing.
5. Validate rendered behavior, mobile use, themes, and interactions rather than relying on HTTP success alone.
