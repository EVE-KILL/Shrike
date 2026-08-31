// Package killtype is the kill-list type registry.
//
// A "type" is one of the named subsets the site slices killmails by — highsec,
// titans, solo, 10b and so on. Two things consume it and they must never
// disagree: the SQL predicates that select a subset for a page, and the
// classifier that bumps the kills_daily_count rollup as each killmail arrives.
// If they drift, the count in the navigation stops matching the list behind it.
//
// The subsets are deliberately not mutually exclusive. A seven-billion ISK solo
// titan loss in nullsec belongs to `latest`, `nullsec`, `solo`, `big`, `5b` and
// `titans` at once, and each of those counts it.
package killtype

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// ISK thresholds. Non-exclusive: a 7b kill is both `big` and `5b`.
const (
	BigISK      = 1_000_000_000
	FiveBISK    = 5_000_000_000
	TenBISK     = 10_000_000_000
	HundredBISK = 100_000_000_000
	TrillionISK = 1_000_000_000_000
)

// Region ranges. Wormhole and abyssal space occupy contiguous blocks of region
// ids, which is why they are ranges rather than lists.
const (
	WSpaceRegionMin  = 11_000_001
	WSpaceRegionMax  = 11_000_033
	AbyssalRegionMin = 12_000_001
	AbyssalRegionMax = 12_000_005
	PochvenRegionID  = 10_000_070

	// NullsecMaxRegionID is the w-space floor: nullsec is security <= 0 in any
	// region below it, which excludes wormholes and abyssal pockets that also
	// have non-positive security.
	NullsecMaxRegionID = 11_000_000
)

// JoveRegionIDs are the unreachable Jove regions.
var JoveRegionIDs = []int32{10_000_004, 10_000_017, 10_000_019}

// SDE-derived constants used by the meta-group subsets.
const (
	CategoryIDShip        = 6
	CategoryIDDrone       = 18
	CategoryIDDeployable  = 22
	CategoryIDStarbase    = 23
	CategoryIDSovereignty = 40
	CategoryIDOrbital     = 46
	CategoryIDCitadel     = 65
	CategoryIDFighter     = 87
	CategoryIDInfantry    = 350001
	MetaT2                = 2
	MetaFaction           = 4
	MetaT3Strategic       = 14
)

// ShipGroups are the victim ship group ids per hull class, taken from the SDE
// and stable across patches.
var ShipGroups = map[string][]int32{
	"frigates":       {25, 324, 830, 831, 834, 893, 1283, 1527},
	"destroyers":     {420, 1305, 1534},
	"cruisers":       {26, 358, 832, 833, 906, 894, 963, 1972},
	"battlecruisers": {419, 1201, 540},
	"battleships":    {27, 898, 900},
	"capitals":       {547, 485, 1538, 883, 659, 30, 4594},
	"freighters":     {513, 902},
	"supercarriers":  {659},
	"titans":         {30},
}

// Types is every kill-list type, in display order.
var Types = []string{
	"latest",
	"highsec", "lowsec", "nullsec", "wspace", "abyssal", "pochven", "jove",
	"timezone-au", "timezone-ru", "timezone-eu", "timezone-us-east", "timezone-us-west",
	"solo", "attackers-1", "attackers-2-4", "attackers-5-9", "attackers-10-24",
	"attackers-25-49", "attackers-50-99", "attackers-100-999", "attackers-1000-plus",
	"pvp", "ganked", "npc",
	"awox", "capital-involved", "supercarrier-involved", "titan-involved", "at-ship-involved",
	"fw-caldari-winner", "fw-gallente-winner", "fw-amarr-winner", "fw-minmatar-winner",
	"fw-caldari-gallente", "fw-amarr-minmatar",
	"big", "5b", "10b",
	"under-1b", "1b-5b", "5b-10b", "10b-100b", "100b-1t", "1t-plus",
	"category-deployable", "category-drone",
	"category-fighter", "category-orbital", "category-starbase", "category-ship",
	"category-sovereignty", "category-structure", "category-infantry",
	"frigates", "destroyers", "cruisers", "battlecruisers", "battleships",
	"capitals", "freighters", "supercarriers", "titans",
	"citadels", "t1", "t2", "t3", "faction",
}

