package campaign

import "testing"

// The funding reason is the only thing standing between a donation and the
// wrong prize pool. It is typed by hand into an in-game wallet transfer, cannot
// be corrected afterwards, and is matched case-insensitively — so the tests
// here are mostly about what must *not* parse.

func TestValidReasonParses(t *testing.T) {
	id, ok := ParseFundingReason("campaign:AbCdEf12345678")
	if !ok {
		t.Fatal("a well-formed reason did not parse")
	}
	if id != "AbCdEf12345678" {
		t.Errorf("parsed id %q, want AbCdEf12345678 — the id's case must survive "+
			"even though the prefix match does not", id)
	}
}

func TestPrefixIsCaseInsensitive(t *testing.T) {
	for _, reason := range []string{
		"campaign:abcdef12345678",
		"CAMPAIGN:abcdef12345678",
		"Campaign:abcdef12345678",
	} {
		if _, ok := ParseFundingReason(reason); !ok {
			t.Errorf("%q did not parse; the prefix is matched case-insensitively", reason)
		}
	}
}

func TestSurroundingWhitespaceIsTolerated(t *testing.T) {
	if _, ok := ParseFundingReason("  campaign:abcdef12345678\n"); !ok {
		t.Error("a reason with surrounding whitespace did not parse — the client " +
			"trims, and a donation should not be lost to a trailing newline")
	}
}

// The length is exact on purpose. A shorter id that happened to prefix a real
// one would otherwise fund a stranger's campaign.
func TestWrongLengthIsRejected(t *testing.T) {
	for _, reason := range []string{
		"campaign:abcdef1234567",   // 13
		"campaign:abcdef123456789", // 15
		"campaign:",
		"campaign:abc",
	} {
		if id, ok := ParseFundingReason(reason); ok {
			t.Errorf("%q parsed as %q; only exactly 14 characters is a campaign id", reason, id)
		}
	}
}

// Anything that is not alphanumeric is not an id, and letting punctuation
// through would open the door to a reason that means something to Postgres.
func TestNonAlphanumericIsRejected(t *testing.T) {
	for _, reason := range []string{
		"campaign:abcdef1234-678",
		"campaign:abcdef 234 678",
		"campaign:abcdef%2345678",
		"campaign:'; DROP TABLE--",
	} {
		if _, ok := ParseFundingReason(reason); ok {
			t.Errorf("%q parsed as a campaign id", reason)
		}
	}
}

// Trailing content after a valid id must not parse — the anchor is what stops
// "campaign:<valid>extra" being read as a donation to <valid>.
func TestTrailingContentIsRejected(t *testing.T) {
	if _, ok := ParseFundingReason("campaign:abcdef12345678 thanks!"); ok {
		t.Error("a reason with trailing content parsed; the pattern must be anchored")
	}
}

func TestUnrelatedReasonIsRejected(t *testing.T) {
	for _, reason := range []string{
		"",
		"thanks for the killboard",
		"campaigns:abcdef12345678",
		"donation",
	} {
		if _, ok := ParseFundingReason(reason); ok {
			t.Errorf("%q parsed as a campaign funding reason", reason)
		}
	}
}
