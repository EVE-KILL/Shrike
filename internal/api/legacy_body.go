package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Typed request bodies for the write surface.
//
// registerLegacy takes the raw huma.Context and sets SkipValidateBody, so Huma
// never sees an input type and the document carried no requestBody at all: 80
// write operations documented nothing about what they accept, and Try it on
// the reference page could only send {}.
//
// The fix is not to move these onto huma.Register. That would also take over
// response serialization, and the legacy handlers deliberately own that — the
// API being replaced returns {"error":"..."} rather than RFC 9457 problem
// details, and consumers depend on it. So the body alone becomes typed: the
// schema is generated from the same struct the handler decodes into, which
// makes the struct the single source of truth rather than a second one that
// drifts from the parsing.
//
// Body limits stay explicit per route. They were chosen per endpoint and are a
// denial-of-service control, not a detail to be inherited from a default.

// defaultBodyLimit bounds a request whose route does not choose its own.
const defaultBodyLimit = 1 << 20

// bodyHandler receives the decoded body alongside the raw request, so a
// handler keeps its access to query values, path parameters, and the session.
type bodyHandler[T any] func(context.Context, *legacyRequest, *T) (legacyPayload, error)

// registerLegacyJSON mounts a write route whose body is described by T.
//
// The generated schema is attached to the operation, so the reference page and
// every generated client see the real shape. Decoding uses the same type, so a
// field renamed in Go is renamed in the document by construction.
func registerLegacyJSON[T any](
	a huma.API,
	op huma.Operation,
	limit int64,
	handler bodyHandler[T],
) {
	if limit <= 0 {
		limit = defaultBodyLimit
	}

	var zero T
	schema := huma.SchemaFromType(
		a.OpenAPI().Components.Schemas,
		reflect.TypeOf(zero),
	)
	if op.RequestBody == nil {
		op.RequestBody = &huma.RequestBody{
			Required: true,
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: schema},
			},
		}
	}

	registerLegacy(a, op, func(
		ctx context.Context,
		req *legacyRequest,
	) (legacyPayload, error) {
		body, err := decodeJSONBody[T](req, limit)
		if err != nil {
			return legacyPayload{}, err
		}
		return handler(ctx, req, body)
	})
}

// decodeJSONBody reads one JSON value of type T, refusing anything larger than
// limit and anything with trailing content.
//
// The error text stays in the {"error":"..."} shape the rest of the API uses.
// Reporting the offending field is deliberate: a 400 that only says "invalid
// body" costs the caller a round of guesswork against an API they cannot read
// the source of.
func decodeJSONBody[T any](req *legacyRequest, limit int64) (*T, error) {
	if req.Body == nil {
		return nil, apiError(http.StatusBadRequest, "Request body is required")
	}

	limited := io.LimitReader(req.Body, limit+1)
	decoder := json.NewDecoder(limited)
	// Unknown fields are ignored, not rejected. The API is live and public,
	// and every one of these routes accepted extra keys before it was typed —
	// turning that into a 400 would break callers that send a superset today.
	// The generated schema still lists exactly the fields that mean anything,
	// so the documentation stays honest about what is read.

	var body T
	if err := decoder.Decode(&body); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, apiError(http.StatusBadRequest, "Request body is required")
		}
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return nil, apiError(http.StatusBadRequest, fmt.Sprintf(
				"Field %q must be a %s", typeErr.Field, typeErr.Type,
			))
		}
		return nil, apiError(http.StatusBadRequest, "Request body is not valid JSON")
	}

	// A second value in the stream means the caller sent something other than
	// the single JSON object the route accepts.
	if decoder.More() {
		return nil, apiError(
			http.StatusBadRequest, "Request body must be a single JSON value",
		)
	}
	if counted, ok := limited.(*io.LimitedReader); ok && counted.N <= 0 {
		return nil, apiError(http.StatusRequestEntityTooLarge, "Request body is too large")
	}
	return &body, nil
}
