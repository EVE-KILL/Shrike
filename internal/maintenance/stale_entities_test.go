package maintenance

import "testing"

func TestStaleEntityOptionsNormalize(t *testing.T) {
	opts := StaleEntityOptions{}
	opts.normalize()
	if opts.AllianceDays != 30 || opts.CorporationDays != 30 || opts.Batch != 500 {
		t.Fatalf("normalized options = %#v", opts)
	}

	opts = StaleEntityOptions{AllianceDays: 7, CorporationDays: 9, Batch: 42}
	opts.normalize()
	if opts.AllianceDays != 7 || opts.CorporationDays != 9 || opts.Batch != 42 {
		t.Fatalf("explicit options changed: %#v", opts)
	}
}
