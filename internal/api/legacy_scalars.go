package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Scalars that accept what the previous TypeScript API accepted.
//
// The map-backed handlers read numbers through jsNumber, which coerces
// strings, booleans and null as JavaScript would. `{"alliances":["99000400"]}`
// is therefore a request this API answers today, and a plain []int32 field
// would start rejecting it with a 400.
//
// That coercion is not an accident to be tidied away during a refactor — these
// routes exist for callers written against the TypeScript implementation, and
// a string id is the natural thing for such a caller to send. Typing the
// bodies must not narrow what they accept, so the permissive parse moves into
// the type rather than being dropped.
//
// Each type also supplies its own schema, so the document says the field takes
// a number or a numeric string instead of claiming a strictness the handler
// does not enforce.

// jsInt is an integer that also accepts the JSON forms jsNumber accepts.
type jsInt int64

func (v *jsInt) UnmarshalJSON(data []byte) error {
	number, ok := coerceJSONNumber(data)
	if !ok || math.Trunc(number) != number {
		return fmt.Errorf("expected an integer, got %s", preview(data))
	}
	*v = jsInt(int64(number))
	return nil
}

func (v jsInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(v))
}

// Schema advertises both accepted forms rather than only the canonical one.
func (jsInt) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeInteger, Format: "int64"},
			{Type: huma.TypeString, Pattern: `^-?\d+$`},
		},
		Description: "An integer. A numeric string is accepted for compatibility.",
	}
}

// jsFloat is the same accommodation for values that need not be whole.
type jsFloat float64

func (v *jsFloat) UnmarshalJSON(data []byte) error {
	number, ok := coerceJSONNumber(data)
	if !ok {
		return fmt.Errorf("expected a number, got %s", preview(data))
	}
	*v = jsFloat(number)
	return nil
}

func (v jsFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(v))
}

func (jsFloat) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeNumber},
			{Type: huma.TypeString, Pattern: `^-?\d+(\.\d+)?$`},
		},
		Description: "A number. A numeric string is accepted for compatibility.",
	}
}

// coerceJSONNumber mirrors jsNumber over raw JSON: numbers pass through,
// numeric strings parse, booleans are one and zero, and null is zero.
func coerceJSONNumber(data []byte) (float64, bool) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return 0, false
	}
	switch {
	case bytes.Equal(trimmed, []byte("null")):
		return 0, true
	case bytes.Equal(trimmed, []byte("true")):
		return 1, true
	case bytes.Equal(trimmed, []byte("false")):
		return 0, true
	}

	var raw any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return 0, false
	}
	return jsNumber(raw)
}

// preview keeps an error message short when the offending value is large.
func preview(data []byte) string {
	const limit = 32
	text := string(bytes.TrimSpace(data))
	if len(text) > limit {
		return text[:limit] + "…"
	}
	return text
}

// optional distinguishes a field that was absent from one sent as null.
//
// The patch-style handlers depend on that difference: an absent description
// leaves the stored value alone, while an explicit null clears it. A plain
// *string collapses both to nil and would silently turn every clear into a
// no-op, so the distinction is carried in the type rather than inferred.
type optional[T any] struct {
	Set   bool
	Value *T
}

func (o *optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

func (o optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.Value)
}

// Schema reports the wrapped type, marked nullable because null is meaningful
// rather than merely tolerated.
func (optional[T]) Schema(r huma.Registry) *huma.Schema {
	var zero T
	schema := huma.SchemaFromType(r, reflect.TypeOf(zero))
	if schema == nil {
		return &huma.Schema{}
	}
	schema.Nullable = true
	return schema
}

// present reports whether the caller sent the field at all.
func (o optional[T]) present() bool { return o.Set }

// valueOr returns the sent value, or the fallback when absent or null.
func (o optional[T]) valueOr(fallback T) T {
	if o.Value == nil {
		return fallback
	}
	return *o.Value
}
