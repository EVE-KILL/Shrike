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
	"sort"
	"strings"
)

// ISK thresholds. Non-exclusive: a 7b kill is both `big` and `5b`.
const (
	BigISK   = 1_000_000_000
	FiveBISK = 5_000_000_000
	TenBISK  = 10_000_000_000
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
	CategoryIDCitadel = 65
	MetaT2            = 2
	MetaFaction       = 4
	MetaT3Strategic   = 14
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
	"solo", "npc",
	"big", "5b", "10b",
	"frigates", "destroyers", "cruisers", "battlecruisers", "battleships",
	"capitals", "freighters", "supercarriers", "titans",
	"citadels", "t1", "t2", "t3", "faction",
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

		"solo": `k.is_solo = true`,
		"npc":  `k.is_npc = true`,

		"big": fmt.Sprintf(`k.total_value >= %d`, BigISK),
		"5b":  fmt.Sprintf(`k.total_value >= %d`, FiveBISK),
		"10b": fmt.Sprintf(`k.total_value >= %d`, TenBISK),

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
	Security    float64
	HasSecurity bool
	RegionID    int32

	IsSolo bool
	IsNPC  bool

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

	if s.IsSolo {
		types = append(types, "solo")
	}
	if s.IsNPC {
		types = append(types, "npc")
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
	}

	if s.HasVictimGroup {
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
	for _, j := range JoveRegionIDs {
		if regionID == j {
			return "jove"
		}
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
		for _, g := range ShipGroups[class] {
			if g == groupID {
				out = append(out, class)
				break
			}
		}
	}
	return out
}
