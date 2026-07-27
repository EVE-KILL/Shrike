package api

import (
	"encoding/json"
	"testing"
)

// These routes were written against a TypeScript API that coerced. A caller
// sending "99000400" has always worked, so typing the body must not start
// rejecting it.
func TestJSIntAcceptsTheFormsTheOldAPIAccepted(t *testing.T) {
	for raw, want := range map[string]int64{
		`99000400`:   99000400,
		`"99000400"`: 99000400,
		`true`:       1,
		`false`:      0,
		`null`:       0,
		`-7`:         -7,
	} {
		var value jsInt
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			t.Errorf("%s was rejected: %v", raw, err)
			continue
		}
		if int64(value) != want {
			t.Errorf("%s = %d, want %d", raw, int64(value), want)
		}
	}
}

func TestJSIntRejectsValuesWithNoNumericReading(t *testing.T) {
	for _, raw := range []string{`"abc"`, `{}`, `[]`, `1.5`} {
		var value jsInt
		if err := json.Unmarshal([]byte(raw), &value); err == nil {
			t.Errorf("%s was accepted as %d", raw, int64(value))
		}
	}
}

func TestJSFloatAcceptsNumericStrings(t *testing.T) {
	var value jsFloat
	if err := json.Unmarshal([]byte(`"1.5"`), &value); err != nil {
		t.Fatalf("numeric string rejected: %v", err)
	}
	if float64(value) != 1.5 {
		t.Errorf("value = %v, want 1.5", float64(value))
	}
}

// The document must not claim a strictness the decoder does not enforce.
func TestJSScalarsDocumentBothAcceptedForms(t *testing.T) {
	schema := jsInt(0).Schema(nil)
	if len(schema.OneOf) != 2 {
		t.Fatalf("schema does not offer both forms: %+v", schema)
	}
	if schema.OneOf[0].Type != "integer" || schema.OneOf[1].Type != "string" {
		t.Errorf("schema forms = %v, %v", schema.OneOf[0].Type, schema.OneOf[1].Type)
	}
}
