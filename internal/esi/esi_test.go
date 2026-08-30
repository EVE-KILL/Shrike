package esi

import (
	"net/http"
	"testing"
	"time"
)

// Group classification decides which budget a request draws from. Getting it
// wrong is not a cosmetic error: `characters` allows 300 per minute while
// `characters-corporationhistory` allows 60 and must be serialised, so a
// misclassified history request bursts an endpoint that 420s on contact.
func TestResolveGroup(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/latest/status/?datasource=tranquility", "status"},
		{"/latest/characters/12345/", "characters"},
		{"/latest/characters/12345/corporationhistory/", "characters-corporationhistory"},
		{"/latest/corporations/98187159/", "corporations"},
		{"/latest/corporations/98187159/alliancehistory/", "corporations-alliancehistory"},
		{"/latest/characters/affiliation/", "characters-affiliation"},
		{"/latest/alliances/", "alliances"},
		{"/latest/alliances/99015054/corporations/", "alliances"},
		{"/latest/killmails/137258027/1d9365aaed385213867e40390d29cd4c7596e0e3/", "killmail"},
		{"/latest/wars/748895/", "wars"},
		{"/latest/wars/748895/killmails/", "wars"},
		{"/latest/fw/systems/", "fw"},
		// Unmeasured endpoints must land on probation, not on a neighbour's
		// generous budget.
		{"/latest/universe/system_kills/", "probation"},
		{"/latest/markets/prices/", "probation"},
		{"", "probation"},
	}
	for _, c := range cases {
		if got := ResolveGroup(c.path).Name; got != c.want {
			t.Errorf("%s → %s, want %s", c.path, got, c.want)
		}
	}

	// A full URL must classify the same as its path.
	if got := ResolveGroup("https://esi.evetech.net/latest/characters/1/").Name; got != "characters" {
		t.Errorf("absolute URL → %s", got)
	}
	// Query strings are not part of the path.
	if got := ResolveGroup("/latest/corporations/98187159/?compatibility_date=2026-07-21").Name; got != "corporations" {
		t.Errorf("path with query → %s", got)
	}
}

// The longer prefix has to win, and Go's random map iteration makes that a real
// risk — this is the test that catches a regression to unordered matching.
func TestResolveGroupPrefersLongerPrefix(t *testing.T) {
	for i := range 50 {
		if got := ResolveGroup("/latest/characters/1/corporationhistory/").Name; got != "characters-corporationhistory" {
			t.Fatalf("iteration %d resolved to %s", i, got)
		}
	}
}

