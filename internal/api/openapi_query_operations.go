package api

import "github.com/danielgtaylor/huma/v2"

// Query parameters per operation. See openapi_query.go for why they live here
// and what they are (and are not) allowed to do.
//
// Every entry was read off the handler that serves it, so the names, bounds,
// defaults, and value sets match the parsing rather than the intent. Where an
// alias shares a handler with its canonical route, both entries point at the
// same slice — they cannot describe different parameters because they cannot
// read different ones.

// --- parameter sets shared by several operations ------------------------------

// idPagination is the cursor pair used by the entity kill and loss lists,
// which page through killmail IDs in either direction.
func idPagination() []*huma.Param {
	return []*huma.Param{
		limitQuery(50, 10, 100),
		afterQuery(),
		beforeQuery(),
	}
}

// namedListPagination is the SDE and entity list contract: a name prefix
// filter plus a forward cursor.
func namedListPagination(fallback, minimum, maximum int) []*huma.Param {
	return []*huma.Param{
		textQuery("name", "Case-insensitive name prefix to match."),
		limitQuery(fallback, minimum, maximum),
		afterQuery(),
	}
}

// killlistParams is the paged killmail list shared by the universe, custom
// domain, and entity kill lists.
func killlistParams() []*huma.Param {
	return []*huma.Param{
		limitQuery(50, 10, 100),
		afterQuery(),
		offsetPageQuery(maxInt32Page),
	}
}

// mostValuableParams is the trailing-window most-valuable list shared by the
// universe and entity pages.
func mostValuableParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"dataType",
			"Restrict the losses to one category of hull.",
			"most_valuable_kills",
			"most_valuable_kills",
			"most_valuable_ships",
			"most_valuable_structures",
		),
		daysQuery(7, 1, 30),
		limitQuery(8, 1, 32),
	}
}

// characterStatsParams is the period selector on /characters/{id}/stats. The
// corporation and alliance equivalents are fixed-period routes and read
// nothing from the query.
func characterStatsParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"type", "Reporting period.", "alltime",
			"alltime", "weekly", "range",
		),
		textQuery(
			"from",
			"Start date as YYYY-MM-DD. Required when `type` is `range`.",
		),
		textQuery(
			"to",
			"End date as YYYY-MM-DD. Required when `type` is `range`.",
		),
	}
}

// esiLogParams is the ESI request log filter shared by the account view and
// its legacy alias.
func esiLogParams() []*huma.Param {
	return []*huma.Param{
		limitQuery(50, 1, 100),
		openPageQuery(),
		textQuery("source", "Match the recorded request source exactly."),
		esiLogStatusQuery(),
		esiLogEndpointTypeQuery(),
		intQuery("after_id", "Return log rows below this log ID."),
	}
}

func esiLogStatusQuery() *huma.Param {
	return enumQuery(
		"status", "Restrict to successful or failed requests.", "",
		"success", "error",
	)
}

func esiLogEndpointTypeQuery() *huma.Param {
	return textQuery(
		"endpoint_type",
		"Restrict to one ESI endpoint family, for example `killmails`.",
	)
}

// commentThreadParams is the cursor pair on the comment lists.
func commentThreadParams() []*huma.Param {
	return []*huma.Param{
		limitQuery(50, 1, 100),
		cursorQuery(),
	}
}

// conflictListParams is the page and limit pair the conflict surface shares.
func conflictListParams(fallback, minimum, maximum int) []*huma.Param {
	return []*huma.Param{
		pageQuery(conflictDefaultPage, 1, conflictMaximumPage),
		limitQuery(fallback, minimum, maximum),
	}
}

// warMemberParams is the member breakdown shared by wars and faction war.
func warMemberParams(limitFallback, limitMaximum int) []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"side", "Which side of the war to report on.", "combined",
			"aggressor", "defender", "combined",
		),
		enumQuery(
			"sort", "Ordering for the member rows.", "activity",
			"kills", "losses", "isk", "activity",
		),
		limitQuery(limitFallback, 1, limitMaximum),
		entityQuery(
			"corporationId", "Restrict to one corporation.", "corporation",
		),
		entityQuery("allianceId", "Restrict to one alliance.", "alliance"),
	}
}

// fittingSearchParams is the shared fitting search contract.
func fittingSearchParams() []*huma.Param {
	return []*huma.Param{
		requiredQuery(entityQuery(
			"ship", "Hull type ID to search fittings for. Required.", "ship",
		)),
		jsonQuery(
			"filters",
			"JSON array of at most 8 module or role filters, each "+
				"`{\"kind\":\"module\"|\"role\",...}`.",
		),
		limitQuery(24, 1, 50),
		cappedQuery("offset", "Rows to skip before the page.", 0, maxInt32Page),
	}
}

