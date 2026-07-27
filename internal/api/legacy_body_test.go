package api

import (
	"strings"
	"testing"
)

type probeBody struct {
	Names []string `json:"names"`
	Type  string   `json:"type,omitempty"`
}

func decodeProbe(t *testing.T, payload string, limit int64) (*probeBody, error) {
	t.Helper()
	return decodeJSONBody[probeBody](
		&legacyRequest{Body: strings.NewReader(payload)}, limit,
	)
}

func TestDecodeJSONBodyReadsTypedFields(t *testing.T) {
	body, err := decodeProbe(t, `{"names":["a","b"],"type":"alliance"}`, defaultBodyLimit)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Names) != 2 || body.Type != "alliance" {
		t.Errorf("body = %+v", body)
	}
}

// Every one of these routes silently ignored extra keys before it was typed.
// Rejecting them now would break callers that send a superset, so leniency is
// the deliberate behaviour rather than an oversight.
func TestDecodeJSONBodyIgnoresUnknownFields(t *testing.T) {
	body, err := decodeProbe(t, `{"names":["a"],"legacyExtra":42}`, defaultBodyLimit)
	if err != nil {
		t.Fatalf("unknown field was rejected: %v", err)
	}
	if len(body.Names) != 1 {
		t.Errorf("body = %+v", body)
	}
}

// The wrong type for a field is a caller mistake worth naming. A bare
// "invalid body" costs a round of guesswork against an API whose source the
// caller cannot read.
func TestDecodeJSONBodyNamesTheOffendingField(t *testing.T) {
	_, err := decodeProbe(t, `{"names":["a"],"type":123}`, defaultBodyLimit)
	if err == nil {
		t.Fatal("a numeric string field was accepted")
	}
	if !strings.Contains(err.Error(), `"type"`) {
		t.Errorf("error does not name the field: %v", err)
	}
}

func TestDecodeJSONBodyRejectsOversizedAndEmptyBodies(t *testing.T) {
	if _, err := decodeProbe(t, `{"names":["`+strings.Repeat("x", 128)+`"]}`, 16); err == nil {
		t.Error("an oversized body was accepted")
	}
	if _, err := decodeProbe(t, ``, defaultBodyLimit); err == nil {
		t.Error("an empty body was accepted")
	}
}

// Trailing content means the caller sent something other than the single
// object the route documents.
func TestDecodeJSONBodyRejectsTrailingContent(t *testing.T) {
	if _, err := decodeProbe(t, `{"names":["a"]}{"names":["b"]}`, defaultBodyLimit); err == nil {
		t.Error("a second JSON value was accepted")
	}
}

// The point of the exercise: the document must describe the body.
func TestTypedRouteCarriesAGeneratedRequestSchema(t *testing.T) {
	document := New(Options{}).document
	operation := document.Paths["/resolve"].Post
	if operation == nil || operation.RequestBody == nil {
		t.Fatal("/resolve has no documented request body")
	}
	media := operation.RequestBody.Content["application/json"]
	if media == nil || media.Schema == nil {
		t.Fatal("/resolve has no application/json schema")
	}
	schema := media.Schema
	if schema.Ref != "" {
		schema = document.Components.Schemas.SchemaFromRef(schema.Ref)
	}
	if _, ok := schema.Properties["names"]; !ok {
		t.Errorf("schema does not describe names: %+v", schema.Properties)
	}
}
