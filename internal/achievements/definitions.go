// Package achievements awards the per-character badges the profile pages show.
//
// The definitions are generated rather than listed, because most of them are
// the same handful of shapes at five escalating thresholds — "destroy 10
// frigates", "destroy 50 frigates", and so on. Generating them keeps the
// thresholds and their point values in one table instead of ninety
// near-identical literals.
//
// Ids are a stable contract: they are the primary key of entity_achievements
// and appear in URLs, so a generated id must never change for an existing
// achievement. That is why the id format is fixed and the categories are keyed
// by a short slug rather than by their display name.
package achievements

import "fmt"

// Rarity is the badge tier, escalating with the threshold.
type Rarity string

const (
	Common    Rarity = "common"
	Uncommon  Rarity = "uncommon"
	Rare      Rarity = "rare"
	Epic      Rarity = "epic"
	Legendary Rarity = "legendary"
)

var rarities = []Rarity{Common, Uncommon, Rare, Epic, Legendary}

// Trigger is what a definition counts.
type Trigger string

const (
	TriggerFinalBlows      Trigger = "final_blows"
	TriggerSoloKills       Trigger = "solo_kills"
	TriggerKillsByValue    Trigger = "kills_by_value"
	TriggerKillsBySecurity Trigger = "kills_by_security"
	TriggerShipKills       Trigger = "ship_kills"
	TriggerShipLosses      Trigger = "ship_losses"
)

// Definition is one achievement.
type Definition struct {
	ID          string
	Name        string
	Description string
	Category    string
	Threshold   int32
	BasePoints  int32
	Rarity      Rarity

	// Negative marks an achievement that subtracts points. Losing ships is
	// tracked and celebrated, but it does not make you look better.
	Negative bool

	Trigger Trigger

	// The trigger's parameters. Only the ones its trigger uses are set.
	GroupIDs []int32
	MinValue float64
	MinSec   float64
	MaxSec   float64
}

// SignedBasePoints is the point value with its sign applied.
func (d Definition) SignedBasePoints() int32 {
	if d.Negative {
		return -d.BasePoints
	}
	return d.BasePoints
}

// MatchesGroup reports whether a ship group is one this definition counts.
func (d Definition) MatchesGroup(groupID int32) bool {
	for _, g := range d.GroupIDs {
		if g == groupID {
			return true
		}
	}
	return false
}

// shipCategory is one hull class with its escalating thresholds.
type shipCategory struct {
	key            string
	name           string
	groupIDs       []int32
	basePoints     int32
	killThresholds []int32
	lossThresholds []int32
}

// The ship groups here are the achievement system's own list and deliberately
// differ from both internal/killtype and internal/relay — the three were built
// at different times for different purposes and production has been awarding
// against this one. Changing it would silently alter who has which badge.
var shipCategories = []shipCategory{
	{"frigates", "Frigates", []int32{324, 893, 25, 831, 237}, 25,
		[]int32{10, 50, 100, 500, 1000}, []int32{10, 50}},
	{"destroyers", "Destroyers", []int32{420, 541}, 35,
		[]int32{10, 25, 50, 250, 500}, []int32{10, 25}},
	{"cruisers", "Cruisers", []int32{906, 26, 833, 358, 894, 832, 963}, 50,
		[]int32{10, 25, 50, 200, 400}, []int32{10, 25}},
	{"battleships", "Battleships", []int32{419, 540}, 75,
		[]int32{5, 10, 25, 100, 200}, []int32{5, 10}},
	{"capitals", "Capital Ships", []int32{547, 485, 659, 30}, 500,
		[]int32{1, 5, 10, 25, 50}, []int32{1, 5}},
	{"industrials", "Industrial Ships", []int32{27, 898, 900, 513, 902}, 20,
		[]int32{10, 50, 100, 500, 1000}, []int32{10, 50}},
}

// All is every achievement, generated once.
var All = generate()

// ByTrigger groups them for matching, so a killmail does not rescan the whole
// list per attacker.
var ByTrigger = groupByTrigger(All)

