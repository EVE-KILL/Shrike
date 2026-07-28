package api

import (
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
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

// Every write route that reads a JSON body must document one. The routes that
// stay undocumented are the action endpoints that read no body at all and the
// two multipart uploads; anything else appearing here is a route whose schema
// was forgotten, which the reference page shows as an untyped Try it form.
func TestEveryJSONWriteRouteDocumentsItsBody(t *testing.T) {
	bodyless := map[string]bool{
		"announcement-admin-archive-compat": true,
		"admin-domain-toggle":               true,
		"admin-users-toggle-admin":          true,
		"wallet-admin-sync":                 true,
		"announcement-dismiss-compat":       true,
		"auth-logout-legacy":                true,
		"campaign-prize-claim":              true,
		"campaign-prize-claim-legacy":       true,
		"account-announcement-dismissal":    true,
		"admin-comment-hide-live-alias":     true,
		"admin-comment-restore-live-alias":  true,
		"other-sessions-revoke-legacy":      true,
		"user-session-revoke-legacy":        true,
		"domain-asset-upload":               true, // multipart
		"domain-asset-upload-compat":        true, // multipart
	}

	document := New(Options{}).document
	var undocumented []string
	for _, item := range document.Paths {
		for _, operation := range []*huma.Operation{item.Post, item.Put, item.Patch} {
			if operation == nil || bodyless[operation.OperationID] {
				continue
			}
			media := (*huma.MediaType)(nil)
			if operation.RequestBody != nil {
				media = operation.RequestBody.Content["application/json"]
			}
			if media == nil || media.Schema == nil {
				undocumented = append(undocumented, operation.OperationID)
			}
		}
	}
	sort.Strings(undocumented)
	if len(undocumented) > 0 {
		t.Errorf("write routes with no documented body: %v", undocumented)
	}
}

func TestReadRoutesNeverDocumentJSONBodies(t *testing.T) {
	document := New(Options{}).document
	for path, item := range document.Paths {
		for _, operation := range []*huma.Operation{item.Get, item.Head} {
			if operation != nil && operation.RequestBody != nil {
				t.Errorf("%s %s unexpectedly documents a request body",
					operation.Method, path)
			}
		}
	}
}

func TestForcedModerationActionsMatchTheirLiveBodies(t *testing.T) {
	document := New(Options{}).document

	for _, path := range []string{
		"/admin/comments/{id}/hide",
		"/admin/comments/{id}/restore",
	} {
		if body := document.Paths[path].Post.RequestBody; body != nil {
			t.Errorf("%s documents a body the handler does not read", path)
		}
	}

	for _, path := range []string{
		"/admin/moderation/{id}/approve",
		"/admin/moderation/{id}/reject",
	} {
		body := document.Paths[path].Post.RequestBody
		if body == nil {
			t.Fatalf("%s has no optional notes body", path)
		}
		if body.Required {
			t.Errorf("%s requires a body, but the live route accepts none", path)
		}
	}
}

func TestDomainAssetDeleteDocumentsBodyOnlyOnBodyAlias(t *testing.T) {
	document := New(Options{}).document
	if body := document.Paths["/me/domains/{id}/assets"].Delete.RequestBody; body != nil {
		t.Error("canonical domain asset delete documents a body it does not read")
	}
	body := document.Paths["/user/domains/{id}/upload"].Delete.RequestBody
	if body == nil || !body.Required {
		t.Error("compatibility domain asset delete lost its required JSON body")
	}
}

func TestBatchStatsDocumentsTheImplementedPeriods(t *testing.T) {
	document := New(Options{}).document
	schema := document.Paths["/characters/stats"].Post.
		RequestBody.Content["application/json"].Schema
	period := schema.Properties["type"]
	got := make([]string, 0, len(period.Enum))
	for _, value := range period.Enum {
		got = append(got, value.(string))
	}
	want := []string{"alltime", "weekly", "range"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("period enum = %v, want %v", got, want)
	}
}
