package sde

import "testing"

// Key must reject 0. EVE uses 0 as the "no entity" sentinel, so importing the
// archive's "#System" placeholder at _key 0 would make every join against an
// absent ship or item resolve to it.
func TestKeyRejectsZeroAndMissing(t *testing.T) {
	if _, ok := (Row{"_key": 0.0}).Key(); ok {
		t.Error("Key() accepted 0; the #System placeholder must be skipped")
	}
	if _, ok := (Row{}).Key(); ok {
		t.Error("Key() accepted a row with no _key")
	}
	if _, ok := (Row{"_key": "abc"}).Key(); ok {
		t.Error("Key() accepted a non-numeric _key")
	}
	got, ok := (Row{"_key": 587.0}).Key()
	if !ok || got != 587 {
		t.Errorf("Key() = (%d, %v); want (587, true)", got, ok)
	}
	// JSON numbers decode as float64, but a string id must work too.
	if got, ok := (Row{"_key": "30000142"}).Key(); !ok || got != 30000142 {
		t.Errorf("Key() on a string = (%d, %v); want (30000142, true)", got, ok)
	}
}

func TestLangPrefersEnglish(t *testing.T) {
	r := Row{"name": map[string]any{"de": "Rifter DE", "en": "Rifter", "ja": "ライフター"}}
	got := r.Lang("name")
	if got == nil || *got != "Rifter" {
		t.Errorf("Lang() = %v; want Rifter", deref(got))
	}

	// Some records genuinely lack "en"; any translation beats NULL.
	r = Row{"name": map[string]any{"ja": "ライフター"}}
	if got := r.Lang("name"); got == nil || *got != "ライフター" {
		t.Errorf("Lang() = %v; want the Japanese fallback", deref(got))
	}

	// A bare string, not a language map.
	if got := (Row{"name": "Plain"}).Lang("name"); got == nil || *got != "Plain" {
		t.Errorf("Lang() on a bare string = %v; want Plain", deref(got))
	}

	// Absent and empty both become NULL.
	if got := (Row{}).Lang("name"); got != nil {
		t.Errorf("Lang() on a missing field = %v; want nil", deref(got))
	}
	if got := (Row{"name": map[string]any{"en": ""}}).Lang("name"); got != nil {
		t.Errorf("Lang() on an empty string = %v; want nil", deref(got))
	}
}

// A wrapped ID would silently corrupt a join, so out-of-range values are dropped
// rather than truncated.
func TestIntRejectsOutOfRange(t *testing.T) {
	if got := (Row{"v": 3000000000.0}).Int("v"); got != nil {
		t.Errorf("Int() = %v for a value past int32; want nil", *got)
	}
	if got := (Row{"v": 2147483647.0}).Int("v"); got == nil || *got != 2147483647 {
		t.Errorf("Int() = %v; want int32 max to be accepted", got)
	}
	// The same value is fine at 64-bit width.
	if got := (Row{"v": 3000000000.0}).Int64("v"); got == nil || *got != 3000000000 {
		t.Errorf("Int64() = %v; want 3000000000", got)
	}
}

// Negative zero compares equal to zero but renders differently, which would
// make every checksum comparison against production report a false difference.
func TestFloatNormalizesNegativeZero(t *testing.T) {
	got := (Row{"v": negZero()}).Float("v")
	if got == nil {
		t.Fatal("Float() = nil")
	}
	if signbit(*got) {
		t.Errorf("Float() preserved negative zero; want +0")
	}
	// Ordinary values are untouched.
	if got := (Row{"v": -1.5}).Float("v"); got == nil || *got != -1.5 {
		t.Errorf("Float() = %v; want -1.5", got)
	}
	// Numeric strings are accepted.
	if got := (Row{"v": "2.5"}).Float("v"); got == nil || *got != 2.5 {
		t.Errorf("Float() on a string = %v; want 2.5", got)
	}
}

