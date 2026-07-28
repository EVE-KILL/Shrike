package api

import "github.com/danielgtaylor/huma/v2"

func campaignCreatorSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "name": nullable(stringSchema()),
	}, "character_id", "name")
}

func campaignEntitySchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"entity_type": intSchema(), "entity_id": intSchema(),
		"name": nullable(stringSchema()), "kills": intSchema(),
		"losses": intSchema(), "isk_destroyed": numberSchema(),
		"isk_lost": numberSchema(),
	}, "entity_type", "entity_id")
}

func campaignSideSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"side_index": intSchema(), "name": stringSchema(),
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
		"palette":  nullable(stringSchema()),
		"entities": arraySchema(campaignEntitySchema()),
	}, "side_index", "name", "kills", "losses", "isk_destroyed",
		"isk_lost", "palette", "entities")
}

func campaignTotalsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"killCount": intSchema(), "iskDestroyed": numberSchema(),
		"charactersInvolved": intSchema(), "corporationsInvolved": intSchema(),
		"alliancesInvolved": intSchema(),
	}, "killCount", "iskDestroyed", "charactersInvolved",
		"corporationsInvolved", "alliancesInvolved")
}

func campaignLocationSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"systemIds":        arraySchema(intSchema()),
		"constellationIds": arraySchema(intSchema()),
		"regionIds":        arraySchema(intSchema()),
	})
}

func campaignCardSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"campaign_id": stringSchema(), "mode": stringSchema(),
		"name": stringSchema(), "description": nullable(stringSchema()),
		"status": intSchema(), "visibility": intSchema(),
		"start_time": timestampSchema(), "end_time": nullable(timestampSchema()),
		"location":         nullable(campaignLocationSchema()),
		"last_activity_at": nullable(timestampSchema()),
		"stats_updated_at": nullable(timestampSchema()),
		"created_at":       timestampSchema(), "totals": nullable(campaignTotalsSchema()),
		"sides": arraySchema(campaignSideSchema()), "creator": campaignCreatorSchema(),
	}, "campaign_id", "mode", "name", "description", "status",
		"visibility", "start_time", "end_time", "location",
		"last_activity_at", "stats_updated_at", "created_at", "totals",
		"sides", "creator")
}

func campaignListResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"campaigns": arraySchema(campaignCardSchema()), "hasMore": boolSchema(),
		"page": intSchema(), "total": intSchema(),
		"counts": responseSchema(map[string]*huma.Schema{
			"active": intSchema(), "archived": intSchema(), "private": intSchema(),
		}, "active", "archived", "private"),
	}, "campaigns", "hasMore", "page", "total", "counts")
}

func campaignStatsPilotSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"characterId": intSchema(), "name": nullable(stringSchema()),
		"kills": intSchema(), "losses": intSchema(),
		"iskDestroyed": numberSchema(), "iskLost": numberSchema(),
		"corporationId":     nullable(intSchema()),
		"corporationName":   nullable(stringSchema()),
		"corporationTicker": nullable(stringSchema()),
		"allianceId":        nullable(intSchema()), "allianceName": nullable(stringSchema()),
		"allianceTicker": nullable(stringSchema()),
	}, "characterId", "name")
}

