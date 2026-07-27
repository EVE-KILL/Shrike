# OpenAPI code generation

The Go/Huma document is EVE-KILL's only API contract. Normal development runs
`make gen-api-client` from the repository root; it regenerates the committed
OpenAPI JSON and the framework-independent TypeScript types in
`web/shared/api/`.

The generator is isolated from the Nuxt dependency graph and version-pinned so
frontend dependency updates cannot silently reshape the contract.
