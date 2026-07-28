# API contract

This document states where each part of the OpenAPI document comes from, and
what to change when you add a route or a parameter.

Shrike serves one OpenAPI 3.1 document at `/api/openapi.json`. Scalar renders it
at `/api/docs`. The Nuxt reference page at `/docs` reads the same document. The
generated TypeScript client in `web/shared/api/` is built from it.

## Contract

Most routes register through `registerLegacy` in `internal/api/legacy.go`. That
function sets `SkipValidateParams` and `SkipValidateBody`, then hands the raw
`huma.Context` to the handler. Huma therefore never builds an input struct, and
nothing about the request reaches the document by itself.

Five sources fill the gap. Each one is applied in `api.New` after the routes
register.

| Part | Source | File |
| --- | --- | --- |
| Tags and groups | Path structure | `internal/api/openapi_taxonomy.go` |
| Tag prose | Table keyed by tag | `internal/api/openapi_tags.go` |
| Operation prose | Table keyed by operation ID | `internal/api/openapi_descriptions.go` |
| Query parameters | Table keyed by operation ID | `internal/api/openapi_query_operations.go` |
| Request bodies | Runtime or documentation-only Go wire struct | `internal/api/legacy_body.go`, `internal/api/openapi_body_types.go` |

`registerLegacyJSON` generates a schema from the same struct the handler
decodes. Routes whose compatibility parser needs `json.RawMessage` instead use
`documentJSONBody` with a documentation-only wire struct. This preserves the
old coercion and presence semantics while still producing concrete nested
types, enums, and required fields.

Successful JSON responses are described by
`internal/api/response_schemas.go` and the focused
`internal/api/response_schemas_*.go` catalogues. These catalogues cover both
the public Bun API compatibility surface and the former Nuxt server API.
Redirects and binary image responses have media-specific response entries
rather than a JSON fallback.

## Defaults

A table entry only fills a gap. A description or parameter set at the
registration site wins, and `applyOperationParameters` appends to the path
parameters rather than replacing them.

Query parameters are documentation. `SkipValidateParams` stays on, handlers keep
reading `req.Query`, and an undeclared parameter is accepted exactly as before.
A wrong entry misleads a reader; it cannot change a response.

Record the bounds the handler applies. `limitQuery(50, 10, 100)` means the
handler called `boundedQueryInt(req, "limit", 50, 10, 100)`. Most handlers clamp
rather than reject, so the maximum is the value the caller gets.

## Procedure

Follow these steps when a handler reads a new query parameter.

1. Open `internal/api/openapi_query_operations.go`.
2. Find the operation ID in `operationQueryParameters`.
3. Add a constructor from `internal/api/openapi_query.go`, such as `enumQuery`.
4. Copy the default and the bounds from the parsing site.
5. Set the entity kind with `entityQuery` when the value is an entity ID.
6. Run `make gen-api-client` to regenerate the committed contract.
7. Commit `web/shared/api.openapi.json` and `web/shared/api/`.

## Verification

Run `go test ./internal/api/`. The core tests guarding the query table are:

- `TestEveryQueryParameterHandlersReadIsDeclared` parses the package source and
  fails when a handler reads a name no operation declares.
- `TestOpenAPIParametersMatchLiveOperations` fails on an operation ID that no
  longer exists.
- `TestOpenAPIQueryParametersAreDescribed` fails on a parameter with no
  description or no schema type.
- `TestOpenAPIEntityParametersNameAKnownKind` fails on an `x-entity` value the
  reference page does not recognize.
- `TestQuerySchemasMatchNumericHandlerInputs` checks the parameters whose
  integer-versus-number distinction changes generated clients.
- `TestQueriesRejectedWhenOmittedAreRequired` checks required query fields.
- `TestOpenAPISuccessResponsesUseConcreteSchemas` rejects the old free-form
  JSON response fallback.
- `TestOpenAPIDocumentsRedirectsAndDomainImagesByMediaType` checks that browser
  redirects have no JSON body and that domain images are binary image media.

Run `bun test test` in `web/` to check the reference page still reads them.

Run `make check-api-client` to confirm the committed contract matches the code.

## Failure modes

The source scan in `TestEveryQueryParameterHandlersReadIsDeclared` is
package-wide, not per operation. Handlers are shared across routes and reached
through dispatch tables, so the scan cannot attribute a name to one operation.
It catches a parameter documented nowhere. It does not catch a parameter
documented on the wrong operation.

The scan reads string literals. A handler that builds a parameter name at run
time is invisible to it.

## Source

- `internal/api/openapi_query.go`: constructors and the apply and check
  functions.
- `internal/api/openapi_query_operations.go`: the parameter table.
- `internal/api/openapi_query_test.go`: the guards listed above.
- `internal/cli/openapi.go`: the `shrike openapi-spec` command.
