package sde

import (
	"slices"
	"testing"
)

// The nested members are where a mapping mistake is least visible: they produce
// far more rows than they read, so a wrong field name yields "0 rows" rather
// than an error.
func TestNestedTableDeclarationsAreConsistent(t *testing.T) {
	for _, tbl := range NestedTables {
		if tbl.Expand == nil {
			t.Errorf("nested table %s has no Expand function", tbl.Name)
			continue
		}
		if tbl.Values != nil {
			t.Errorf("nested table %s sets both Values and Expand", tbl.Name)
		}
		if !tbl.PruneAbsent {
			t.Errorf("nested table %s is not authoritative; removed child rows would remain stale", tbl.Name)
		}
		for _, pk := range tbl.PK {
			found := slices.Contains(tbl.Columns, pk)
			if !found {
				t.Errorf("nested table %s: primary key %q is not among its columns", tbl.Name, pk)
			}
		}
		// A row with no usable key must produce nothing.
		if rows := tbl.Expand(Row{}); len(rows) != 0 {
			t.Errorf("nested table %s: Expand returned %d rows for a keyless record", tbl.Name, len(rows))
		}
	}
}

func TestExpandTypeDogmaAttributes(t *testing.T) {
	tbl := *TableByName("type_dogma_attributes")
	rows := tbl.Expand(Row{
		"_key": 18.0,
		"dogmaAttributes": []any{
			map[string]any{"attributeID": 182.0, "value": 3386.0},
			map[string]any{"attributeID": 277.0, "value": 1.0},
			map[string]any{"value": 5.0}, // no attributeID — must be skipped
		},
	})
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (the entry without an attributeID is skipped)", len(rows))
	}
	if rows[0][0] != int32(18) || rows[0][1] != int32(182) {
		t.Errorf("first row = %v, want type 18 attribute 182", rows[0][:2])
	}
}

// One blueprint record fans out across five tables, keyed by the activity name.
func TestExpandBlueprintActivities(t *testing.T) {
	bp := Row{
		"_key": 681.0,
		"activities": map[string]any{
			"copying": map[string]any{"time": 480.0},
			"manufacturing": map[string]any{
				"time":      600.0,
				"materials": []any{map[string]any{"typeID": 38.0, "quantity": 86.0}},
				"products":  []any{map[string]any{"typeID": 165.0, "quantity": 1.0, "probability": 0.3}},
				"skills":    []any{map[string]any{"typeID": 11442.0, "level": 1.0}},
			},
		},
	}

	if got := len(TableByName("blueprint_activities").Expand(bp)); got != 2 {
		t.Errorf("activities: got %d rows, want 2", got)
	}

	mats := TableByName("blueprint_activity_materials").Expand(bp)
	if len(mats) != 1 {
		t.Fatalf("materials: got %d rows, want 1", len(mats))
	}
	// Nested entries use "typeID" where typeMaterials uses "materialTypeID";
	// reading the wrong one yields zero rows rather than an error.
	if mats[0][2] != int32(38) {
		t.Errorf("material type = %v, want 38", mats[0][2])
	}
	if mats[0][1] != "manufacturing" {
		t.Errorf("activity = %v, want manufacturing", mats[0][1])
	}

	prods := TableByName("blueprint_activity_products").Expand(bp)
	if len(prods) != 1 || prods[0][2] != int32(165) {
		t.Errorf("products = %v, want one row for type 165", prods)
	}

	skills := TableByName("blueprint_activity_skills").Expand(bp)
	if len(skills) != 1 || skills[0][2] != int32(11442) {
		t.Errorf("skills = %v, want one row for type 11442", skills)
	}
}

// Planet names are Roman numerals; anything past the table falls back to
// decimal rather than producing a wrong name.
func TestToRoman(t *testing.T) {
	cases := map[int32]string{1: "I", 4: "IV", 9: "IX", 10: "X", 20: "XX", 30: "XXX"}
	for in, want := range cases {
		if got := toRoman(in); got != want {
			t.Errorf("toRoman(%d) = %q, want %q", in, got, want)
		}
	}
	if got := toRoman(99); got != "99" {
		t.Errorf("toRoman(99) = %q, want the decimal fallback", got)
	}
}

// The manual list is the authority for hulls the market cannot price, so a
// duplicate or a truncated value would be written silently.
func TestManualPricesAreSaneAndUnique(t *testing.T) {
	seen := map[int32]bool{}
	for _, p := range manualPrices {
		if seen[p.TypeID] {
			t.Errorf("type %d appears twice in the manual price list", p.TypeID)
		}
		seen[p.TypeID] = true
		if p.Price <= 0 {
			t.Errorf("type %d has a non-positive price %v", p.TypeID, p.Price)
		}
	}
	// Several entries are pinned to 0.01; an integer type would truncate them.
	found := false
	for _, p := range manualPrices {
		if p.Price > 0 && p.Price < 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("no sub-1 ISK entries — the 0.01 overrides were lost")
	}
}

func TestInvFlagsAreUniqueAndNamed(t *testing.T) {
	seen := map[int32]bool{}
	for _, f := range invFlags {
		if seen[f.ID] {
			t.Errorf("flag %d declared twice", f.ID)
		}
		seen[f.ID] = true
		if f.Name == "" {
			t.Errorf("flag %d has no name", f.ID)
		}
	}
	if len(invFlags) != 78 {
		t.Errorf("got %d flags, want the 78 present in production", len(invFlags))
	}
}