func communityFittingsParams() []*huma.Param {
	return []*huma.Param{limitQuery(20, 1, 50)}
}

// entityTopParams is the leaderboard selector on the per-entity top lists.
func entityTopParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"slice",
			"Which half of the leaderboard set to build. `right` also "+
				"accepts `days=alltime`.",
			"left", "left", "right",
		),
		textQuery(
			"days",
			"Window in days, between 1/24 and 365. Send `alltime` with "+
				"`slice=right` for the unbounded set. Default 7.",
		),
	}
}

func entityPageKilllistParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"role", "Whose killmails to return.", "combined",
			"kills", "losses", "combined",
		),
		limitQuery(50, 10, 100),
		afterQuery(),
		openOffsetPageQuery(),
		intQuery("ship_group", "Restrict to one victim ship group ID."),
	}
}

func entityPageMemberParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"sort", "Ordering for the member rows.", "name",
			"name", "last_active", "security_status",
		),
		limitQuery(100, 1, 200),
		openPageQuery(),
		entityQuery(
			"corporation_id",
			"Restrict an alliance's members to one corporation.",
			"corporation",
		),
		boundedQuery(
			"activity",
			"Only members active within this many days. 0 disables the filter.",
			0, 0, 100000,
		),
	}
}

func entityPageCorporationParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"sort", "Ordering for the corporation rows.", "member_count",
			"member_count", "name",
		),
		enumQuery("dir", "Sort direction.", "desc", "asc", "desc"),
	}
}

// maxInt32Page mirrors the ceiling the handlers clamp page numbers to.
const maxInt32Page = 2147483647

// --- the table ----------------------------------------------------------------

