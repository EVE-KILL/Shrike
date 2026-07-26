package relay

import (
	"sort"
	"strconv"

	"github.com/eve-kill/shrike/internal/killmail"
)

// Routing keys for a killmail.
//
// A subscriber says which keys it wants and receives only the kills carrying
// them, so this is what makes a live feed filtered by system, by alliance, or
// by "capitals only" possible without the relay understanding killmails.
//
// The group sets below are deliberately NOT the ones in internal/killtype, and
// that difference is real rather than an oversight. The kill-list types drive
// what a page query returns and are maintained against the SDE; these drive
// what a live subscriber sees. They were built at different times from
// different lists and production has been serving both. Unifying them would
// change which kills existing subscriptions deliver — a behaviour change
// dressed as a tidy-up — so they stay separate until someone decides to move
// the subscribers too.
var (
	bigGroups           = set(547, 485, 513, 902, 941, 30, 659)
	citadelGroups       = set(1657, 1406, 1404, 1408, 2017, 2016)
	t1Groups            = set(419, 27, 29, 547, 26, 420, 25, 28, 941, 463, 237, 31)
	t2Groups            = set(324, 898, 906, 540, 830, 893, 543, 541, 833, 358, 894, 831, 902, 832, 900, 834, 380)
	t3Groups            = set(963, 1305)
	frigateGroups       = set(324, 893, 25, 831, 237)
	destroyerGroups     = set(420, 541)
	cruiserGroups       = set(906, 26, 833, 358, 894, 832, 963)
	battlecruiserGroups = set(419, 540)
	battleshipGroups    = set(27, 898, 900)
	capitalGroups       = set(547, 485)
	freighterGroups     = set(513, 902)
	supercarrierGroups  = set(659)
	titanGroups         = set(30)
)

// RoutingKeys builds the keys a parsed killmail publishes under.
//
// systemSecurity is separate because it is not on the killmail row — it comes
// from the static-data cache — and hasSecurity distinguishes "security 0.0",
// which is a real nullsec value, from "we do not know the system".
func RoutingKeys(p *killmail.Parsed, systemSecurity float64, hasSecurity bool) []string {
	keys := map[string]bool{"all": true}
	km := p.Killmail

	// Value brackets are not exclusive: a 12b kill is both.
	if km.TotalValue >= 10_000_000_000 {
		keys["10b"] = true
	}
	if km.TotalValue >= 5_000_000_000 {
		keys["5b"] = true
	}

	if km.RegionID >= 12_000_000 && km.RegionID <= 13_000_000 {
		keys["abyssal"] = true
	}
	if km.RegionID >= 11_000_001 && km.RegionID <= 11_000_033 {
		keys["wspace"] = true
	}

	if hasSecurity {
		switch {
		case systemSecurity >= 0.45:
			keys["highsec"] = true
		case systemSecurity >= 0.0:
			keys["lowsec"] = true
		default:
			keys["nullsec"] = true
		}
	}

	for key, groups := range map[string]map[int32]bool{
		"big": bigGroups, "citadel": citadelGroups,
		"t1": t1Groups, "t2": t2Groups, "t3": t3Groups,
		"frigates": frigateGroups, "destroyers": destroyerGroups,
		"cruisers": cruiserGroups, "battlecruisers": battlecruiserGroups,
		"battleships": battleshipGroups, "capitals": capitalGroups,
		"freighters": freighterGroups, "supercarriers": supercarrierGroups,
		"titans": titanGroups,
	} {
		if groups[km.VictimShipGroupID] {
			keys[key] = true
		}
	}

	if km.IsSolo {
		keys["solo"] = true
	}
	if km.IsNPC {
		keys["npc"] = true
	}

	// Entity keys are the ones that make "follow this alliance" work. Victim
	// and attacker are separate namespaces so a subscriber can ask for losses
	// without kills.
	for _, id := range []int32{
		km.VictimCharacterID, km.VictimCorporationID, km.VictimAllianceID, km.VictimFactionID,
	} {
		if id != 0 {
			keys["victim."+itoa(id)] = true
		}
	}
	for _, a := range p.Attackers {
		for _, id := range []int32{a.CharacterID, a.CorporationID, a.AllianceID, a.FactionID} {
			if id != 0 {
				keys["attacker."+itoa(id)] = true
			}
		}
	}

	for prefix, id := range map[string]int32{
		"system.": km.SolarSystemID, "region.": km.RegionID, "constellation.": km.ConstellationID,
	} {
		if id != 0 {
			keys[prefix+itoa(id)] = true
		}
	}

	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	// Sorted so the published payload is stable between runs — otherwise two
	// identical killmails produce different JSON and nothing downstream can be
	// compared or cached by content.
	sort.Strings(out)
	return out
}

func set(ids ...int32) map[int32]bool {
	m := make(map[int32]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func itoa(v int32) string { return strconv.FormatInt(int64(v), 10) }
