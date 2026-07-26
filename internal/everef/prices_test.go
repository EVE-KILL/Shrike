package everef

import "testing"

// An empty or zero field has to become NULL rather than a stored zero: a price
// of zero would drag every valuation that reads it down, and production holds
// neither. Getting this wrong is invisible until something is priced at nothing.
func TestCSVCoercion(t *testing.T) {
	if csvFloat("") != nil {
		t.Error("csvFloat(empty) should be nil")
	}
	if csvFloat("0") != nil {
		t.Error("csvFloat(zero) should be nil")
	}
	if got := csvFloat("3.98"); got == nil || *got != 3.98 {
		t.Errorf("csvFloat = %v, want 3.98", got)
	}
	if csvInt("") != nil {
		t.Error("csvInt(empty) should be nil")
	}
	if csvInt("0") != nil {
		t.Error("csvInt(zero) should be nil")
	}
	if got := csvInt("12"); got == nil || *got != 12 {
		t.Errorf("csvInt = %v, want 12", got)
	}
	// Counts occasionally arrive as floats.
	if got := csvInt("12.0"); got == nil || *got != 12 {
		t.Errorf("csvInt(12.0) = %v, want 12", got)
	}
}

func TestDateRange(t *testing.T) {
	got, err := DateRange("2026-07-23", "2026-07-26")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-07-23", "2026-07-24", "2026-07-25", "2026-07-26"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("day %d = %s, want %s", i, got[i], want[i])
		}
	}

	// A range crossing a month and a leap day must not lose or repeat a day.
	feb, err := DateRange("2024-02-27", "2024-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(feb) != 4 || feb[2] != "2024-02-29" {
		t.Errorf("leap range = %v", feb)
	}

	// An inverted range is empty, not an error: callers use it to mean
	// "nothing to do".
	if empty, err := DateRange("2026-07-26", "2026-07-23"); err != nil || len(empty) != 0 {
		t.Errorf("inverted range = %v, %v", empty, err)
	}

	if _, err := DateRange("not-a-date", "2026-07-23"); err == nil {
		t.Error("expected an error for an unparseable date")
	}
}

func TestDayAfter(t *testing.T) {
	if got, _ := DayAfter("2026-07-31"); got != "2026-08-01" {
		t.Errorf("month boundary = %s", got)
	}
	if got, _ := DayAfter("2026-12-31"); got != "2027-01-01" {
		t.Errorf("year boundary = %s", got)
	}
	if _, err := DayAfter("2026-13-01"); err == nil {
		t.Error("expected an error for an invalid month")
	}
}