// StableFactTypes are the rollups affected by the attacker-derived stable-fact
// backfill. Operators can pass this list to kills-daily-count without
// rebuilding unrelated labels.
var StableFactTypes = []string{
	"awox", "capital-involved", "supercarrier-involved", "titan-involved", "at-ship-involved",
	"fw-caldari-winner", "fw-gallente-winner", "fw-amarr-winner", "fw-minmatar-winner",
	"fw-caldari-gallente", "fw-amarr-minmatar",
}

// Predicates returns the SQL WHERE fragment for each type, written against the
// `k` alias of the killmails table.
//
// These are fragments rather than parameterised queries because they are
// composed into an INSERT ... SELECT that groups over a date range; every value
// in them is a compile-time constant from this file, never input.
//
// `latest` is "TRUE" on purpose so the same INSERT ... SELECT shape works for
// it without a special case.
func Predicates() map[string]string {
	return map[string]string{
		"latest": "TRUE",

		"highsec": `k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security >= 0.45)`,
		"lowsec":  `k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security > 0.0 AND security < 0.45)`,
		"nullsec": fmt.Sprintf(
			`k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security <= 0.0 AND region_id < %d)`,
			NullsecMaxRegionID),

		"wspace":  fmt.Sprintf(`k.region_id >= %d AND k.region_id <= %d`, WSpaceRegionMin, WSpaceRegionMax),
		"abyssal": fmt.Sprintf(`k.region_id >= %d AND k.region_id <= %d`, AbyssalRegionMin, AbyssalRegionMax),
		"pochven": fmt.Sprintf(`k.region_id = %d`, PochvenRegionID),
		"jove":    fmt.Sprintf(`k.region_id IN (%s)`, csv(JoveRegionIDs)),

		"timezone-au":      utcHourPredicate(8, 14),
		"timezone-ru":      utcHourPredicate(14, 17),
		"timezone-eu":      utcHourPredicate(17, 22),
		"timezone-us-east": `(EXTRACT(HOUR FROM k.killmail_time AT TIME ZONE 'UTC') >= 22 OR EXTRACT(HOUR FROM k.killmail_time AT TIME ZONE 'UTC') < 4)`,
		"timezone-us-west": utcHourPredicate(4, 8),

		"solo":                  `k.is_solo = true`,
		"attackers-1":           `k.attacker_count = 1 AND k.is_solo = false`,
		"attackers-2-4":         `k.attacker_count BETWEEN 2 AND 4`,
		"attackers-5-9":         `k.attacker_count BETWEEN 5 AND 9`,
		"attackers-10-24":       `k.attacker_count BETWEEN 10 AND 24`,
		"attackers-25-49":       `k.attacker_count BETWEEN 25 AND 49`,
		"attackers-50-99":       `k.attacker_count BETWEEN 50 AND 99`,
		"attackers-100-999":     `k.attacker_count BETWEEN 100 AND 999`,
		"attackers-1000-plus":   `k.attacker_count >= 1000`,
		"pvp":                   `k.is_npc = false`,
		"ganked":                `k.is_npc = false AND k.attacker_count >= 10 AND k.solar_system_id IN (SELECT solar_system_id FROM solar_systems WHERE security >= 0.45)`,
		"npc":                   `k.is_npc = true`,
		"awox":                  `k.is_awox = true`,
		"capital-involved":      `k.is_capital_involved = true`,
		"supercarrier-involved": `k.is_super_involved = true`,
		"titan-involved":        `k.is_titan_involved = true`,
		"at-ship-involved":      `k.is_at_ship_involved = true`,
		"fw-caldari-winner":     `k.fw_winner_faction_id = 500001`,
		"fw-minmatar-winner":    `k.fw_winner_faction_id = 500002`,
		"fw-amarr-winner":       `k.fw_winner_faction_id = 500003`,
		"fw-gallente-winner":    `k.fw_winner_faction_id = 500004`,
		"fw-caldari-gallente":   `k.fw_winner_faction_id IN (500001, 500004)`,
		"fw-amarr-minmatar":     `k.fw_winner_faction_id IN (500002, 500003)`,

		"big":      fmt.Sprintf(`k.total_value >= %d`, BigISK),
		"5b":       fmt.Sprintf(`k.total_value >= %d`, FiveBISK),
		"10b":      fmt.Sprintf(`k.total_value >= %d`, TenBISK),
		"under-1b": fmt.Sprintf(`k.total_value < %d`, BigISK),
		"1b-5b":    fmt.Sprintf(`k.total_value >= %d AND k.total_value < %d`, BigISK, FiveBISK),
		"5b-10b":   fmt.Sprintf(`k.total_value >= %d AND k.total_value < %d`, FiveBISK, TenBISK),
		"10b-100b": fmt.Sprintf(`k.total_value >= %d AND k.total_value < %d`, TenBISK, HundredBISK),
		"100b-1t":  fmt.Sprintf(`k.total_value >= %d AND k.total_value < %d`, HundredBISK, TrillionISK),
		"1t-plus":  fmt.Sprintf(`k.total_value >= %d`, TrillionISK),

		"category-deployable":  categoryPredicate(CategoryIDDeployable),
		"category-drone":       categoryPredicate(CategoryIDDrone),
		"category-fighter":     categoryPredicate(CategoryIDFighter),
		"category-orbital":     categoryPredicate(CategoryIDOrbital),
		"category-starbase":    categoryPredicate(CategoryIDStarbase),
		"category-ship":        categoryPredicate(CategoryIDShip),
		"category-sovereignty": categoryPredicate(CategoryIDSovereignty),
		"category-structure":   categoryPredicate(CategoryIDCitadel),
		"category-infantry":    categoryPredicate(CategoryIDInfantry),

		"frigates":       groupPredicate("frigates"),
		"destroyers":     groupPredicate("destroyers"),
		"cruisers":       groupPredicate("cruisers"),
		"battlecruisers": groupPredicate("battlecruisers"),
		"battleships":    groupPredicate("battleships"),
		"capitals":       groupPredicate("capitals"),
		"freighters":     groupPredicate("freighters"),
		"supercarriers":  groupPredicate("supercarriers"),
		"titans":         groupPredicate("titans"),

		// The meta subsets are resolved against the SDE at query time rather
		// than hardcoded, because CCP moves hulls between meta groups.
		"citadels": fmt.Sprintf(
			`k.victim_ship_group_id IN (SELECT group_id FROM inv_groups WHERE category_id = %d)`,
			CategoryIDCitadel),
		// T1 treats a missing meta_group_id as 1, which is what the SDE means
		// by leaving it unset.
		"t1":      `k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE COALESCE(meta_group_id, 1) = 1)`,
		"t2":      metaPredicate(MetaT2),
		"t3":      metaPredicate(MetaT3Strategic),
		"faction": metaPredicate(MetaFaction),
	}
}

