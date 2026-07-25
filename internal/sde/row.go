package sde

import "strconv"

// Row is one decoded JSONL object.
//
// SDE JSON is loose in ways that matter: every number decodes as float64, some
// numeric fields arrive as strings, booleans are sometimes 0/1, and localised
// text is either a bare string or a {lang: text} map. The accessors below are
// the single place those quirks are handled, so table definitions stay
// declarative.
//
// All of them return pointers, because "absent" and "zero" are different things
// here — a missing volume must become SQL NULL, not 0.0, or downstream
// arithmetic silently treats unknown as free.
type Row map[string]any

// Key returns the row's primary identifier. Every SDE record carries "_key".
//
// Zero is rejected. The archive ships placeholder records at _key 0 — a
// "#System" entry appears in types, groups and categories — and importing them
// is actively harmful, not merely untidy: EVE data uses 0 as the "no entity"
// sentinel, so a real inv_types row with type_id 0 would make every join
// against an absent ship or item spuriously resolve to "#System".
//
// This also matches the TypeScript importer, whose `if (!id) continue` drops
// zero as falsy.
func (r Row) Key() (int64, bool) {
	v, ok := r["_key"]
	if !ok {
		return 0, false
	}
	n := toInt64(v)
	if n == nil || *n == 0 {
		return 0, false
	}
	return *n, true
}

// Lang extracts localised text, preferring English.
//
// Falls back to any other non-empty translation rather than returning nothing:
// a name in Japanese is more useful than NULL, and a handful of SDE records
// genuinely lack "en".
func (r Row) Lang(field string) *string {
	v, ok := r[field]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case string:
		return emptyToNil(t)
	case map[string]any:
		if s, ok := t["en"].(string); ok && s != "" {
			return &s
		}
		for _, val := range t {
			if s, ok := val.(string); ok && s != "" {
				return &s
			}
		}
	}
	return nil
}

// Str returns a bare string field.
func (r Row) Str(field string) *string {
	v, ok := r[field]
	if !ok {
		return nil
	}
	if s, ok := v.(string); ok {
		return emptyToNil(s)
	}
	return nil
}

// Int returns an integer field as int32, the width sqlc generates for Postgres
// `integer` columns. Values beyond int32 are dropped rather than silently
// wrapped — a wrapped type ID would corrupt a join.
func (r Row) Int(field string) *int32 {
	n := r.Int64(field)
	if n == nil {
		return nil
	}
	if *n > 2147483647 || *n < -2147483648 {
		return nil
	}
	v := int32(*n)
	return &v
}

// Int64 returns an integer field at full width, for bigint columns.
func (r Row) Int64(field string) *int64 {
	v, ok := r[field]
	if !ok {
		return nil
	}
	return toInt64(v)
}

// Float returns a numeric field, accepting numeric strings.
func (r Row) Float(field string) *float64 {
	v, ok := r[field]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case float64:
		return normalizeZero(&t)
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return normalizeZero(&f)
		}
	case bool:
		// Not expected, but coercing rather than erroring matches the
		// permissiveness of the TypeScript importer this replaces.
		f := 0.0
		if t {
			f = 1.0
		}
		return &f
	}
	return nil
}

// Bool returns a boolean field, accepting 0/1 and "true"/"false".
func (r Row) Bool(field string) *bool {
	v, ok := r[field]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case bool:
		return &t
	case float64:
		b := t != 0
		return &b
	case string:
		switch t {
		case "true", "True", "1":
			b := true
			return &b
		case "false", "False", "0":
			b := false
			return &b
		}
	}
	return nil
}

// List returns an array field for callers that need to walk nested structures
// (blueprint activities, dogma attribute lists).
func (r Row) List(field string) []any {
	v, ok := r[field]
	if !ok {
		return nil
	}
	if a, ok := v.([]any); ok {
		return a
	}
	return nil
}

// Map returns an object field, used the same way as List.
func (r Row) Map(field string) map[string]any {
	v, ok := r[field]
	if !ok {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func toInt64(v any) *int64 {
	switch t := v.(type) {
	case float64:
		n := int64(t)
		return &n
	case string:
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return &n
		}
		// Some fields are numeric-looking floats in string form ("42.0").
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			n := int64(f)
			return &n
		}
	case bool:
		var n int64
		if t {
			n = 1
		}
		return &n
	}
	return nil
}

// emptyToNil maps "" to NULL.
//
// This is a deliberate divergence from the TypeScript importer, which writes
// empty strings (its extractLanguageField returns "" and only `undefined` is
// stripped). Production therefore has ” in roughly 600 places where this
// writes NULL — inv_market_groups.description alone accounts for 524.
//
// NULL is the more honest encoding: absent text and empty text are not
// different states here, and nothing depends on telling them apart. Both are
// falsy in JavaScript, so rendering is unaffected, and a search of the codebase
// found no SQL comparing these columns to ”. Rows converge to NULL as they are
// re-imported.
func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// normalizeZero collapses negative zero.
//
// A handful of SDE coordinates are -0.0, and Go preserves the sign bit where
// the TypeScript path did not. IEEE 754 says -0 == 0, so this changes no
// comparison, join, or arithmetic — but it does change the rendered text, which
// would make every future checksum comparison against production report a
// false difference on 12 solar systems.
func normalizeZero(f *float64) *float64 {
	if f != nil && *f == 0 {
		zero := 0.0
		return &zero
	}
	return f
}