func TestPathPrefix(t *testing.T) {
	cases := map[string]string{
		"/latest/characters/12345/":                        "characters",
		"/v5/characters/12345/corporationhistory/":         "characters-corporationhistory",
		"/dev/alliances/":                                  "alliances",
		"/legacy/killmails/99/aabbccddeeff00112233445566/": "killmails",
	}
	for in, want := range cases {
		if got := PathPrefix(in); got != want {
			t.Errorf("PathPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every group's presets have to be usable. A zero limit would deadlock the
// bucket; a zero window would make it refill on every call.
func TestGroupPresetsAreSane(t *testing.T) {
	all := make([]Group, 0, len(Groups)+1)
	for name, g := range Groups {
		if g.Name != name {
			t.Errorf("group %q is keyed under a different name (%q)", g.Name, name)
		}
		all = append(all, g)
	}
	all = append(all, Probation)

	for _, g := range all {
		if g.Limit < TokenCost {
			t.Errorf("group %s cannot afford a single request (limit %d, cost %d)", g.Name, g.Limit, TokenCost)
		}
		if g.Window <= 0 {
			t.Errorf("group %s has a non-positive window", g.Name)
		}
		// The families with no headers are exactly the ones that must be
		// serialised; the reverse would burst an endpoint with no feedback.
		if !g.HeaderAuthoritative && !g.Sequential {
			t.Errorf("group %s trusts no headers but allows concurrency", g.Name)
		}
	}

	// Probation must be the most conservative thing in the registry.
	for _, g := range Groups {
		if perSecond(Probation) > perSecond(g) {
			t.Errorf("probation is faster than %s", g.Name)
		}
	}
}

func perSecond(g Group) float64 { return float64(g.Limit) / float64(g.Window) }

// A group whose headers are not authoritative must ignore them entirely —
// otherwise a hand-tuned preset gets overwritten by whatever noise a proxy
// happened to set.
func TestApplyHeadersIgnoresNonAuthoritativeGroups(t *testing.T) {
	// Both calls must no-op before touching Redis, so a nil client is safe and
	// is itself the assertion.
	r := &RateLimiter{redis: nil}
	g := Groups["wars"]
	if err := r.ApplyHeaders(t.Context(), g, 1, time.Second); err != nil {
		t.Errorf("ApplyHeaders on a preset-only group: %v", err)
	}
	if err := r.ApplyRemaining(t.Context(), g, 1); err != nil {
		t.Errorf("ApplyRemaining on a preset-only group: %v", err)
	}
}

func TestWaitCap(t *testing.T) {
	// A sequential group's bucket legitimately takes most of a window to
	// refill; capping it at the normal 10s would fail every trailing request
	// in a burst.
	if waitCap(Groups["wars"]) <= waitCap(Groups["characters"]) {
		t.Error("sequential groups need the longer cap")
	}
	if waitCap(Groups["wars"]) < time.Duration(Groups["wars"].Window)*time.Second {
		t.Error("sequential cap is shorter than the window it has to outlast")
	}
}

func TestParseExpires(t *testing.T) {
	if _, ok := ParseExpires(""); ok {
		t.Error("empty header should not parse")
	}
	if _, ok := ParseExpires("not a date"); ok {
		t.Error("garbage should not parse")
	}
	ms, ok := ParseExpires("Sat, 25 Jul 2026 20:24:59 GMT")
	if !ok {
		t.Fatal("a valid HTTP date should parse")
	}
	if got := time.UnixMilli(ms).UTC().Format(time.RFC3339); got != "2026-07-25T20:24:59Z" {
		t.Errorf("parsed to %s", got)
	}
}

// Freshness is what decides whether ESI is contacted at all, so the boundary
// matters more than it looks.
func TestEntryFresh(t *testing.T) {
	now := time.Now()
	var absent *Entry
	if absent.Fresh(now) {
		t.Error("a missing entry is never fresh")
	}
	if (&Entry{Expires: now.Add(-time.Second).UnixMilli()}).Fresh(now) {
		t.Error("an expired entry is not fresh")
	}
	if !(&Entry{Expires: now.Add(time.Minute).UnixMilli()}).Fresh(now) {
		t.Error("an unexpired entry is fresh")
	}
}

func TestResponsePermanent(t *testing.T) {
	for _, status := range []int{400, 404, 410, 422} {
		if !(Response[int]{Status: status}).Permanent() {
			t.Errorf("HTTP %d should be permanent", status)
		}
	}
	for _, status := range []int{0, 200, 420, 429, 500, 502, 503} {
		if (Response[int]{Status: status}).Permanent() {
			t.Errorf("HTTP %d should be retryable", status)
		}
	}
}

func TestKillmailPath(t *testing.T) {
	got := KillmailPath(137258027, "1d9365aaed385213867e40390d29cd4c7596e0e3")
	want := "/latest/killmails/137258027/1d9365aaed385213867e40390d29cd4c7596e0e3/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// The trailing slash is load-bearing: ESI redirects without it, and a
	// redirect on every killmail doubles the request count.
	if got[len(got)-1] != '/' {
		t.Error("the trailing slash was dropped")
	}
}

// The 2026-07-21 corporation schema renamed and rescaled fields. Reading the
// wrong one shows a 10% corporation as taxing 1000%.
func TestCorporationSchemaStraddle(t *testing.T) {
	old := Corporation{TaxRate: new(0.1), FactionID: 500001}
	if got := old.TaxRateFraction(); got != 0.1 {
		t.Errorf("legacy tax rate = %v, want 0.1", got)
	}
	if old.LoyaltyPointTaxFraction() != nil {
		t.Error("the legacy shape has no loyalty-point tax; nil is not zero")
	}
	if got := old.Faction(); got != 500001 {
		t.Errorf("legacy faction = %d", got)
	}

	current := Corporation{EnlistedFactionID: 500002,
		TaxRates: &struct {
			ISK          float64 `json:"isk"`
			LoyaltyPoint float64 `json:"loyalty_point"`
		}{ISK: 10, LoyaltyPoint: 25}}

	if got := current.TaxRateFraction(); got != 0.1 {
		t.Errorf("current tax rate = %v, want 0.1 (10%% expressed as a fraction)", got)
	}
	lp := current.LoyaltyPointTaxFraction()
	if lp == nil || *lp != 0.25 {
		t.Errorf("loyalty-point tax = %v, want 0.25", lp)
	}
	if got := current.Faction(); got != 500002 {
		t.Errorf("current faction = %d", got)
	}

	// A corporation in neither shape taxes nothing rather than panicking.
	if got := (Corporation{}).TaxRateFraction(); got != 0 {
		t.Errorf("empty corporation tax = %v", got)
	}
}

func TestFullURL(t *testing.T) {
	c := &Client{BaseURL: "https://example.test"}
	if got := c.fullURL("/latest/x/"); got != "https://example.test/latest/x/" {
		t.Errorf("relative path → %s", got)
	}
	// An absolute URL is passed through, which is what lets a caller point at a
	// test server without the client rewriting it.
	if got := c.fullURL("http://other.test/y"); got != "http://other.test/y" {
		t.Errorf("absolute URL → %s", got)
	}
	empty := &Client{}
	if got := empty.fullURL("/z"); got != DefaultBaseURL+"/z" {
		t.Errorf("default base → %s", got)
	}
}

// Retrying a 4xx spends error budget to be told the same thing again, and the
// error budget is global — burning it on a known-bad id throttles everything.
func TestRetryClassification(t *testing.T) {
	retryable := map[int]bool{500: true, 502: true, 503: true, 504: true}
	for status := range retryable {
		if status < 500 || status >= 600 {
			t.Errorf("%d is not a server error", status)
		}
	}
	for _, status := range []int{http.StatusNotFound, http.StatusBadRequest, http.StatusUnprocessableEntity} {
		if retryable[status] {
			t.Errorf("%d must not be retried", status)
		}
	}
}

// The tests run with a compressed retry ladder, so the real one is asserted
// here instead. It is deliberately not exponential past five seconds: ESI
// recovers in seconds or in minutes, and doubling into minutes only holds a
// worker hostage.
func TestProductionBackoffLadder(t *testing.T) {
	c := &Client{}
	var total time.Duration
	prev := time.Duration(0)
	for attempt := 1; attempt <= maxHTTPRetries; attempt++ {
		wait := c.retryBackoff(attempt)
		if wait < prev {
			t.Errorf("attempt %d waits %v, less than the previous %v", attempt, wait, prev)
		}
		if wait > 10*time.Second {
			t.Errorf("attempt %d waits %v — long enough to hold a worker hostage", attempt, wait)
		}
		prev = wait
		total += wait
	}
	// The whole ladder has to fit inside the singleflight claim, or a retrying
	// request loses its claim and a second worker starts the same fetch.
	if total >= claimTTL {
		t.Errorf("the retry ladder totals %v, at or beyond the %v claim TTL", total, claimTTL)
	}
}