func utcHourPredicate(start, end int) string {
	return fmt.Sprintf(`EXTRACT(HOUR FROM k.killmail_time AT TIME ZONE 'UTC') >= %d AND EXTRACT(HOUR FROM k.killmail_time AT TIME ZONE 'UTC') < %d`, start, end)
}

func categoryPredicate(category int32) string {
	return fmt.Sprintf(`k.victim_ship_group_id IN (SELECT group_id FROM inv_groups WHERE category_id = %d)`, category)
}

func groupPredicate(class string) string {
	return fmt.Sprintf(`k.victim_ship_group_id IN (%s)`, csv(ShipGroups[class]))
}

func metaPredicate(meta int) string {
	return fmt.Sprintf(`k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE meta_group_id = %d)`, meta)
}

func csv(ids []int32) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprint(id)
	}
	// Sorted so the generated SQL is stable between runs, which makes a diff of
	// two predicate sets readable.
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// Subject is the part of a killmail that decides which subsets it belongs to.
//
// Assembled by the caller rather than read from the killmail row, because the
// two things it needs beyond the row — the system's security and the victim
// hull's meta group — live in the SDE cache, and the classifier must not reach
// for a database.
type Subject struct {
	KillmailTime time.Time
	Security     float64
	HasSecurity  bool
	RegionID     int32

	IsSolo            bool
	IsNPC             bool
	AttackerCount     int32
	IsAwox            bool
	IsCapitalInvolved bool
	IsSuperInvolved   bool
	IsTitanInvolved   bool
	IsATShipInvolved  bool
	FWWinnerFactionID int32

	TotalValue    float64
	HasTotalValue bool

	VictimShipGroupID int32

	// GroupCategoryID is the SDE category of the victim's hull group. Only
	// CategoryIDCitadel is interesting, but the caller has it for free from the
	// same lookup that produced the group.
	GroupCategoryID int32

	// MetaGroupID is the victim hull's meta group. Zero and one both mean T1 —
	// the SDE leaves it null for ordinary hulls.
	MetaGroupID    int32
	HasVictimShip  bool
	HasVictimGroup bool
}

