package api

import "testing"

func TestEmptyMatchupKeepsMirrorAndStableTopFits(t *testing.T) {
	result := emptyMatchup(670, 670)
	if result["mirror"] != true || result["attacker_win_rate"] != float64(0) {
		t.Fatalf("result = %#v", result)
	}
	top, ok := result["top_fits"].([]map[string]any)
	if !ok || top == nil || len(top) != 0 {
		t.Fatalf("top_fits = %#v", result["top_fits"])
	}
}

func TestRoundMatchupUsesOneDecimalPlace(t *testing.T) {
	if got := roundMatchup(2.0 / 3.0 * 100); got != 66.7 {
		t.Fatalf("roundMatchup = %v, want 66.7", got)
	}
}
