package api

import (
	"fmt"
	"sort"

	"github.com/danielgtaylor/huma/v2"
)

// Prose for the public read API, keyed by operation ID.
//
// It lives here rather than at each registration site because those sites do
// not have one shape: some are `huma.Operation` literals, some come from
// helpers like entityIDOperation, and the whole SDE surface is generated from
// table specs. Threading a description through all three costs more churn than
// it saves, and a single map is easier to read as a body of documentation.
//
// Scope is the public read surface — the endpoints third parties consume, and
// the ones the previous documentation page covered. Account, administration,
// and community operations run on their summaries; for those, the summary and
// the schema are the documentation.
//
// A description set at the registration site wins. This map only fills gaps,
// so moving prose next to a handler later needs no change here.
var operationDescriptions = map[string]string{
	// --- Ported from the previous documentation index ---
	"killmails-count":        "Fast estimated row count from pg_class.reltuples. Not exact — use for dashboards and scale indicators, not for anything that requires precision.",
	"killmails":              "Cursor-paginated killmail feed with type filters (security, ship class, solo, NPC, etc.) and optional victim faction filter.",
	"killmail":               "Victim, attackers, items, prices, and sibling kills from the same fight.",
	"killmail-fitting":       "Items organised by slot (high, mid, low, rig, drone, cargo, etc.) with resolved type names and quantities.",
	"killmail-eft":           "Returns an EFT-compatible fitting block for pasting into Pyfa / EFT / the in-game fitting tool.",
	"killmail-esi":           "Killmail in the exact shape CCP's /killmails/{id}/{hash}/ endpoint returns. No resolved names.",
	"killmail-search":        "Returns killmails in ESI format within a time window, optionally filtered by location and/or entity. Filter categories are AND-ed together, IDs within a category are OR-ed, and entity filters (character/corporation/alliance) match either attackers OR the victim. Each filter field is capped at 15 IDs and at most 3 filter categories may be combined per request to keep query plans tight. Cursor-paginated on killmail_id: pass the previous response's pagination.cursor as `after` to walk backwards in time through the window.",
	"history-latest":         "Returns { killmail_id: hash } for the 10,000 most recently ingested killmails in insertion order. Pair with /killmails/:killmailId or CCP's ESI to catch up on recent activity without polling the feed stream. Cached 60s.",
	"history-date":           "Returns { killmail_id: hash } for every killmail on the given day. Use this with /killmails/:killmailId or CCP's ESI to rebuild a day's archive.",
	"characters-count":       "Fast pg_class.reltuples estimate. Not exact.",
	"character":              "Character profile with all-time stats, recent stats, top ships, top systems, and corporation history with per-stint kill/loss counts.",
	"character-stats":        "Kills, losses, ISK, efficiency, top ships, top systems for the chosen period.",
	"character-intel":        "Intelligence profile derived from killmail activity: playstyle breakdown (solo/gang/fleet/blob), FC likelihood, capital pilot flag, logi pilot flag, cyno alt detection, bait detection, awox history, top ships flown & lost, target alliances, fleet partners with corp/alliance info, and groups flown with.",
	"characters-batch-stats": "Resolve stats for up to 100 characters in a single call.",
	"character-analyze":      "Batch intel endpoint for fleet analysis and awoxer detection. Returns per-character 90-day stats: total_kills, total_losses, efficiency (%), gang_probability (% of kills that are gang), average_gang_size, cyno_probability (% of recent losses with a cyno fitted), and last_5_ships (each with ship_type_id, ship_name, kill_count, and last_loss { killmail_id, killmail_time } or null).",
	"corporations-count":     "Fast pg_class.reltuples estimate. Not exact.",
	"corporation":            "Profile with all-time stats, top members, top ships, top systems, and alliance history.",
	"alliances-count":        "Fast pg_class.reltuples estimate. Not exact.",
	"alliance":               "Profile with all-time stats and top members.",
	"coalition-stats":        "Aggregated A-vs-B stats for two coalitions. Returns exact pairwise kills/losses/ISK, per-side overall activity (not opponent-filtered), and per-side top ships used plus active systems & regions — all computed from raw killmails. Also returns a daily pairwise timeseries and an intersection of 'clashed systems' where both sides were active. Corps listed inside a side that are already members of a listed alliance on that side are dropped to prevent double counting. Window capped at 90 days.",
	"ship-fittings":          "Popular fits for a hull over the last 90 days, grouped by meta-variant family. Returns top families with canonical module list, variant count, and per-alliance doctrine-share percentages. Ships in custom_prices bypass the min-samples threshold to surface rare-hull data. Optional `modules` query filters down to fits that contain ALL listed module/charge/drone type IDs (family-aware: T1/T2/meta variants of the same item count as a hit).",
	"resolve":                "Takes up to 100 entity names and returns their IDs. Similar to CCP's /universe/ids/ but scoped to the entity types EVE-KILL tracks.",
	"location":               "Given a solar system and in-system xyz coordinates, returns the nearest named celestial (planet, moon, stargate, station, belt, POCO, etc.). Uses Euclidean 3D distance — same approach used to label killmail locations.",
	"global-stats":           "Top-N leaderboards across characters, corps, alliances, ships, systems, regions, and more. The time window is controlled by 'days' (default 7).",
	"search":                 "Trigram + ILIKE search across characters, corporations, alliances, ships, items, systems, regions, constellations, and factions. Results are ranked by a combination of similarity score, entity type, and weight (member_count, etc.).",
	"battles":                "Offset-paginated battle list with sort and date filters. Default order is newest-first (battle_id DESC).",
	"battle":                 "Teams, members, timeline, and kill totals.",
	"corporation-battles":    "Offset-paginated. `total_isk_destroyed`, `kill_count`, `losses` sort by this corporation's own contribution in each battle (SUM over battle_team_members).",
	"alliance-battles":       "Offset-paginated. `total_isk_destroyed`, `kill_count`, `losses` sort by this alliance's own contribution in each battle (SUM over battle_team_members).",
	"war":                    "Aggressor, defender, allies, and kill stats.",
	"feed-index":             "Self-describing document with topic list, usage examples, and sequence-ID semantics.",
	"feed-stream":            "Server-Sent Events stream. Send Last-Event-ID header (or ?lastEventId=) on reconnect to resume exactly where you left off.",
	"feed-poll":              "Long-poll alternative to SSE. Pass after=0 to discover the current head, then loop with after=<latest>.",
	"feed-status":            "Connected SSE clients and the latest sequence ID.",
	"sde-celestial":          "Look up any celestial (station, gate, moon, planet, etc.) by its in-game item_id.",
	"sde-prices":             "Daily price snapshot history for an item in a region. Defaults to The Forge (Jita).",
	"sde-custom-prices":      "Every price EVE-KILL applies on top of market data, in one unpaginated response. Market history is unusable for items that rarely or never trade — titans, faction capitals, officer modules — so killmail values use these overrides where they exist. `valid_until` is the last date an override applies to; open-ended ones carry the 9999-12-31 sentinel, and a type may have several rows if its override changed over time.",

	// --- Newly written ---
	"health":                    "Reports process liveness without consulting external dependencies.",
	"ready":                     "Verifies that the API can acquire its normal Postgres pool and complete a round trip.",
	"history":                   "One row per day with the number of killmails recorded. Use it to find days worth fetching before calling /history/{date}.",
	"characters":                "Cursor-paginated list of every known character. Pass limit (1-100, default 50) and cursor from the previous response.",
	"character-kills":           "Killmails where this character was an attacker, newest first. Cursor-paginated.",
	"character-losses":          "Killmails where this character was the victim, newest first. Cursor-paginated.",
	"corporations":              "Cursor-paginated list of every known corporation. Pass limit (1-100, default 50) and cursor from the previous response.",
	"corporation-stats-alltime": "Every kill and loss on record, aggregated. Heavier than the weekly variant; prefer weekly for dashboards that refresh often.",
	"corporation-stats-weekly":  "The same aggregation restricted to the last seven days. Cheaper than all-time and the better default for activity views.",
	"corporation-members":       "Characters currently recorded in this corporation, with their individual kill and loss counts.",
	"corporation-kills":         "Killmails where a member of this corporation was an attacker, newest first. Cursor-paginated.",
	"corporation-losses":        "Killmails where a member of this corporation was the victim, newest first. Cursor-paginated.",
	"corporations-batch-stats":  "Resolve stats for up to 100 corporations in a single call. Prefer this over looping the single-corporation endpoint.",
	"alliances":                 "Cursor-paginated list of every known alliance. Pass limit (1-100, default 50) and cursor from the previous response.",
	"alliance-stats-alltime":    "Every kill and loss on record, aggregated across all member corporations. Heavier than the weekly variant.",
	"alliance-stats-weekly":     "The same aggregation restricted to the last seven days. Cheaper than all-time and the better default for activity views.",
	"alliance-corporations":     "Corporations currently recorded in this alliance, with their individual kill and loss counts.",
	"alliance-members":          "Characters across every member corporation. Large for major alliances; paginate rather than fetching in one call.",
	"alliance-kills":            "Killmails where a member of this alliance was an attacker, newest first. Cursor-paginated.",
	"alliance-losses":           "Killmails where a member of this alliance was the victim, newest first. Cursor-paginated.",
	"alliances-batch-stats":     "Resolve stats for up to 100 alliances in a single call. Prefer this over looping the single-alliance endpoint.",
	"wars":                      "Wars known to the killboard, newest first. Use /wars/{id} for participants and kill totals.",

	// --- Static Data Export ---
	// List endpoints are cursor-paginated: limit is 1-100 (default 50), and
	// the response carries {data, pagination:{hasMore, cursor}}. Nested lists
	// return every row under a named key and take no pagination.
	"sde-systems":             "Cursor-paginated solar systems, ordered by solar_system_id. Filterable by region and constellation.",
	"sde-system":              "One solar system: security status, position, and its region and constellation.",
	"sde-system-kills":        "Killmails recorded in this system, newest first. This reads killboard data, not static data, so it moves independently of the rest of the SDE.",
	"sde-system-jumps":        "Stargate connections leading out of this system. Every row under jumps; not paginated.",
	"sde-system-celestials":   "Planets, moons, belts, stations, and gates in this system, in celestial_index order. Every row under celestials; not paginated.",
	"sde-regions":             "Cursor-paginated regions, ordered by region_id.",
	"sde-region":              "One region, with its name and identifiers.",
	"sde-region-kills":        "Killmails recorded anywhere in this region, newest first. Killboard data rather than static data.",
	"sde-constellations":      "Cursor-paginated constellations, ordered by constellation_id. Filterable by region.",
	"sde-constellation":       "One constellation and the region containing it.",
	"sde-types":               "Cursor-paginated inventory types — every ship, module, charge, and item in the game. Filterable by group and category.",
	"sde-type":                "One inventory type: name, description, volume, mass, and its group and category.",
	"sde-type-dogma":          "Dogma attributes and effects for this type. These drive fitting calculations. Every row under dogma; not paginated.",
	"sde-type-materials":      "Materials this type manufactures into, for build-cost calculations. Every row under materials; not paginated.",
	"sde-type-insurance":      "Insurance payout levels for this hull. Every row under levels; not paginated.",
	"sde-groups":              "Cursor-paginated inventory groups, the level between category and type. Filterable by category.",
	"sde-group":               "One inventory group and the category containing it.",
	"sde-categories":          "Cursor-paginated inventory categories, the broadest classification (Ship, Module, Charge, and so on).",
	"sde-category":            "One inventory category.",
	"sde-market-groups":       "Cursor-paginated market groups, matching the in-game market tree.",
	"sde-market-group":        "One market group and its position in the tree.",
	"sde-meta-groups":         "Meta groups (Tech I, Tech II, Faction, Officer, and so on). Small and unpaginated.",
	"sde-factions":            "Every NPC faction. Small and unpaginated.",
	"sde-faction":             "One NPC faction.",
	"sde-races":               "Every playable race. Small and unpaginated.",
	"sde-race":                "One playable race.",
	"sde-bloodlines":          "Every character bloodline. Small and unpaginated.",
	"sde-bloodline":           "One character bloodline and the race it belongs to.",
	"sde-npc-corporations":    "Cursor-paginated NPC corporations, including the starter and faction corporations.",
	"sde-npc-corporation":     "One NPC corporation.",
	"sde-flags":               "Inventory flags — the slot and container identifiers that place an item on a fitting. Small and unpaginated.",
	"sde-stations":            "Cursor-paginated NPC stations. Player structures are separate; see /sde/structures.",
	"sde-station":             "One NPC station, its services, and the system holding it.",
	"sde-station-operations":  "Station operation types, which determine the services a station offers. Small and unpaginated.",
	"sde-station-operation":   "One station operation type.",
	"sde-structures":          "Cursor-paginated player-owned structures known to the killboard. Unlike the rest of the SDE this is observed rather than shipped by CCP, so coverage depends on what has been seen.",
	"sde-structure":           "One player-owned structure, as last observed.",
	"sde-sovereignty":         "Current sovereignty holder for every claimed system. Refreshed from ESI rather than shipped with the SDE.",
	"sde-sovereignty-system":  "Current sovereignty holder for one system.",
	"sde-sovereignty-history": "Every recorded sovereignty change for this system, newest first. Every row under history; not paginated.",
}

