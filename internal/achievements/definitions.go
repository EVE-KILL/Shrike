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

import (
	"fmt"
	"slices"
)

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
	TriggerKills           Trigger = "kills"
	TriggerLosses          Trigger = "losses"
	TriggerSoloKills       Trigger = "solo_kills"
	TriggerKillsByValue    Trigger = "kills_by_value"
	TriggerKillsBySecurity Trigger = "kills_by_security"
	TriggerShipKills       Trigger = "ship_kills"
	TriggerShipLosses      Trigger = "ship_losses"
	TriggerKillsByRegion   Trigger = "kills_by_region"
	TriggerLossesByRegion  Trigger = "losses_by_region"
	TriggerKillsBySystem   Trigger = "kills_by_system"
	TriggerLossesBySystem  Trigger = "losses_by_system"
	TriggerConcorded       Trigger = "concorded"
	TriggerKilledByCorp    Trigger = "killed_by_corporation"
	TriggerKilledCorp      Trigger = "killed_corporation"
	TriggerTournament      Trigger = "tournament_participation"
	TriggerAwox            Trigger = "awox"
	TriggerAwoxed          Trigger = "awoxed"
	TriggerGank            Trigger = "gank"
)

// Definition is one achievement.
type Definition struct {
	ID          string
	Name        string
	Description string
	Category    string
	Threshold   int32
	// Thresholds enables one trophy to have several capped levels. Threshold is
	// retained for the original single-level achievements and remains the value
	// stored in entity_achievements for compatibility.
	Thresholds []int32
	BasePoints int32
	Rarity     Rarity

	// Negative marks an achievement that subtracts points. Losing ships is
	// tracked and celebrated, but it does not make you look better.
	Negative bool

	Trigger Trigger

	// The trigger's parameters. Only the ones its trigger uses are set.
	GroupIDs      []int32
	MinValue      float64
	MinSec        float64
	MaxSec        float64
	RegionID      int32
	SystemID      int32
	CorporationID int32
}

// Levels returns the thresholds for this definition in ascending order.
func (d Definition) Levels() []int32 {
	if len(d.Thresholds) > 0 {
		return d.Thresholds
	}
	return []int32{d.Threshold}
}

// LevelFor returns the capped level reached at count.
func (d Definition) LevelFor(count int32) int32 {
	var level int32
	for _, threshold := range d.Levels() {
		if count < threshold {
			break
		}
		level++
	}
	return level
}