func campaignStatsSchema() *huma.Schema {
	shipClass := responseSchema(map[string]*huma.Schema{
		"groupId": intSchema(), "name": nullable(stringSchema()),
		"losses": intSchema(), "iskLost": numberSchema(),
	}, "groupId", "name", "losses", "iskLost")
	topSystem := responseSchema(map[string]*huma.Schema{
		"systemId": intSchema(), "name": nullable(stringSchema()),
		"regionId": nullable(intSchema()), "regionName": nullable(stringSchema()),
		"kills": intSchema(), "iskDestroyed": numberSchema(),
	}, "systemId", "name", "regionId", "regionName", "kills", "iskDestroyed")
	mostValuable := responseSchema(map[string]*huma.Schema{
		"killmailId": intSchema(), "value": numberSchema(),
		"shipTypeId": nullable(intSchema()), "shipName": nullable(stringSchema()),
		"victimCharacterId":     nullable(intSchema()),
		"victimCharacterName":   nullable(stringSchema()),
		"victimCorporationId":   nullable(intSchema()),
		"victimCorporationName": nullable(stringSchema()),
		"victimSide":            nullable(intSchema()), "killmailTime": timestampSchema(),
	}, "killmailId", "value", "shipTypeId", "shipName",
		"victimCharacterId", "victimCharacterName", "victimCorporationId",
		"victimCorporationName", "victimSide", "killmailTime")
	intelPilot := responseSchema(map[string]*huma.Schema{
		"characterId": intSchema(), "name": nullable(stringSchema()),
		"corporationName": nullable(stringSchema()),
		"allianceName":    nullable(stringSchema()),
		"shipTypeId":      nullable(intSchema()), "shipName": nullable(stringSchema()),
		"shipGroupName": nullable(stringSchema()), "damage": intSchema(),
		"died": boolSchema(),
	}, "characterId", "name", "corporationName", "allianceName",
		"shipTypeId", "shipName", "shipGroupName", "damage", "died")
	intel := responseSchema(map[string]*huma.Schema{
		"fcs": arraySchema(intelPilot), "logistics": arraySchema(intelPilot),
		"logisticsCount": intSchema(), "capitals": arraySchema(intelPilot),
		"capitalsCount": intSchema(),
	}, "fcs", "logistics", "logisticsCount", "capitals", "capitalsCount")
	return responseSchema(map[string]*huma.Schema{
		"totals":             campaignTotalsSchema(),
		"topKillersBySide":   mapOfSchema(arraySchema(campaignStatsPilotSchema())),
		"topVictimsBySide":   mapOfSchema(arraySchema(campaignStatsPilotSchema())),
		"shipClassesBySide":  mapOfSchema(arraySchema(shipClass)),
		"topKillersOverall":  arraySchema(campaignStatsPilotSchema()),
		"topVictimsOverall":  arraySchema(campaignStatsPilotSchema()),
		"shipClassesOverall": arraySchema(shipClass),
		"topSystems":         arraySchema(topSystem),
		"intelBySide":        mapOfSchema(nullable(intel)),
		"mostValuable":       arraySchema(mostValuable),
	}, "totals", "topKillersBySide", "topVictimsBySide",
		"shipClassesBySide", "topKillersOverall", "topVictimsOverall",
		"shipClassesOverall", "topSystems", "intelBySide", "mostValuable")
}

func campaignPrizePoolSchema() *huma.Schema {
	result := responseSchema(map[string]*huma.Schema{
		"rank": intSchema(), "character_id": intSchema(),
		"character_name": stringSchema(), "metric_value": stringSchema(),
		"secondary_value": stringSchema(), "payout_percentage": intSchema(),
		"payout_amount": numberSchema(), "claimed": boolSchema(),
		"paid": boolSchema(), "can_claim": boolSchema(),
	}, "rank", "character_id", "character_name", "metric_value",
		"secondary_value", "payout_percentage", "payout_amount",
		"claimed", "paid", "can_claim")
	contribution := responseSchema(map[string]*huma.Schema{
		"id": stringSchema(), "source": stringSchema(),
		"contributor_id": nullable(intSchema()), "contributor_name": stringSchema(),
		"contributor_type": stringSchema(), "amount": stringSchema(),
		"contributed_at": timestampSchema(),
	}, "id", "source", "contributor_id", "contributor_name",
		"contributor_type", "amount", "contributed_at")
	return responseSchema(map[string]*huma.Schema{
		"metric": intSchema(), "metric_label": stringSchema(),
		"winner_count": intSchema(), "payout_percentages": arraySchema(intSchema()),
		"status": intSchema(), "funded_total": stringSchema(),
		"contribution_count": intSchema(), "contributions": arraySchema(contribution),
		"rules_locked_at": nullable(timestampSchema()),
		"finalized_at":    nullable(timestampSchema()), "funding_reference": stringSchema(),
		"funding_closes_at": nullable(timestampSchema()),
		"settles_at":        nullable(timestampSchema()),
		"last_wallet_sync":  nullable(timestampSchema()), "discord_url": stringSchema(),
		"projected_lead_percent": nullable(numberSchema()),
		"results":                arraySchema(result),
	}, "metric", "metric_label", "winner_count", "payout_percentages",
		"status", "funded_total", "contribution_count", "contributions",
		"rules_locked_at", "finalized_at", "funding_reference",
		"funding_closes_at", "settles_at", "last_wallet_sync", "discord_url",
		"projected_lead_percent", "results")
}