var operationQueryParameters = map[string][]*huma.Param{
	// --- Killboard -----------------------------------------------------------
	"killlist": {
		killTypeQuery(),
		limitQuery(50, 10, 100),
		afterQuery(),
		offsetPageQuery(maxInt32Page),
		victimFactionsQuery(),
	},
	"killlist-advanced": {
		jsonQuery(
			"filters",
			"JSON filter tree: `entities`, `items`, `location`, "+
				"`timeRange`, `attackerCount`, `attackerType`, `iskValue`, "+
				"`iskMin`, `iskMax`, `shipCategory`, `techLevel`, and "+
				"`sort`. Each ID-bearing list holds at most 15 entries.",
		),
		enumQuery(
			"view", "Return killmails or the fittings behind them.",
			"kills", "kills", "fits",
		),
		enumQuery(
			"dedup",
			"Group the `fits` view by exact fit or by fit family.",
			"none", "none", "exact", "family",
		),
		textQuery("fitHash", "64-character hex fit hash to drill into."),
		textQuery(
			"familyHash", "64-character hex fit family hash to drill into.",
		),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"killmails": {
		killTypeQuery(),
		limitQuery(50, 10, 100),
		afterQuery(),
		beforeQuery(),
		victimFactionsQuery(),
	},
	"kills-most-valuable": {
		killTypeQuery(),
		daysQuery(7, 1, 365),
		limitQuery(7, 1, 20),
	},
	"kills-top": {
		killTypeQuery(),
		enumQuery(
			"dataType", "Which leaderboard to build.", "characters",
			"characters", "corporations", "alliances",
			"ships", "systems", "regions",
		),
		daysQuery(7, 1, 365),
		limitQuery(10, 1, 50),
	},
	"feed-poll": {
		limitQuery(100, 1, 1000),
		afterQuery(),
	},

	// --- Entities ------------------------------------------------------------
	"characters":            namedListPagination(50, 1, 1000),
	"corporations":          namedListPagination(50, 1, 1000),
	"alliances":             namedListPagination(50, 1, 1000),
	"character-kills":       idPagination(),
	"character-losses":      idPagination(),
	"corporation-kills":     idPagination(),
	"corporation-losses":    idPagination(),
	"alliance-kills":        idPagination(),
	"alliance-losses":       idPagination(),
	"corporation-members":   {limitQuery(50, 1, 1000), afterQuery()},
	"alliance-members":      {limitQuery(50, 1, 1000), afterQuery()},
	"alliance-corporations": {limitQuery(50, 1, 1000), afterQuery()},
	"character-stats":       characterStatsParams(),
	"character-intel": {
		daysQuery(365, 7, 365),
	},
	"entity-resolve":        entityResolveParams(),
	"entity-resolve-compat": entityResolveParams(),

	// --- Entity pages --------------------------------------------------------
	"entity-page-killlist":                     entityPageKilllistParams(),
	"entity-page-killlist-generic-compat":      entityPageKilllistParams(),
	"entity-page-most-valuable":                mostValuableParams(),
	"entity-page-most-valuable-generic-compat": mostValuableParams(),
	"entity-page-members":                      entityPageMemberParams(),
	"entity-page-members-alliance-compat":      entityPageMemberParams(),
	"entity-page-members-corporation-compat":   entityPageMemberParams(),
	"entity-page-corporations":                 entityPageCorporationParams(),
	"entity-page-corporations-alliance-compat": entityPageCorporationParams(),
	"entity-page-stats":                        entityPageStatsParams(),
	"entity-page-stats-alliance-compat":        entityPageStatsParams(),
	"entity-page-stats-character-compat":       entityPageStatsParams(),
	"entity-page-stats-corporation-compat":     entityPageStatsParams(),
	"entity-page-top-lists":                    entityPageStatsParams(),
	"entity-page-top-lists-generic-compat":     entityPageStatsParams(),
	"entity-page-intel":                        entityPageIntelParams(),
	"entity-page-intel-alliance-compat":        entityPageIntelParams(),
	"entity-page-intel-character-compat":       entityPageIntelParams(),
	"entity-page-intel-corporation-compat":     entityPageIntelParams(),
	"entity-top-character-compat":              entityTopParams(),
	"entity-top-corporation-compat":            entityTopParams(),
	"entity-top-alliance-compat":               entityTopParams(),

	// --- Universe kill lists -------------------------------------------------
	"universe-system-killmails":            killlistParams(),
	"universe-region-killmails":            killlistParams(),
	"universe-constellation-killmails":     killlistParams(),
	"system-killlist-compat":               killlistParams(),
	"region-killlist-compat":               killlistParams(),
	"constellation-killlist-compat":        killlistParams(),
	"universe-type-killmails":              {limitQuery(50, 10, 100), afterQuery()},
	"item-killlist-compat":                 {limitQuery(50, 10, 100), afterQuery()},
	"ship-killlist-compat":                 {limitQuery(50, 10, 100), afterQuery()},
	"universe-system-most-valuable":        mostValuableParams(),
	"universe-region-most-valuable":        mostValuableParams(),
	"universe-constellation-most-valuable": mostValuableParams(),
	"system-most-valuable-compat":          mostValuableParams(),
	"region-most-valuable-compat":          mostValuableParams(),
	"constellation-most-valuable-compat":   mostValuableParams(),

	// --- Battles -------------------------------------------------------------
	"battles":             battleListParams(),
	"alliance-battles":    battleListParams(),
	"corporation-battles": battleListParams(),
	"battle-report-killlist": {
		limitQuery(conflictDefaultLimit, 10, conflictMaximumLimit),
		afterQuery(),
		offsetPageQuery(conflictMaximumPage),
	},
	"killmail-battle-killlist": {
		limitQuery(conflictDefaultLimit, 10, conflictMaximumLimit),
		afterQuery(),
		offsetPageQuery(conflictMaximumPage),
	},
	"battle-report-most-valuable":   battleMostValuableParams(),
	"killmail-battle-most-valuable": battleMostValuableParams(),

	// --- Conflicts and wars --------------------------------------------------
	"conflict-battles": append(
		conflictListParams(50, 10, 50),
		intQuery("year", "Restrict to battles that started in this year."),
		intQuery("minKills", "Minimum killmail count in the battle."),
		numberQuery("minIsk", "Minimum ISK destroyed in the battle."),
		flagQuery(
			"custom",
			"Restrict to battles inside the custom domain's own scope.",
		),
		entityQuery("allianceId", "Restrict to one alliance.", "alliance"),
		entityQuery(
			"corporationId", "Restrict to one corporation.", "corporation",
		),
		entityQuery("characterId", "Restrict to one character.", "character"),
		entityQuery(
			"constellationId", "Restrict to one constellation.",
			"constellation",
		),
		entityQuery("regionId", "Restrict to one region.", "region"),
		entityQuery("systemId", "Restrict to one solar system.", "system"),
	),
	"conflict-wars": append(
		conflictListParams(conflictDefaultLimit, 1, conflictMaximumLimit),
		flagQuery("upcoming", "Only wars that have not started yet."),
		flagQuery("finished", "Only wars that have finished."),
		flagQuery("ongoing", "Only wars that are currently running."),
		flagQuery("mutual", "Only wars both parties agreed to."),
		flagQuery("hasActivity", "Only wars with recorded activity."),
		flagQuery("hasKills", "Only wars with at least one killmail."),
		flagQuery("hasAllies", "Only wars that have allies."),
		enumQuery(
			"sort",
			"Ordering. Ignored while `upcoming` is set, which always "+
				"orders by start date.",
			"",
			"kills", "isk",
		),
		entityQuery("allianceId", "Restrict to one alliance.", "alliance"),
		entityQuery(
			"corporationId", "Restrict to one corporation.", "corporation",
		),
		entityQuery("characterId", "Restrict to one character.", "character"),
	),
	"wars": {
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"wars-eligible": {
		enumQuery(
			"type", "Which kind of war target to list.", "corporations",
			"corporations", "alliances",
		),
	},
	"war-members": warMemberParams(
		conflictMaximumMembers, conflictMaximumMembers,
	),
	"war-killlist": {
		limitQuery(conflictDefaultLimit, 10, conflictMaximumLimit),
		afterQuery(),
		offsetPageQuery(conflictMaximumPage),
		timestampQuery("warStart", "Override the window start."),
		timestampQuery("warEnd", "Override the window end."),
		textQuery(
			"warSideCorps", "Comma-separated corporation IDs on the side.",
		),
		textQuery(
			"warSideAlliances", "Comma-separated alliance IDs on the side.",
		),
	},

	// --- Faction war ---------------------------------------------------------
	"faction-war-dashboard-detail": {factionWarDaysQuery()},
	"faction-war-intel":            {factionWarDaysQuery()},
	"faction-war-dashboard": {
		factionWarDaysQuery(),
		factionWarPeriodQuery(),
	},
	"faction-war-overview": {factionWarPeriodQuery()},
	"faction-war-members": append(
		warMemberParams(factionWarDefaultLimit, factionWarMaximumLimit),
		factionWarDaysQuery(),
	),

	// --- Campaigns -----------------------------------------------------------
	"campaigns": {
		enumQuery(
			"status",
			"Which campaigns to list. Defaults to `all` when the request "+
				"is scoped to an entity.",
			"active",
			"active", "archived", "private", "all",
		),
		enumQuery(
			"mode", "Restrict to two-sided conflicts or area campaigns.", "",
			"conflict", "area",
		),
		enumQuery(
			"entityType",
			"Scope the list to one entity. Requires `entityId`.", "",
			"character", "corporation", "alliance",
		),
		intQuery(
			"entityId", "Entity ID to scope to. Requires `entityType`.",
		),
		textQuery("q", "Name search, truncated to 100 characters."),
		pageQuery(1, 1, campaignMaximumPage),
	},
	"campaign-killmails":       campaignKilllistParams(),
	"campaign-killlist-legacy": campaignKilllistParams(),

	// --- Statistics ----------------------------------------------------------
	"global-stats": {
		requiredQuery(enumQuery(
			"dataType", "Which leaderboard to build. Required.", "",
			"characters", "corporations", "alliances", "factions", "ships", "systems",
			"regions", "isk_destroyers_chars", "isk_destroyers_corps",
			"isk_destroyers_alliances", "solo_killers", "top_points",
			"dangerous_systems", "deadliest_regions", "most_used_ships",
			"most_destroyed_ships", "biggest_losers", "pirate_characters",
			"carebear_characters", "most_valuable_kills",
			"most_valuable_ships", "most_valuable_structures",
		)),
		cappedNumberQuery(
			"days",
			"Window in days. Values below 1 select the realtime hourly "+
				"window instead.",
			7, 90,
		),
		cappedQuery("limit", "Maximum results to return.", 10, 100),
	},
	"stats-rankings": {
		requiredQuery(enumQuery(
			"section", "Which ranking to build. Required.", "",
			"largest", "security", "growth", "newest", "achievements", "eve-kill",
		)),
		enumQuery(
			"entityType", "Which entity the ranking covers.", "alliance",
			"character", "corporation", "alliance", "ship", "system", "region",
		),
		enumQuery(
			"window", "EVE-KILL ranking window.", "all_time",
			"weekly", "ninety_days", "all_time",
		),
		enumQuery(
			"rank",
			"Security band. Only read when `section` is `security`.",
			"pirate", "pirate", "carebear",
		),
		enumQuery(
			"direction",
			"Growth direction. Only read when `section` is `growth`.",
			"growing", "growing", "shrinking",
		),
		flooredQuery(
			"days",
			"Growth window in days. Only read when `section` is `growth`.",
			7, 1,
		),
		cappedQuery("limit", "Maximum results to return.", 10, 50),
	},
	"history": {
		intQuery(
			"year",
			"Restrict the daily counts to one year. Omit for every day on "+
				"record.",
		),
	},

	// --- Search, graph, map, market ------------------------------------------
	"search": {
		requiredQuery(textQuery("q", "Search text. Required.")),
		textQuery(
			"type",
			"Comma-separated entity kinds to search: `character`, "+
				"`corporation`, `alliance`, `faction`, `ship`, "+
				"`shipgroup`, `system`, `region`, `constellation`. Omit to "+
				"search all of them.",
		),
		cappedQuery("limit", "Maximum results to return.", 25, 50),
	},
	"graph": {
		enumQuery(
			"mode", "Which graph query to run.", "path_finder",
			"path_finder", "coalitions", "rivalries", "entity_intel",
			"hunting_grounds", "hot_zones", "migration", "spy_check",
			"census",
		),
		enumQuery(
			"entityType",
			"Entity kind for the modes that take one. Anything other than "+
				"`corporation` is read as `alliance`.",
			"alliance", "character", "corporation", "alliance",
		),
		intQuery("entityId", "Entity ID for the entity-scoped modes."),
		entityQuery(
			"fromId", "Starting character for `path_finder`.", "character",
		),
		entityQuery(
			"toId", "Destination character for `path_finder`.", "character",
		),
		limitQuery(50, 10, 100),
	},
	"map-scope": {
		enumQuery(
			"type", "Which slice of New Eden to return.", "new-eden",
			"new-eden", "zarzakh", "wormhole", "abyssal", "proving",
		),
	},
	"bulk-prices": {
		textQuery(
			"types",
			"Comma-separated inventory type IDs. Omit for an empty price map.",
		),
	},
	"ship-matchup": {
		requiredQuery(entityQuery(
			"attacker", "Attacking hull type ID. Required.", "ship",
		)),
		requiredQuery(entityQuery(
			"victim", "Victim hull type ID. Required.", "ship",
		)),
	},
	"location": {
		requiredQuery(entityQuery(
			"system_id", "Solar system to resolve within.", "system",
		)),
		requiredQuery(numberQuery("x", "X coordinate in metres.")),
		requiredQuery(numberQuery("y", "Y coordinate in metres.")),
		requiredQuery(numberQuery("z", "Z coordinate in metres.")),
	},

	// --- Fittings ------------------------------------------------------------
	"fittings-search":                     fittingSearchParams(),
	"fittings-search-legacy":              fittingSearchParams(),
	"fittings-community-latest":           communityFittingsParams(),
	"fittings-community-latest-legacy":    communityFittingsParams(),
	"fittings-community-top-rated":        communityFittingsParams(),
	"fittings-community-top-rated-legacy": communityFittingsParams(),
	"ship-fittings": {
		textQuery(
			"modules",
			"Comma-separated module type IDs the fit must contain.",
		),
		limitQuery(defaultFitFamilies, 1, 50),
	},

	// --- Static data ---------------------------------------------------------
	"sde-types": {
		textQuery("name", "Case-insensitive name prefix to match."),
		sdePublishedQuery(),
		intQuery("group_id", "Restrict to one inventory group."),
		intQuery("category_id", "Restrict to one inventory category."),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"sde-groups": {
		sdePublishedQuery(),
		intQuery("category_id", "Restrict to one inventory category."),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"sde-categories":    {sdePublishedQuery()},
	"sde-market-groups": {intQuery("parent_id", "Restrict to one parent market group.")},
	"sde-systems": {
		textQuery("name", "Case-insensitive system name prefix to match."),
		entityQuery("region_id", "Restrict to one region.", "region"),
		entityQuery(
			"constellation_id", "Restrict to one constellation.",
			"constellation",
		),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"sde-constellations": {
		entityQuery("region_id", "Restrict to one region.", "region"),
	},
	"sde-stations": {
		entityQuery(
			"solar_system_id", "Restrict to one solar system.", "system",
		),
		entityQuery("region_id", "Restrict to one region.", "region"),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"sde-structures": {
		entityQuery(
			"solar_system_id", "Restrict to one solar system.", "system",
		),
		entityQuery("region_id", "Restrict to one region.", "region"),
		intQuery("owner_id", "Restrict to one owning corporation."),
		limitQuery(50, 1, 100),
		afterQuery(),
	},
	"sde-sovereignty": {
		entityQuery("alliance_id", "Restrict to one alliance.", "alliance"),
		entityQuery("faction_id", "Restrict to one faction.", "faction"),
	},
	"sde-prices": {
		entityQuery(
			"region_id",
			"Market region to price against. Defaults to The Forge "+
				"(10000002).",
			"region",
		),
		boundedQuery(
			"limit", "Most recent daily rows to return.", 30, 1, 365,
		),
	},
	"sde-system-kills": idPagination(),
	"sde-region-kills": idPagination(),

	// --- Historical archive --------------------------------------------------
	"legacy-archive-kills": {
		textQuery("victim", "Victim name contains this text."),
		textQuery("corp", "Victim corporation name contains this text."),
		textQuery("alliance", "Victim alliance name contains this text."),
		textQuery("system", "System name contains this text."),
		textQuery(
			"ship",
			"Comma-separated hull names; a row matches any of them.",
		),
		textQuery("attacker", "Attacker name contains this text."),
		textQuery("from", "Earliest killmail date, as YYYY-MM-DD."),
		textQuery("to", "Latest killmail date, as YYYY-MM-DD."),
		enumQuery(
			"sort",
			"Sort field and direction joined by an underscore.",
			"id_desc",
			"id_desc", "id_asc", "value_desc", "value_asc",
			"time_desc", "time_asc",
		),
		limitQuery(50, 10, 100),
		afterQuery(),
	},
	"legacy-archive-top": {
		enumQuery(
			"dataType", "Which leaderboard to build.", "characters",
			"characters", "corporations", "alliances", "ships", "systems",
		),
		intQuery("year", "Restrict to one year."),
		boundedQuery("limit", "Maximum results to return.", 10, 0, 50),
	},
	"legacy-archive-autocomplete": {
		textQuery("q", "Search text. At least two characters."),
		enumQuery(
			"field", "Which column to complete.", "victim",
			"victim", "attacker", "corp", "alliance", "system",
		),
		boundedQuery("limit", "Maximum suggestions to return.", 10, 0, 20),
	},

	// --- Comments ------------------------------------------------------------
	"comments-feed": append(
		commentThreadParams(),
		commentTargetTypeQuery(),
		entityQuery("character_id", "Restrict to one author.", "character"),
		entityQuery(
			"corporation_id", "Restrict to one author corporation.",
			"corporation",
		),
		entityQuery(
			"alliance_id", "Restrict to one author alliance.", "alliance",
		),
		textQuery(
			"q",
			"Body search. Applied once the text is at least two characters.",
		),
	),
	"comments-thread": append(
		commentThreadParams(),
		commentTargetTypeQuery(),
		intQuery("target_id", "Numeric ID of the commented object."),
		textQuery("target_slug", "Slug of the commented object."),
	),
	"my-comments":             {limitQuery(25, 1, 100), cursorQuery()},
	"my-comments-live-alias":  {limitQuery(25, 1, 100), cursorQuery()},
	"comments-klipy-search":   klipyParams(),
	"comments-klipy-trending": klipyParams(),

	// --- Blog ----------------------------------------------------------------
	"blog-posts": {
		limitQuery(20, 1, 50),
		cursorQuery(),
		textQuery("tag", "Restrict to one tag."),
	},

	// --- Custom domains ------------------------------------------------------
	"domain-killlist": {
		killTypeQuery(),
		limitQuery(50, 10, 100),
		afterQuery(),
	},
	"domain-system-killlist":        domainLocationKilllistParams(),
	"domain-region-killlist":        domainLocationKilllistParams(),
	"domain-constellation-killlist": domainLocationKilllistParams(),
	"domain-kills-most-valuable": {
		killTypeQuery(),
		daysQuery(7, 1, 90),
		limitQuery(7, 1, 20),
	},
	"domain-kills-top": {
		killTypeQuery(),
		domainDataTypeQuery(),
		daysQuery(7, 1, 90),
		limitQuery(10, 1, 50),
	},
	"domain-statistics": {
		domainDataTypeQuery(),
		daysQuery(7, 1, 90),
		limitQuery(10, 1, 100),
	},
	"domain-asset-preview": {
		textQuery(
			"token", "Signed preview token for an unapproved asset.",
		),
	},
	"image-domain-asset-preview": {
		textQuery(
			"token", "Signed preview token for an unapproved asset.",
		),
	},
	"domain-subdomain-check": {
		textQuery("subdomain", "Subdomain to test."),
	},
	"domain-subdomain-check-compat": {
		textQuery("subdomain", "Subdomain to test."),
	},
	"domain-campaign-search": {
		textQuery("q", "Campaign name search."),
	},
	"domain-campaign-search-compat": {
		textQuery("q", "Campaign name search."),
	},
	"domain-assets-delete-type": {domainAssetTypeQuery()},

	// --- Account -------------------------------------------------------------
	"account-esi-logs":             esiLogParams(),
	"user-esi-logs-compat":         esiLogParams(),
	"account-notification-replies": notificationParams(),
	"notification-replies-compat":  notificationParams(),
	"wallet-account":               {walletPageQuery()},
	"wallet-account-legacy":        {walletPageQuery()},
	"wallet-public":                walletParams(),
	"other-sessions-revoke":        {sessionExceptQuery()},
	"other-sessions-revoke-legacy": {sessionExceptQuery()},

	// --- Authentication ------------------------------------------------------
	"eve-login-start":           loginParams(),
	"auth-login-legacy":         loginParams(),
	"eve-login-callback":        callbackParams(),
	"eve-login-callback-legacy": callbackParams(),

	// --- Administration ------------------------------------------------------
	"admin-users-list": {
		textQuery("search", "Match a character name or ID."),
		enumQuery(
			"sort", "Ordering for the user rows.", "last_login",
			"last_login", "created_at", "character_name",
		),
		enumQuery("dir", "Sort direction.", "desc", "asc", "desc"),
		limitQuery(50, 1, 100),
		openPageQuery(),
	},
	"admin-esi-logs": append(
		esiLogParams(),
		textQuery("search", "Match an endpoint or error message."),
		entityQuery(
			"character_id", "Restrict to one character.", "character",
		),
		entityQuery(
			"corporation_id", "Restrict to one corporation.", "corporation",
		),
		flagQuery("has_new", "Only requests that returned new items."),
	),
	"admin-esi-entities": {
		textQuery("q", "Entity name or ID search."),
	},
	"admin-river-jobs": {
		textQuery("queue", "Restrict jobs to one River queue."),
		enumQuery("state", "Restrict jobs to one River state.", "",
			"available", "cancelled", "completed", "discarded", "pending", "retryable", "running", "scheduled"),
		limitQuery(50, 1, 200),
		intQuery("before_id", "Return jobs below this River job ID."),
	},
	"admin-comments":                    {moderationFilterQuery(), limitQuery(50, 1, 200)},
	"admin-comments-live-queue-alias":   {moderationFilterQuery(), limitQuery(50, 1, 200)},
	"admin-moderation":                  moderationQueueParams(),
	"admin-moderation-live-queue-alias": moderationQueueParams(),
	"announcement-admin-list": {
		enumQuery(
			"status", "Restrict to one lifecycle state.", "",
			"active", "scheduled", "expired", "archived",
		),
		rangeQuery("tier", "Restrict to one tier.", 1, 3),
		limitQuery(50, 1, 200),
	},
	"blog-admin-list": {
		enumQuery(
			"status", "Restrict to one lifecycle state.", "",
			"draft", "published", "archived",
		),
		limitQuery(50, 1, 200),
	},
	"campaign-admin-list": {
		enumQuery(
			"state", "Restrict to one processing state.", "",
			"pending", "active", "archived", "paused", "failed",
		),
		textQuery("q", "Campaign name, ID, or creator search."),
		pageQuery(1, 1, campaignMaximumPage),
	},
	"wallet-admin": walletParams(),
}

// --- small shared sets ---------------------------------------------------------

func entityResolveParams() []*huma.Param {
	return []*huma.Param{
		requiredQuery(enumQuery(
			"type", "Entity kind to resolve. Required.", "",
			"character", "corporation", "alliance", "faction",
		)),
		requiredQuery(intQuery("id", "Entity ID to resolve. Required.")),
	}
}

func entityPageStatsParams() []*huma.Param {
	return []*huma.Param{
		defaultedQuery(
			"days",
			"Trailing window in days. 0 covers the whole record.",
			0,
		),
	}
}

func entityPageIntelParams() []*huma.Param {
	return []*huma.Param{
		defaultedQuery("days", "Trailing window in days.", 90),
	}
}

func battleListParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"sort", "Ordering for the battle rows.", "battle_id",
			"battle_id", "total_isk_destroyed", "kill_count", "start_time",
		),
		enumQuery("order", "Sort direction.", "desc", "asc", "desc"),
		limitQuery(20, 1, 100),
		pageQuery(1, 1, 500),
		timestampQuery("start_after", "Only battles starting after this."),
		timestampQuery("start_before", "Only battles starting before this."),
	}
}

func battleMostValuableParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"dataType", "Restrict the losses to one category of hull.",
			"most_valuable_kills",
			"most_valuable_kills", "most_valuable_ships",
			"most_valuable_structures",
		),
		intQuery("team", "Restrict to one team index in the battle."),
		limitQuery(8, 1, 32),
	}
}