// PointsFor returns the capped score at count. Each successive level is worth
// one additional base unit, making a five-level trophy worth 1+2+3+4+5 units.
func (d Definition) PointsFor(count int32) int32 {
	level := d.LevelFor(count)
	points := d.BasePoints * level * (level + 1) / 2
	if d.Negative {
		return -points
	}
	return points
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
	return slices.Contains(d.GroupIDs, groupID)
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

type trophyShipGroup struct {
	id   int32
	name string
}

// Published ship groups used by zKillboard's trophy system. These are kept
// explicit so an SDE rename does not silently rename a stable achievement.
var trophyShipGroups = []trophyShipGroup{
	{324, "Assault Frigate"}, {1201, "Attack Battlecruiser"}, {27, "Battleship"},
	{898, "Black Ops"}, {1202, "Blockade Runner"}, {883, "Capital Industrial Ship"},
	{29, "Capsule"}, {547, "Carrier"}, {419, "Combat Battlecruiser"},
	{906, "Combat Recon Ship"}, {5120, "Command Carrier"}, {1534, "Command Destroyer"},
	{540, "Command Ship"}, {237, "Corvette"}, {830, "Covert Ops"}, {26, "Cruiser"},
	{380, "Deep Space Transport"}, {420, "Destroyer"}, {485, "Dreadnought"},
	{893, "Electronic Attack Ship"}, {543, "Exhumer"}, {4902, "Expedition Command Ship"},
	{1283, "Expedition Frigate"}, {1972, "Flag Cruiser"}, {1538, "Force Auxiliary"},
	{833, "Force Recon Ship"}, {513, "Freighter"}, {25, "Frigate"}, {28, "Hauler"},
	{358, "Heavy Assault Cruiser"}, {894, "Heavy Interdiction Cruiser"},
	{941, "Industrial Command Ship"}, {831, "Interceptor"}, {541, "Interdictor"},
	{902, "Jump Freighter"}, {4594, "Lancer Dreadnought"}, {832, "Logistics"},
	{1527, "Logistics Frigate"}, {900, "Marauder"}, {463, "Mining Barge"},
	{1022, "Prototype Exploration Ship"}, {31, "Shuttle"}, {5087, "Special Edition Yachts"},
	{834, "Stealth Bomber"}, {963, "Strategic Cruiser"}, {659, "Supercarrier"},
	{1305, "Tactical Destroyer"}, {30, "Titan"},
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

		{ID: "kill_kill_kill", Name: "Kill Kill Kill", Description: "Take part in capsuleer ship kills",
			Category: "Combat", Threshold: 625, Thresholds: []int32{1, 5, 25, 125, 625}, BasePoints: 10,
			Rarity: Legendary, Trigger: TriggerKills},
		{ID: "ships_are_ammo", Name: "Ships Are Ammo", Description: "Lose ships in capsuleer combat",
			Category: "Battle Scars", Threshold: 16, Thresholds: []int32{1, 2, 4, 8, 16}, BasePoints: 2,
			Rarity: Legendary, Trigger: TriggerLosses, Negative: true},
		{ID: "anoikis_hunter", Name: "Anoikis Hunter", Description: "Get a kill in wormhole space",
			Category: "Locations", Threshold: 1, BasePoints: 35, Rarity: Uncommon,
			Trigger: TriggerKillsByRegion, RegionID: -1},
		{ID: "pochven_hunter", Name: "Pochven Hunter", Description: "Get a kill in Pochven",
			Category: "Locations", Threshold: 1, BasePoints: 35, Rarity: Uncommon,
			Trigger: TriggerKillsByRegion, RegionID: 10000070},
		{ID: "thera_hunter", Name: "Thera Hunter", Description: "Get a kill in Thera",
			Category: "Locations", Threshold: 1, BasePoints: 50, Rarity: Rare,
			Trigger: TriggerKillsBySystem, SystemID: 31000005},
		{ID: "zarzakh_hunter", Name: "Zarzakh Hunter", Description: "Get a kill in Zarzakh",
			Category: "Locations", Threshold: 1, BasePoints: 50, Rarity: Rare,
			Trigger: TriggerKillsBySystem, SystemID: 30100000},
		{ID: "thera_loss", Name: "Lost in Thera", Description: "Lose a ship in Thera",
			Category: "Battle Scars", Threshold: 1, BasePoints: 5, Rarity: Uncommon, Negative: true,
			Trigger: TriggerLossesBySystem, SystemID: 31000005},
		{ID: "zarzakh_loss", Name: "Lost in Zarzakh", Description: "Lose a ship in Zarzakh",
			Category: "Battle Scars", Threshold: 1, BasePoints: 5, Rarity: Uncommon, Negative: true,
			Trigger: TriggerLossesBySystem, SystemID: 30100000},
		{ID: "concordokken", Name: "Concordokken", Description: "Get destroyed by CONCORD",
			Category: "Special", Threshold: 1, BasePoints: 25, Rarity: Uncommon, Negative: true,
			Trigger: TriggerConcorded, CorporationID: 1000125},
		{ID: "banhammer_incoming", Name: "Banhammer Incoming", Description: "Destroy a CCP developer's ship",
			Category: "Special", Threshold: 1, BasePoints: 250, Rarity: Legendary,
			Trigger: TriggerKilledCorp, CorporationID: 109299958},
		{ID: "dev_killed_me", Name: "What Did You Do?", Description: "Get destroyed by a CCP developer",
			Category: "Special", Threshold: 1, BasePoints: 100, Rarity: Epic, Negative: true,
			Trigger: TriggerKilledByCorp, CorporationID: 109299958},
		{ID: "tournament_pilot", Name: "Tournament Pilot", Description: "Participate in a tournament killmail",
			Category: "Special", Threshold: 1, BasePoints: 500, Rarity: Legendary,
			Trigger: TriggerTournament, RegionID: 10000004},
		{ID: "backstab_special", Name: "Backstab Special", Description: "Destroy a member of your own corporation or alliance",
			Category: "Special", Threshold: 1, BasePoints: 25, Rarity: Rare,
			Trigger: TriggerAwox},
		{ID: "backstabbed", Name: "My Back Hurts", Description: "Get destroyed by a member of your own corporation or alliance",
			Category: "Special", Threshold: 1, BasePoints: 10, Rarity: Rare, Negative: true,
			Trigger: TriggerAwoxed},
		{ID: "ganktastic", Name: "Ganktastic", Description: "Take part in a high-security-space suicide gank",
			Category: "Special", Threshold: 25, Thresholds: []int32{1, 2, 5, 10, 25}, BasePoints: 10,
			Rarity: Legendary, Trigger: TriggerGank},
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

	for _, group := range trophyShipGroups {
		slug := fmt.Sprintf("ship_group_%d", group.id)
		out = append(out,
			Definition{
				ID: slug + "_kills", Name: group.name + " Hunter",
				Description: "Destroy " + lower(group.name) + " hulls",
				Category:    "Ship Specialization", Threshold: 625,
				Thresholds: []int32{1, 5, 25, 125, 625}, BasePoints: 5,
				Rarity: Legendary, Trigger: TriggerShipKills, GroupIDs: []int32{group.id},
			},
			Definition{
				ID: slug + "_losses", Name: group.name + " Battle Scars",
				Description: "Lose " + lower(group.name) + " hulls",
				Category:    "Battle Scars", Threshold: 16,
				Thresholds: []int32{1, 2, 4, 8, 16}, BasePoints: 1,
				Rarity: Legendary, Negative: true, Trigger: TriggerShipLosses, GroupIDs: []int32{group.id},
			},
		)
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
