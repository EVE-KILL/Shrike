package killtype

// Label describes one public killmail classification. The ID is also the
// kill-list type and kills_daily_count type, which keeps the explorer, list,
// historical SQL and live classifier on one stable contract.
type Label struct {
	ID          string
	Name        string
	Description string
	Category    string
	Search      map[string]any
}

// Labels is the public classification catalogue in display order. `latest` is
// deliberately absent: it is the unfiltered feed, not a classification.
var Labels = []Label{
	{ID: "highsec", Name: "Highsec", Category: "Space", Description: "Killmails in systems whose displayed security rounds to 0.5 or higher.", Search: searchLocation("highsec")},
	{ID: "lowsec", Name: "Lowsec", Category: "Space", Description: "Killmails in systems with security above 0.0 and below the highsec boundary.", Search: searchLocation("lowsec")},
	{ID: "nullsec", Name: "Nullsec", Category: "Space", Description: "Killmails in ordinary New Eden systems with security at or below 0.0.", Search: searchLocation("nullsec")},
	{ID: "wspace", Name: "W-Space", Category: "Space", Description: "Killmails in wormhole-space regions.", Search: searchLocation("wspace")},
	{ID: "abyssal", Name: "Abyssal", Category: "Space", Description: "Killmails in Abyssal regions.", Search: searchLocation("abyssal")},
	{ID: "pochven", Name: "Pochven", Category: "Space", Description: "Killmails in the Pochven region.", Search: searchLocation("pochven")},
	{ID: "jove", Name: "Jove Space", Category: "Space", Description: "Killmails associated with the unreachable Jove regions."},

	{ID: "timezone-au", Name: "AU Timezone", Category: "Timezone", Description: "Killmails from 08:00 through 13:59 UTC."},
	{ID: "timezone-ru", Name: "RU Timezone", Category: "Timezone", Description: "Killmails from 14:00 through 16:59 UTC."},
	{ID: "timezone-eu", Name: "EU Timezone", Category: "Timezone", Description: "Killmails from 17:00 through 21:59 UTC."},
	{ID: "timezone-us-east", Name: "US East Timezone", Category: "Timezone", Description: "Killmails from 22:00 through 03:59 UTC."},
	{ID: "timezone-us-west", Name: "US West Timezone", Category: "Timezone", Description: "Killmails from 04:00 through 07:59 UTC."},

	{ID: "solo", Name: "Solo", Category: "Engagement", Description: "PvP killmails with exactly one player attacker.", Search: map[string]any{"attackerCount": "solo"}},
	{ID: "attackers-1", Name: "1 Attacker (Non-solo)", Category: "Engagement", Description: "Killmails with one recorded attacker that are not classified as solo."},
	{ID: "attackers-2-4", Name: "2–4 Attackers", Category: "Engagement", Description: "Killmails with two through four attackers."},
	{ID: "attackers-5-9", Name: "5–9 Attackers", Category: "Engagement", Description: "Killmails with five through nine attackers."},
	{ID: "attackers-10-24", Name: "10–24 Attackers", Category: "Engagement", Description: "Killmails with ten through twenty-four attackers."},
	{ID: "attackers-25-49", Name: "25–49 Attackers", Category: "Engagement", Description: "Killmails with twenty-five through forty-nine attackers."},
	{ID: "attackers-50-99", Name: "50–99 Attackers", Category: "Engagement", Description: "Killmails with fifty through ninety-nine attackers."},
	{ID: "attackers-100-999", Name: "100–999 Attackers", Category: "Engagement", Description: "Killmails with one hundred through nine hundred ninety-nine attackers."},
	{ID: "attackers-1000-plus", Name: "1,000+ Attackers", Category: "Engagement", Description: "Killmails with at least one thousand attackers."},
	{ID: "pvp", Name: "PvP", Category: "Killmail Type", Description: "Killmails attributed to player combat."},
	{ID: "ganked", Name: "Ganked", Category: "Killmail Type", Description: "Non-NPC highsec killmails with at least ten attackers.", Search: map[string]any{"attackerType": "ganked"}},
	{ID: "npc", Name: "NPC", Category: "Engagement", Description: "Killmails classified as NPC kills.", Search: map[string]any{"attackerType": "npc"}},

	{ID: "big", Name: "1B+ ISK", Category: "Value", Description: "Killmails valued at one billion ISK or more.", Search: map[string]any{"iskValue": "1b"}},
	{ID: "5b", Name: "5B+ ISK", Category: "Value", Description: "Killmails valued at five billion ISK or more.", Search: map[string]any{"iskValue": "5b"}},
	{ID: "10b", Name: "10B+ ISK", Category: "Value", Description: "Killmails valued at ten billion ISK or more.", Search: map[string]any{"iskValue": "10b"}},
	{ID: "under-1b", Name: "Under 1B ISK", Category: "Value Bands", Description: "Killmails valued below one billion ISK."},
	{ID: "1b-5b", Name: "1B–5B ISK", Category: "Value Bands", Description: "Killmails valued from one billion up to five billion ISK."},
	{ID: "5b-10b", Name: "5B–10B ISK", Category: "Value Bands", Description: "Killmails valued from five billion up to ten billion ISK."},
	{ID: "10b-100b", Name: "10B–100B ISK", Category: "Value Bands", Description: "Killmails valued from ten billion up to one hundred billion ISK."},
	{ID: "100b-1t", Name: "100B–1T ISK", Category: "Value Bands", Description: "Killmails valued from one hundred billion up to one trillion ISK."},
	{ID: "1t-plus", Name: "1T+ ISK", Category: "Value Bands", Description: "Killmails valued at one trillion ISK or more."},

	{ID: "category-deployable", Name: "Deployable", Category: "Victim Category", Description: "Victims in the Deployable inventory category."},
	{ID: "category-drone", Name: "Drone", Category: "Victim Category", Description: "Victims in the Drone inventory category."},
	{ID: "category-fighter", Name: "Fighter", Category: "Victim Category", Description: "Victims in the Fighter inventory category."},
	{ID: "category-orbital", Name: "Orbital", Category: "Victim Category", Description: "Victims in the Orbitals inventory category."},
	{ID: "category-starbase", Name: "Starbase", Category: "Victim Category", Description: "Victims in the Starbase inventory category."},
	{ID: "category-ship", Name: "Ship", Category: "Victim Category", Description: "Victims in the Ship inventory category."},
	{ID: "category-sovereignty", Name: "Sovereignty Structure", Category: "Victim Category", Description: "Victims in the Sovereignty Structures inventory category."},
	{ID: "category-structure", Name: "Structure", Category: "Victim Category", Description: "Victims in the Structure inventory category."},
	{ID: "category-infantry", Name: "Infantry", Category: "Victim Category", Description: "Victims in the Infantry inventory category."},

	{ID: "frigates", Name: "Frigates", Category: "Victim Hull", Description: "Destroyed frigate-class hulls.", Search: searchShipCategory("frigates")},
	{ID: "destroyers", Name: "Destroyers", Category: "Victim Hull", Description: "Destroyed destroyer-class hulls.", Search: searchShipCategory("destroyers")},
	{ID: "cruisers", Name: "Cruisers", Category: "Victim Hull", Description: "Destroyed cruiser-class hulls.", Search: searchShipCategory("cruisers")},
	{ID: "battlecruisers", Name: "Battlecruisers", Category: "Victim Hull", Description: "Destroyed battlecruiser-class hulls.", Search: searchShipCategory("battlecruisers")},
	{ID: "battleships", Name: "Battleships", Category: "Victim Hull", Description: "Destroyed battleship-class hulls.", Search: searchShipCategory("battleships")},
	{ID: "capitals", Name: "Capitals", Category: "Victim Hull", Description: "Destroyed capital, supercapital, or titan hulls.", Search: searchShipCategory("capitals")},
	{ID: "freighters", Name: "Freighters", Category: "Victim Hull", Description: "Destroyed freighter or jump-freighter hulls.", Search: searchShipCategory("freighters")},
	{ID: "supercarriers", Name: "Supercarriers", Category: "Victim Hull", Description: "Destroyed supercarrier hulls.", Search: searchShipCategory("supercarriers")},
	{ID: "titans", Name: "Titans", Category: "Victim Hull", Description: "Destroyed titan hulls.", Search: searchShipCategory("titans")},
	{ID: "citadels", Name: "Structures", Category: "Victim Hull", Description: "Destroyed Upwell structure hulls.", Search: searchShipCategory("citadels")},

	{ID: "t1", Name: "Tech I", Category: "Technology", Description: "Destroyed Tech I hulls.", Search: searchTech("t1")},
	{ID: "t2", Name: "Tech II", Category: "Technology", Description: "Destroyed Tech II hulls.", Search: searchTech("t2")},
	{ID: "t3", Name: "Tech III", Category: "Technology", Description: "Destroyed Tech III strategic-cruiser hulls.", Search: searchTech("t3")},
	{ID: "faction", Name: "Faction", Category: "Technology", Description: "Destroyed faction hulls.", Search: searchTech("faction")},
}

func searchLocation(id string) map[string]any {
	return map[string]any{"location": map[string]any{"securityTypes": []string{id}}}
}

func searchShipCategory(id string) map[string]any {
	return map[string]any{"shipCategory": id}
}

func searchTech(id string) map[string]any {
	return map[string]any{"techLevel": id}
}
