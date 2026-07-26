package esi

import (
	"net/url"
	"regexp"
	"strings"
)

// Endpoint groups.
//
// ESI rate-limits by endpoint family, and — critically — does not tell you the
// budget for every family. Where it emits `x-ratelimit-*` the learned values are
// authoritative; where it does not, the preset here is the only ground truth,
// hand-tuned against observed 420s. Treating a missing header as "no limit" is
// how you get the whole cluster banned.
//
// See the TypeScript at backend/src/esi/direct/groups.ts, which these presets
// are copied from rather than re-derived.

// Group is one endpoint family's budget and dispatch rules.
type Group struct {
	// Name keys the group's Redis state.
	Name string
	// PathPrefix is the normalised path this group claims.
	PathPrefix string
	// Limit and Window are the preset budget. They seed Redis on first use and
	// remain the fallback whenever headers are absent.
	Limit  int
	Window int // seconds

	// HeaderAuthoritative marks a family where ESI reports its own accounting
	// and learned values should overwrite the preset. False means ignore any
	// headers and trust the preset.
	HeaderAuthoritative bool

	// Sequential forces one request at a time across the whole cluster. Some
	// families return 420 under a concurrent burst even while the token bucket
	// still has capacity, so pacing alone is not enough.
	Sequential bool
}

// Groups is the registry, keyed by name.
var Groups = map[string]Group{
	"characters": {
		Name: "characters", PathPrefix: "characters",
		Limit: 300, Window: 60, HeaderAuthoritative: true,
	},
	"corporations": {
		Name: "corporations", PathPrefix: "corporations",
		Limit: 300, Window: 60, HeaderAuthoritative: true,
	},
	"alliances": {
		Name: "alliances", PathPrefix: "alliances",
		Limit: 300, Window: 60, HeaderAuthoritative: true,
	},
	"killmail": {
		Name: "killmail", PathPrefix: "killmails",
		Limit: 3600, Window: 900, HeaderAuthoritative: true,
	},

	// Below: no rate-limit headers, ever. Conservative and serialised.
	"characters-corporationhistory": {
		Name: "characters-corporationhistory", PathPrefix: "characters-corporationhistory",
		Limit: 60, Window: 60, Sequential: true,
	},
	"corporations-alliancehistory": {
		Name: "corporations-alliancehistory", PathPrefix: "corporations-alliancehistory",
		Limit: 60, Window: 60, Sequential: true,
	},
	"characters-affiliation": {
		Name: "characters-affiliation", PathPrefix: "characters-affiliation",
		Limit: 60, Window: 60, Sequential: true,
	},
	"wars": {
		Name: "wars", PathPrefix: "wars",
		Limit: 60, Window: 60, Sequential: true,
	},
	"fw": {
		Name: "fw", PathPrefix: "fw",
		Limit: 150, Window: 900, Sequential: true,
	},
}

// Probation catches anything unrecognised: slow, serialised, header-distrusting.
//
// An unknown endpoint is one whose limits nobody has measured, so it gets the
// treatment that cannot cause harm. Promoting it means adding a Groups entry
// after watching how it actually behaves under load.
var Probation = Group{
	Name: "probation", PathPrefix: "",
	Limit: 10, Window: 60, Sequential: true,
}

// groupOrder fixes the iteration order for prefix matching. A Go map iterates
// randomly, so without this a URL could land in different groups on different
// runs — the kind of bug that shows up once a week and never reproduces.
var groupOrder = []string{
	"characters-corporationhistory",
	"corporations-alliancehistory",
	"characters-affiliation",
	"characters",
	"corporations",
	"alliances",
	"killmail",
	"wars",
	"fw",
}

var (
	versionSegment = regexp.MustCompile(`^(latest|dev|legacy|v\d+)$`)
	numericSegment = regexp.MustCompile(`^\d+$`)
	hashSegment    = regexp.MustCompile(`^[a-f0-9]{20,}$`)
)

// ResolveGroup classifies a URL or path into a group.
//
// Longer prefixes are tried first, so `characters-corporationhistory` wins over
// `characters` — they have very different budgets and getting that backwards
// would burst an endpoint that 420s on contact.
func ResolveGroup(rawURL string) Group {
	prefix := PathPrefix(rawURL)
	if prefix == "" {
		return Probation
	}
	for _, name := range groupOrder {
		g := Groups[name]
		if prefix == g.PathPrefix || strings.HasPrefix(prefix, g.PathPrefix+"-") {
			return g
		}
	}
	return Probation
}

// PathPrefix normalises an ESI path for classification: version segments,
// numeric IDs and hashes are dropped, and what remains is joined with dashes.
//
//	/latest/characters/12345/                     → characters
//	/latest/characters/12345/corporationhistory/  → characters-corporationhistory
//	/latest/killmails/99/aabbcc.../               → killmails
func PathPrefix(rawURL string) string {
	path := rawURL
	if strings.HasPrefix(rawURL, "http") {
		u, err := url.Parse(rawURL)
		if err != nil {
			return ""
		}
		path = u.Path
	} else if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}

	var segments []string
	for _, p := range strings.Split(path, "/") {
		if p == "" || versionSegment.MatchString(p) || numericSegment.MatchString(p) || hashSegment.MatchString(p) {
			continue
		}
		segments = append(segments, p)
	}
	return strings.Join(segments, "-")
}