func TestBoolAcceptsNumericAndString(t *testing.T) {
	cases := []struct {
		in   any
		want *bool
	}{
		{true, ptr(true)},
		{false, ptr(false)},
		{1.0, ptr(true)},
		{0.0, ptr(false)},
		{"true", ptr(true)},
		{"false", ptr(false)},
		{"nonsense", nil},
	}
	for _, tc := range cases {
		got := (Row{"v": tc.in}).Bool("v")
		switch {
		case tc.want == nil && got != nil:
			t.Errorf("Bool(%v) = %v; want nil", tc.in, *got)
		case tc.want != nil && got == nil:
			t.Errorf("Bool(%v) = nil; want %v", tc.in, *tc.want)
		case tc.want != nil && got != nil && *got != *tc.want:
			t.Errorf("Bool(%v) = %v; want %v", tc.in, *got, *tc.want)
		}
	}
}

// An empty array must become NULL, not '{}' — the TypeScript importer omitted
// the field entirely in that case.
func TestInt32Slice(t *testing.T) {
	if got := int32Slice(nil); got != nil {
		t.Errorf("int32Slice(nil) = %v; want nil", got)
	}
	if got := int32Slice([]any{}); got != nil {
		t.Errorf("int32Slice(empty) = %v; want nil", got)
	}
	got := int32Slice([]any{3.0, 5.0, 7.0})
	if len(got) != 3 || got[0] != 3 || got[2] != 7 {
		t.Errorf("int32Slice = %v; want [3 5 7]", got)
	}
}

// Every declaration must be internally consistent, or CopyFrom fails at runtime
// with a column-count mismatch on a table nobody exercised.
func TestTableDeclarationsAreConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, tbl := range Tables {
		if tbl.Member == "" || tbl.Name == "" {
			t.Errorf("table %+v has an empty member or name", tbl)
		}
		if seen[tbl.Name] {
			t.Errorf("table %s declared twice", tbl.Name)
		}
		seen[tbl.Name] = true

		if len(tbl.PK) == 0 {
			t.Errorf("table %s has no primary key", tbl.Name)
		}
		for _, pk := range tbl.PK {
			found := false
			for _, c := range tbl.Columns {
				if c == pk {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("table %s: primary key %q is not among its columns", tbl.Name, pk)
			}
		}

		// Values must return exactly one entry per declared column.
		row := Row{"_key": 1.0}
		vals, ok := tbl.Values(row)
		if !ok {
			t.Errorf("table %s: Values rejected a row with a valid _key", tbl.Name)
			continue
		}
		if len(vals) != len(tbl.Columns) {
			t.Errorf("table %s: Values returned %d values for %d columns",
				tbl.Name, len(vals), len(tbl.Columns))
		}

		// And must skip a row with no usable key.
		if _, ok := tbl.Values(Row{}); ok {
			t.Errorf("table %s: Values accepted a row with no _key", tbl.Name)
		}
	}
}

func TestMergeSQLExcludesPrimaryKeyFromUpdate(t *testing.T) {
	tbl := Table{
		Name:    "inv_types",
		PK:      []string{"type_id"},
		Columns: []string{"type_id", "name"},
	}
	got := mergeSQL(tbl, "staging")
	if contains(got, "type_id = EXCLUDED.type_id") {
		t.Errorf("merge assigns the conflict target:\n%s", got)
	}
	if !contains(got, "name = EXCLUDED.name") {
		t.Errorf("merge does not update a non-key column:\n%s", got)
	}
	// Duplicate keys within one archive member would otherwise abort the merge.
	if !contains(got, "DISTINCT ON (type_id)") {
		t.Errorf("merge lacks the duplicate-key guard:\n%s", got)
	}
}

func ptr[T any](v T) *T { return &v }

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func negZero() float64 {
	z := 0.0
	return -z
}

func signbit(f float64) bool {
	// A direct check avoids importing math for one call: -0 is the only value
	// where 1/f is negative infinity while f == 0.
	return f == 0 && 1/f < 0
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