func campaignKilllistParams() []*huma.Param {
	return []*huma.Param{
		intQuery(
			"side",
			"Restrict to one campaign side index. Must match a side the "+
				"campaign defines.",
		),
		limitQuery(50, 10, 100),
		afterQuery(),
	}
}

func factionWarDaysQuery() *huma.Param {
	return daysQuery(
		factionWarDefaultDays, 1, factionWarMaximumDays,
	)
}

func factionWarPeriodQuery() *huma.Param {
	return enumQuery(
		"period", "Reporting window.", "last_week",
		"yesterday", "last_week", "active_total",
	)
}

func commentTargetTypeQuery() *huma.Param {
	return rangeQuery(
		"target_type",
		"Numeric target kind: 1 killmail, 2 character, 3 corporation, "+
			"4 alliance, 5 system, 6 page, 7 battle, 8 fit, 9 blog, "+
			"10 campaign.",
		1, 10,
	)
}

func klipyParams() []*huma.Param {
	return []*huma.Param{
		textQuery("q", "Search text."),
		openPageQuery(),
		boundedQuery("per_page", "Results per page.", 24, 1, 48),
	}
}

func moderationFilterQuery() *huma.Param {
	return enumQuery(
		"filter",
		"Which comments to queue. Unknown values behave like `all`.",
		"flagged", "flagged", "reported", "all",
	)
}