// Classify returns every subset a killmail belongs to.
//
// This must agree exactly with Predicates: the counts it produces are what the
// navigation shows, and the predicates are what the list behind that number
// selects. A kill counted here but not selected there is a page that says 41
// and shows 40.
func Classify(s Subject) []string {
	types := []string{"latest"}

	if b := SecurityBucket(s.Security, s.HasSecurity, s.RegionID); b != "" {
		types = append(types, b)
	}
	if b := RegionBucket(s.RegionID); b != "" {
		types = append(types, b)
	}
	if !s.KillmailTime.IsZero() {
		types = append(types, TimezoneBucket(s.KillmailTime))
	}

	if s.IsSolo {
		types = append(types, "solo")
	}
	// The raw attacker-row bands can overlap Solo: a solo player kill may also
	// contain NPC participants. The one-attacker label is the only deliberate
	// exception, matching its public "excluding solo" definition.
	if b := AttackerBucket(s.AttackerCount); b != "" && (b != "attackers-1" || !s.IsSolo) {
		types = append(types, b)
	}
	if s.IsNPC {
		types = append(types, "npc")
	} else {
		types = append(types, "pvp")
		if s.AttackerCount >= 10 && s.HasSecurity && s.Security >= 0.45 {
			types = append(types, "ganked")
		}
	}
	if s.IsAwox {
		types = append(types, "awox")
	}
	if s.IsCapitalInvolved {
		types = append(types, "capital-involved")
	}
	if s.IsSuperInvolved {
		types = append(types, "supercarrier-involved")
	}
	if s.IsTitanInvolved {
		types = append(types, "titan-involved")
	}
	if s.IsATShipInvolved {
		types = append(types, "at-ship-involved")
	}
	switch s.FWWinnerFactionID {
	case 500001:
		types = append(types, "fw-caldari-winner", "fw-caldari-gallente")
	case 500002:
		types = append(types, "fw-minmatar-winner", "fw-amarr-minmatar")
	case 500003:
		types = append(types, "fw-amarr-winner", "fw-amarr-minmatar")
	case 500004:
		types = append(types, "fw-gallente-winner", "fw-caldari-gallente")
	}

	if s.HasTotalValue {
		if s.TotalValue >= BigISK {
			types = append(types, "big")
		}
		if s.TotalValue >= FiveBISK {
			types = append(types, "5b")
		}
		if s.TotalValue >= TenBISK {
			types = append(types, "10b")
		}
		switch {
		case s.TotalValue < BigISK:
			types = append(types, "under-1b")
		case s.TotalValue < FiveBISK:
			types = append(types, "1b-5b")
		case s.TotalValue < TenBISK:
			types = append(types, "5b-10b")
		case s.TotalValue < HundredBISK:
			types = append(types, "10b-100b")
		case s.TotalValue < TrillionISK:
			types = append(types, "100b-1t")
		default:
			types = append(types, "1t-plus")
		}
	}

	if s.HasVictimGroup {
		if category := CategoryType(s.GroupCategoryID); category != "" {
			types = append(types, category)
		}
		types = append(types, ShipClass(s.VictimShipGroupID)...)
		if s.GroupCategoryID == CategoryIDCitadel {
			types = append(types, "citadels")
		}
	}

	// Tech tier belongs to the hull type, not its group: a Loki and a Tengu are
	// both strategic cruisers, and the group they share says nothing about it.
	if s.HasVictimShip {
		switch s.MetaGroupID {
		case 0, 1:
			types = append(types, "t1")
		case MetaT2:
			types = append(types, "t2")
		case MetaT3Strategic:
			types = append(types, "t3")
		case MetaFaction:
			types = append(types, "faction")
		}
	}

	return types
}

