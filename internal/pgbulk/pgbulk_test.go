package pgbulk

import (
	"strings"
	"testing"
)

// The merge statement decides whether an import corrects stored data or leaves
// it alone, which is the difference between refreshing a snapshot and silently
// rewriting history. It is built by string concatenation, so its shape is
// pinned here rather than discovered in production.
func TestMergeSQL(t *testing.T) {
	columns := []string{"type_id", "region_id", "date", "average"}
	pk := []string{"type_id", "region_id", "date"}

	update := MergeSQL("prices", "staging", columns, pk, DoUpdate)
	if !strings.Contains(update, "ON CONFLICT (type_id, region_id, date) DO UPDATE SET average = EXCLUDED.average") {
		t.Errorf("DoUpdate did not target the non-key column:\n%s", update)
	}
	// Key columns must never appear in the SET clause: assigning them is
	// pointless and, if it were EXCLUDED, would be a no-op that hides a bug.
	for _, k := range pk {
		if strings.Contains(update, k+" = EXCLUDED."+k) {
			t.Errorf("key column %s appears in the SET clause:\n%s", k, update)
		}
	}

	nothing := MergeSQL("prices", "staging", columns, pk, DoNothing)
	if !strings.HasSuffix(nothing, "ON CONFLICT (type_id, region_id, date) DO NOTHING") {
		t.Errorf("DoNothing produced:\n%s", nothing)
	}
	if strings.Contains(nothing, "DO UPDATE") {
		t.Errorf("DoNothing must not update:\n%s", nothing)
	}

	// Without DISTINCT ON, a key repeated inside one archive aborts the whole
	// statement with "cannot affect row a second time".
	for _, sql := range []string{update, nothing} {
		if !strings.Contains(sql, "SELECT DISTINCT ON (type_id, region_id, date)") {
			t.Errorf("missing the duplicate-key guard:\n%s", sql)
		}
	}
}

// A table whose every column is part of the key has nothing to update, and
// asking for DoUpdate must not produce an empty SET clause.
func TestMergeSQLAllKeyColumns(t *testing.T) {
	columns := []string{"war_id", "alliance_id"}
	sql := MergeSQL("war_allies", "staging", columns, columns, DoUpdate)
	if !strings.HasSuffix(sql, "DO NOTHING") {
		t.Errorf("expected a fallback to DO NOTHING, got:\n%s", sql)
	}
	if strings.Contains(sql, "SET") {
		t.Errorf("produced an empty SET clause:\n%s", sql)
	}
}

// A row of the wrong width would misalign every column after it, so it is
// rejected at Add rather than at COPY where the error names only the table.
func TestCopierRejectsMisshapenRow(t *testing.T) {
	c := NewCopier(t.Context(), nil, "staging", []string{"a", "b", "c"})
	if err := c.Add([]any{1, 2}); err == nil {
		t.Error("expected an error for a short row")
	}
	if err := c.Add([]any{1, 2, 3, 4}); err == nil {
		t.Error("expected an error for a long row")
	}
	if c.Written() != 0 {
		t.Errorf("rejected rows were counted: %d", c.Written())
	}
}