func moderationQueueParams() []*huma.Param {
	return []*huma.Param{
		enumQuery(
			"kind", "Restrict to one queue.", "all",
			"all", "comments", "bios", "bio_character",
			"bio_corporation", "bio_alliance",
		),
		enumQuery(
			"status", "Restrict to one review state.", "pending",
			"all", "pending", "auto_approved", "auto_rejected",
			"approved", "rejected",
		),
		limitQuery(50, 1, 200),
		cursorQuery(),
	}
}

func domainLocationKilllistParams() []*huma.Param {
	return []*huma.Param{limitQuery(50, 10, 100), afterQuery()}
}

func domainDataTypeQuery() *huma.Param {
	return enumQuery(
		"dataType", "Which leaderboard to build.", "characters",
		"characters", "corporations", "alliances", "ships", "systems",
		"regions",
	)
}

func domainAssetTypeQuery() *huma.Param {
	return textQuery(
		"type", "Asset slot to clear, for example `banner` or `logo`.",
	)
}

func sdePublishedQuery() *huma.Param {
	return enumQuery(
		"published",
		"Send `false` to include unpublished rows. Anything else keeps "+
			"published rows only.",
		"true", "true", "false",
	)
}

func notificationParams() []*huma.Param {
	return []*huma.Param{
		limitQuery(50, 1, 100),
		intQuery("since", "Return replies with an ID above this one."),
	}
}