func TimezoneBucket(t time.Time) string {
	h := t.UTC().Hour()
	switch {
	case h >= 8 && h < 14:
		return "timezone-au"
	case h >= 14 && h < 17:
		return "timezone-ru"
	case h >= 17 && h < 22:
		return "timezone-eu"
	case h >= 22 || h < 4:
		return "timezone-us-east"
	default:
		return "timezone-us-west"
	}
}

func AttackerBucket(n int32) string {
	switch {
	case n == 1:
		return "attackers-1"
	case n >= 2 && n <= 4:
		return "attackers-2-4"
	case n >= 5 && n <= 9:
		return "attackers-5-9"
	case n >= 10 && n <= 24:
		return "attackers-10-24"
	case n >= 25 && n <= 49:
		return "attackers-25-49"
	case n >= 50 && n <= 99:
		return "attackers-50-99"
	case n >= 100 && n <= 999:
		return "attackers-100-999"
	case n >= 1000:
		return "attackers-1000-plus"
	default:
		return ""
	}
}

func CategoryType(id int32) string {
	return map[int32]string{
		CategoryIDDeployable: "category-deployable",
		CategoryIDDrone:      "category-drone", CategoryIDFighter: "category-fighter",
		CategoryIDOrbital: "category-orbital", CategoryIDStarbase: "category-starbase",
		CategoryIDShip: "category-ship", CategoryIDSovereignty: "category-sovereignty",
		CategoryIDCitadel: "category-structure", CategoryIDInfantry: "category-infantry",
	}[id]
}

// SecurityBucket classifies a kill by the security of the system it happened
// in, mirroring the highsec/lowsec/nullsec predicates exactly.
//
// Returns "" for kills that belong to none of them — wormhole, abyssal, Pochven
// and Jove space are region-based and handled by RegionBucket instead.
func SecurityBucket(security float64, hasSecurity bool, regionID int32) string {
	if !hasSecurity {
		return ""
	}
	switch {
	case security >= 0.45:
		return "highsec"
	case security > 0.0:
		return "lowsec"
	case regionID != 0 && regionID < NullsecMaxRegionID:
		return "nullsec"
	}
	return ""
}

// RegionBucket classifies a kill by the kind of space its region is.
//
// Checked before the wormhole range because Pochven and the Jove regions sit
// outside it but are likewise not ordinary nullsec.
func RegionBucket(regionID int32) string {
	if regionID == 0 {
		return ""
	}
	if regionID == PochvenRegionID {
		return "pochven"
	}
	if slices.Contains(JoveRegionIDs, regionID) {
		return "jove"
	}
	if regionID >= AbyssalRegionMin && regionID <= AbyssalRegionMax {
		return "abyssal"
	}
	if regionID >= WSpaceRegionMin && regionID <= WSpaceRegionMax {
		return "wspace"
	}
	return ""
}

// ShipClass returns the hull class subset a victim ship group belongs to, or ""
// when it belongs to none.
//
// A group can appear in more than one class — 30 (titans) and 659
// (supercarriers) are both also capitals — so this reports the most specific
// one and Classify records the rest.
func ShipClass(groupID int32) []string {
	var out []string
	for _, class := range []string{
		"frigates", "destroyers", "cruisers", "battlecruisers", "battleships",
		"freighters", "supercarriers", "titans", "capitals",
	} {
		if slices.Contains(ShipGroups[class], groupID) {
			out = append(out, class)
		}
	}
	return out
}