func generate() []Definition {
	out := []Definition{
		{ID: "babys_first_kill", Name: "Baby's First Kill", Description: "Get your first final blow",
			Category: "Combat", Threshold: 1, BasePoints: 5, Rarity: Common, Trigger: TriggerFinalBlows},
		{ID: "rookie_killer", Name: "Rookie Killer", Description: "Get 10 final blows",
			Category: "Combat", Threshold: 10, BasePoints: 15, Rarity: Common, Trigger: TriggerFinalBlows},
		{ID: "experienced_killer", Name: "Experienced Killer", Description: "Get 100 final blows",
			Category: "Combat", Threshold: 100, BasePoints: 50, Rarity: Uncommon, Trigger: TriggerFinalBlows},
		{ID: "veteran_killer", Name: "Veteran Killer", Description: "Get 1,000 final blows",
			Category: "Combat", Threshold: 1000, BasePoints: 200, Rarity: Rare, Trigger: TriggerFinalBlows},
		{ID: "solo_first_blood", Name: "Solo First Blood", Description: "Get your first solo kill",
			Category: "Combat", Threshold: 1, BasePoints: 10, Rarity: Common, Trigger: TriggerSoloKills},

		// The security bands overlap the boundaries deliberately: 1.1 as the
		// upper bound catches 1.0 systems, and -10 as the lower catches the
		// most negative nullsec.
		{ID: "highsec_hunter", Name: "Highsec Hunter", Description: "Get a kill in highsec",
			Category: "Locations", Threshold: 1, BasePoints: 15, Rarity: Common,
			Trigger: TriggerKillsBySecurity, MinSec: 0.5, MaxSec: 1.1},
		{ID: "lowsec_prowler", Name: "Lowsec Prowler", Description: "Get a kill in lowsec",
			Category: "Locations", Threshold: 1, BasePoints: 25, Rarity: Common,
			Trigger: TriggerKillsBySecurity, MinSec: 0.0, MaxSec: 0.5},
		{ID: "nullsec_warrior", Name: "Nullsec Warrior", Description: "Get a kill in nullsec",
			Category: "Locations", Threshold: 1, BasePoints: 30, Rarity: Common,
			Trigger: TriggerKillsBySecurity, MinSec: -10, MaxSec: 0.0},

		{ID: "bling_hunter", Name: "Bling Hunter", Description: "Get a kill worth 1B+ ISK",
			Category: "High Value", Threshold: 1, BasePoints: 100, Rarity: Uncommon,
			Trigger: TriggerKillsByValue, MinValue: 1_000_000_000},
	}

	for _, cat := range shipCategories {
		for i, threshold := range cat.killThresholds {
			out = append(out, Definition{
				ID:          fmt.Sprintf("%s_killer_%d", cat.key, threshold),
				Name:        fmt.Sprintf("%s Killer (%d)", cat.name, threshold),
				Description: fmt.Sprintf("Destroy %d %s", threshold, lower(cat.name)),
				Category:    "Ship Kills",
				Threshold:   threshold,
				// Points escalate with the tier, so the fifth threshold is
				// worth five times the first.
				BasePoints: cat.basePoints * int32(i+1),
				Rarity:     rarityAt(i),
				Trigger:    TriggerShipKills,
				GroupIDs:   cat.groupIDs,
			})
		}

		for i, threshold := range cat.lossThresholds {
			out = append(out, Definition{
				ID:          fmt.Sprintf("%s_loser_%d", cat.key, threshold),
				Name:        fmt.Sprintf("%s Lost (%d)", cat.name, threshold),
				Description: fmt.Sprintf("Lose %d %s", threshold, lower(cat.name)),
				Category:    "Ship Losses",
				Threshold:   threshold,
				// Losses are worth roughly a third of the equivalent kill, and
				// subtract rather than add.
				BasePoints: (cat.basePoints * 3 / 10) * int32(i+1),
				Rarity:     rarityAt(i),
				Negative:   true,
				Trigger:    TriggerShipLosses,
				GroupIDs:   cat.groupIDs,
			})
		}
	}

	return out
}

func groupByTrigger(defs []Definition) map[Trigger][]Definition {
	out := map[Trigger][]Definition{}
	for _, d := range defs {
		out[d.Trigger] = append(out[d.Trigger], d)
	}
	return out
}

// rarityAt escalates with the tier, saturating at legendary for anything past
// the fifth.
func rarityAt(i int) Rarity {
	if i < len(rarities) {
		return rarities[i]
	}
	return Legendary
}

func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + 32
		}
	}
	return string(out)
}