// applyOperationDescriptions fills in the prose above, leaving any description
// already set at the registration site alone.
func applyOperationDescriptions(document *huma.OpenAPI) {
	forEachOperation(document, func(operation *huma.Operation) {
		if operation.Description != "" {
			return
		}
		if text, ok := operationDescriptions[operation.OperationID]; ok {
			operation.Description = text
		}
	})
}

// forEachOperation visits every operation in the document.
func forEachOperation(document *huma.OpenAPI, visit func(*huma.Operation)) {
	for _, item := range document.Paths {
		for _, operation := range []*huma.Operation{
			item.Get, item.Put, item.Post, item.Delete,
			item.Options, item.Head, item.Patch, item.Trace,
		} {
			if operation != nil {
				visit(operation)
			}
		}
	}
}

// checkDescribedOperations reports IDs in the map that match no operation.
// Those are the silent failure: a route renamed or removed leaves prose behind
// that no longer reaches the page, and nothing else would say so.
func checkDescribedOperations(document *huma.OpenAPI) error {
	live := map[string]bool{}
	forEachOperation(document, func(operation *huma.Operation) {
		live[operation.OperationID] = true
	})
	var stale []string
	for id := range operationDescriptions {
		if !live[id] {
			stale = append(stale, id)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		return fmt.Errorf("described operations no longer exist: %v", stale)
	}
	return nil
}