func campaignDetailResponseSchema() *huma.Schema {
	locationEntry := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(),
	}, "id", "name")
	dailyRow := responseSchema(map[string]*huma.Schema{
		"period": dateSchema(), "side_index": intSchema(), "kills": intSchema(),
		"losses": intSchema(), "isk_destroyed": numberSchema(),
		"isk_lost": numberSchema(),
	}, "period", "side_index", "kills", "losses", "isk_destroyed", "isk_lost")
	allowed := responseSchema(map[string]*huma.Schema{
		"type": stringSchema(), "id": intSchema(), "name": stringSchema(),
	}, "type", "id")
	return responseSchema(map[string]*huma.Schema{
		"campaign_id": stringSchema(), "mode": stringSchema(),
		"name": stringSchema(), "description": nullable(stringSchema()),
		"status": intSchema(), "visibility": intSchema(),
		"allowed_entities": arraySchema(allowed), "start_time": timestampSchema(),
		"end_time": nullable(timestampSchema()),
		"location": nullable(campaignLocationSchema()),
		"location_details": responseSchema(map[string]*huma.Schema{
			"systems":        arraySchema(locationEntry),
			"constellations": arraySchema(locationEntry),
			"regions":        arraySchema(locationEntry),
		}, "systems", "constellations", "regions"),
		"stats":             nullable(campaignStatsSchema()),
		"processed_through": nullable(timestampSchema()),
		"last_activity_at":  nullable(timestampSchema()),
		"stats_updated_at":  nullable(timestampSchema()),
		"processing": responseSchema(map[string]*huma.Schema{
			"paused": boolSchema(), "note": nullable(stringSchema()),
			"estimated_killmails": nullable(intSchema()),
			"last_started_at":     nullable(timestampSchema()),
			"last_duration_ms":    nullable(intSchema()),
			"last_killmails":      nullable(intSchema()),
			"last_error":          nullable(stringSchema()),
		}, "paused", "note", "estimated_killmails", "last_started_at",
			"last_duration_ms", "last_killmails", "last_error"),
		"created_at": timestampSchema(), "creator": campaignCreatorSchema(),
		"prize_pool": nullable(campaignPrizePoolSchema()),
		"sides":      arraySchema(campaignSideSchema()),
		"daily": responseSchema(map[string]*huma.Schema{
			"granularity": stringSchema(), "rows": arraySchema(dailyRow),
		}, "granularity", "rows"),
	}, "campaign_id", "mode", "name", "description", "status",
		"visibility", "allowed_entities", "start_time", "end_time",
		"location", "location_details", "stats", "processed_through",
		"last_activity_at", "stats_updated_at", "processing", "created_at",
		"creator", "prize_pool", "sides", "daily")
}

func campaignAdminListSchema() *huma.Schema {
	row := responseSchema(map[string]*huma.Schema{
		"campaign_id": stringSchema(), "name": stringSchema(),
		"status": intSchema(), "visibility": intSchema(),
		"created_by_character_id": intSchema(), "creator_name": nullable(stringSchema()),
		"start_time": timestampSchema(), "end_time": nullable(timestampSchema()),
		"created_at": timestampSchema(), "updated_at": timestampSchema(),
		"processing_paused": boolSchema(), "processing_note": nullable(stringSchema()),
		"estimated_killmails":         intSchema(),
		"last_processing_started_at":  nullable(timestampSchema()),
		"last_processing_duration_ms": nullable(intSchema()),
		"last_processing_killmails":   nullable(intSchema()),
		"last_processing_error":       nullable(stringSchema()),
		"totals":                      nullable(campaignTotalsSchema()),
	}, "campaign_id", "name", "status", "visibility",
		"created_by_character_id", "creator_name", "start_time", "end_time",
		"created_at", "updated_at", "processing_paused", "processing_note",
		"estimated_killmails", "last_processing_started_at",
		"last_processing_duration_ms", "last_processing_killmails",
		"last_processing_error", "totals")
	return responseSchema(map[string]*huma.Schema{
		"campaigns": arraySchema(row), "page": intSchema(), "hasMore": boolSchema(),
	}, "campaigns", "page", "hasMore")
}