func walletParams() []*huma.Param {
	return []*huma.Param{
		walletPageQuery(),
		rangeQuery("division", "Corporation wallet division.", 1, 7),
	}
}

func walletPageQuery() *huma.Param {
	return pageQuery(1, 1, walletMaximumPage)
}

func sessionExceptQuery() *huma.Param {
	return textQuery(
		"except", "Session ID to keep. Every other session is revoked.",
	)
}

func loginParams() []*huma.Param {
	return []*huma.Param{
		textQuery(
			"returnTo",
			"Same-origin path to return to after login. `redirect` is "+
				"accepted as an alias.",
		),
		textQuery("redirect", "Alias for `returnTo`."),
		textQuery("delay", "Delay applied before the redirect, in seconds."),
		enumQuery(
			"charKm",
			"Send `0` to drop the character killmail scope from the "+
				"authorization request.",
			"1", "0", "1",
		),
		enumQuery(
			"corpKm",
			"Send `0` to drop the corporation killmail scope from the "+
				"authorization request.",
			"1", "0", "1",
		),
	}
}

func callbackParams() []*huma.Param {
	return []*huma.Param{
		textQuery("code", "Authorization code returned by EVE SSO."),
		textQuery("state", "Signed state issued when the flow started."),
		textQuery("error", "Error returned by EVE SSO, when it refuses."),
	}
}
