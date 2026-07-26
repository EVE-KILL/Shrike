package campaign

import "testing"

// Payout arithmetic. These numbers are real ISK, so the property that matters
// most is not that each share is right in isolation but that they add up to
// exactly what was funded — a pool that does not balance is a support ticket
// however defensible each individual share is.

func sum(xs []int64) int64 {
	var t int64
	for _, x := range xs {
		t += x
	}
	return t
}

func TestPayoutsSumToTheFundedTotal(t *testing.T) {
	// Deliberately awkward: three ranks that do not divide evenly.
	cases := []struct {
		total       float64
		percentages []float64
	}{
		{1_000_000_000, []float64{50, 30, 20}},
		{1, []float64{50, 30, 20}},
		{7, []float64{33, 33, 34}},
		{999_999_999, []float64{60, 25, 15}},
		{1_000_000_007, []float64{1, 1, 1}},
	}

	for _, c := range cases {
		payouts := CalculatePayouts(c.total, c.percentages)
		if got, want := sum(payouts), int64(c.total); got != want {
			t.Errorf("payouts for %v of %.0f sum to %d, want %d — the pool does not balance",
				c.percentages, c.total, got, want)
		}
	}
}

// The rounding remainder goes to first place, so first is never shortchanged
// by the flooring the other ranks get.
func TestRemainderGoesToFirst(t *testing.T) {
	// 10 ISK over three equal shares: 3, 3, and 4 left for first.
	payouts := CalculatePayouts(10, []float64{1, 1, 1})
	if payouts[0] != 4 {
		t.Errorf("first place got %d of 10 across three equal shares, want 4 "+
			"(3 each to the others, remainder to first)", payouts[0])
	}
	if payouts[1] != 3 || payouts[2] != 3 {
		t.Errorf("lower ranks got %d and %d, want 3 each", payouts[1], payouts[2])
	}
}

// An empty pool pays nothing rather than paying a negative first place.
func TestEmptyPoolPaysNothing(t *testing.T) {
	payouts := CalculatePayouts(0, []float64{50, 30, 20})
	if len(payouts) != 3 {
		t.Fatalf("got %d payouts for 3 ranks", len(payouts))
	}
	for i, p := range payouts {
		if p != 0 {
			t.Errorf("rank %d was paid %d from an empty pool", i+1, p)
		}
	}
}

// A negative funded total is not a pool anyone can be paid from.
func TestNegativeTotalPaysNothing(t *testing.T) {
	for _, p := range CalculatePayouts(-500, []float64{50, 50}) {
		if p != 0 {
			t.Errorf("a negative pool paid out %d", p)
		}
	}
}

// Every rank weighted zero has no basis for division. Paying the whole pool to
// first — which is what the remainder rule would do unguarded — would be a
// windfall nobody configured.
func TestZeroWeightsPayNothing(t *testing.T) {
	payouts := CalculatePayouts(1_000_000, []float64{0, 0, 0})
	if s := sum(payouts); s != 0 {
		t.Errorf("zero-weighted ranks were paid %d in total; with no weights "+
			"there is no basis to divide by", s)
	}
}

// Negative weights are clamped, not honoured as a claw-back.
func TestNegativeWeightsAreClamped(t *testing.T) {
	payouts := CalculatePayouts(1000, []float64{50, -50, 50})
	if payouts[1] != 0 {
		t.Errorf("a negatively-weighted rank was paid %d, want 0", payouts[1])
	}
	if s := sum(payouts); s != 1000 {
		t.Errorf("payouts sum to %d, want 1000", s)
	}
}

// No ranks configured means nothing to pay, and specifically not a panic.
func TestNoRanks(t *testing.T) {
	if got := CalculatePayouts(1_000_000, nil); len(got) != 0 {
		t.Errorf("got %d payouts for no ranks", len(got))
	}
}

// One rank takes the whole pool.
func TestSingleRankTakesEverything(t *testing.T) {
	payouts := CalculatePayouts(1_234_567, []float64{100})
	if payouts[0] != 1_234_567 {
		t.Errorf("the only rank was paid %d of 1,234,567", payouts[0])
	}
}

// Fractional ISK is floored away before anything is divided — the game has no
// fractions and a payout instruction with one cannot be executed.
func TestFractionalTotalIsFloored(t *testing.T) {
	payouts := CalculatePayouts(100.9, []float64{50, 50})
	if s := sum(payouts); s != 100 {
		t.Errorf("payouts sum to %d for a pool of 100.9, want 100", s)
	}
}
