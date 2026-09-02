package api

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// publicOperationResponseSchema describes the compatibility payload, not the
// implementation that produced it. These handlers deliberately remain
// untyped so their validation and error responses can match the old API, but
// their successful responses should still produce useful OpenAPI and JSON
// Schema documents.
func publicOperationResponseSchema(operationID string) *huma.Schema {
	switch operationID {
	case "health":
		return responseSchema(map[string]*huma.Schema{
			"ok":        boolSchema(),
			"timestamp": timestampSchema(),
		}, "ok", "timestamp")
	case "history":
		return dataSchema(arraySchema(recordSchema(map[string]*huma.Schema{
			"date":  dateSchema(),
			"count": intSchema(),
		})))
	case "history-latest", "history-date":
		return dataSchema(&huma.Schema{
			Type:                 huma.TypeObject,
			AdditionalProperties: stringSchema(),
			Description:          "Map of killmail ID to killmail hash.",
		})
	case "resolve":
		return responseSchema(map[string]*huma.Schema{
			"type": stringSchema(),
			"results": arraySchema(recordSchema(map[string]*huma.Schema{
				"name":          stringSchema(),
				"id":            nullable(intSchema()),
				"resolved_name": nullable(stringSchema()),
			})),
			"resolved":   intSchema(),
			"unresolved": intSchema(),
		}, "type", "results", "resolved", "unresolved")
	case "search":
		return searchResponseSchema()
	case "location":
		return responseSchema(map[string]*huma.Schema{
			"system_id": numberSchema(),
			"x":         numberSchema(),
			"y":         numberSchema(),
			"z":         numberSchema(),
			"nearest": nullable(recordSchema(map[string]*huma.Schema{
				"item_id":   intSchema(),
				"item_name": stringSchema(),
				"type_id":   intSchema(),
				"group_id":  intSchema(),
				"distance":  numberSchema(),
			})),
		}, "system_id", "x", "y", "z", "nearest")
	case "killmails", "killmail-search":
		return cursorPageSchema(killlistRowSchema())
	case "killmails-count", "characters-count", "corporations-count", "alliances-count":
		return countResponseSchema()
	case "killmail-esi":
		return killmailESISchema()
	case "killmail-fitting":
		return killmailFittingSchema()
	case "killmail":
		return killmailDetailSchema()
	case "characters":
		return cursorPageSchema(characterListSchema())
	case "corporations":
		return cursorPageSchema(corporationListSchema())
	case "alliances":
		return cursorPageSchema(allianceListSchema())
	case "character":
		return entityDetailSchema("character", characterListSchema(),
			"corporationHistory")
	case "corporation":
		return entityDetailSchema("corporation", corporationListSchema(),
			"allianceHistory")
	case "alliance":
		return entityDetailSchema("alliance", allianceListSchema(), "")
	case "corporation-members", "alliance-members":
		return cursorPageSchema(characterListSchema())
	case "alliance-corporations":
		return cursorPageSchema(corporationListSchema())
	case "character-kills", "character-losses",
		"corporation-kills", "corporation-losses",
		"alliance-kills", "alliance-losses":
		return cursorPageSchema(killmailESISchema())
	case "character-stats", "corporation-stats-alltime",
		"corporation-stats-weekly", "alliance-stats-alltime",
		"alliance-stats-weekly":
		return entityStatsResponseSchema()
	case "character-intel":
		return characterIntelSchema()
	case "character-intel-batch":
		return responseSchema(map[string]*huma.Schema{
			"data":      arraySchema(characterIntelSchema()),
			"not_found": arraySchema(intSchema()),
			"days":      intSchema(),
		}, "data", "not_found", "days")
	case "character-analyze":
		return dataSchema(arraySchema(characterAnalysisSchema()))
	case "characters-batch-stats", "corporations-batch-stats",
		"alliances-batch-stats":
		return batchStatsSchema()
	case "coalition-stats":
		return coalitionStatsSchema()
	case "global-stats":
		return responseSchema(map[string]*huma.Schema{
			"entries": arraySchema(globalStatsEntrySchema()),
		}, "entries")
	case "ship-fittings":
		return shipFittingsSchema()
	case "battles", "corporation-battles", "alliance-battles":
		return battlePageSchema()
	case "battle":
		return battleDetailSchema()
	case "wars":
		return cursorPageSchema(warListSchema())
	case "war":
		return warDetailSchema()
	case "feed-index":
		return feedIndexSchema()
	case "feed-poll":
		return feedPollSchema()
	case "feed-status":
		return responseSchema(map[string]*huma.Schema{
			"status":    stringSchema(),
			"clients":   intSchema(),
			"latestSeq": nullable(intSchema()),
		}, "status", "clients", "latestSeq")
	}

	if strings.HasPrefix(operationID, "sde-") {
		return sdeResponseSchema(operationID)
	}
	return applicationOperationResponseSchema(operationID)
}

func responseSchema(
	properties map[string]*huma.Schema,
	required ...string,
) *huma.Schema {
	return &huma.Schema{
		Type:                 huma.TypeObject,
		Properties:           properties,
		Required:             required,
		AdditionalProperties: false,
	}
}

// recordSchema leaves additional properties open. It is used for SQL-backed
// records where the documented stable fields matter to clients but new
// columns may be added without making the schema reject otherwise-compatible
// payloads.
func recordSchema(properties map[string]*huma.Schema) *huma.Schema {
	return &huma.Schema{
		Type:                 huma.TypeObject,
		Properties:           properties,
		AdditionalProperties: true,
	}
}

func stringSchema() *huma.Schema {
	return &huma.Schema{Type: huma.TypeString}
}

func intSchema() *huma.Schema {
	return &huma.Schema{Type: huma.TypeInteger, Format: "int64"}
}

func numberSchema() *huma.Schema {
	return &huma.Schema{Type: huma.TypeNumber, Format: "double"}
}

func boolSchema() *huma.Schema {
	return &huma.Schema{Type: huma.TypeBoolean}
}

func timestampSchema() *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "date-time",
		Description: "UTC timestamp with millisecond precision.",
	}
}

func dateSchema() *huma.Schema {
	return &huma.Schema{Type: huma.TypeString, Format: "date"}
}

func nullable(schema *huma.Schema) *huma.Schema {
	copy := *schema
	copy.Nullable = true
	return &copy
}

func arraySchema(items *huma.Schema) *huma.Schema {
	return &huma.Schema{Type: huma.TypeArray, Items: items}
}

func dataSchema(data *huma.Schema) *huma.Schema {
	return responseSchema(map[string]*huma.Schema{"data": data}, "data")
}

func countResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{"count": intSchema()}, "count")
}

func cursorPageSchema(item *huma.Schema) *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"data": arraySchema(item),
		"pagination": responseSchema(map[string]*huma.Schema{
			"hasMore": boolSchema(),
			"cursor":  nullable(intSchema()),
		}, "hasMore", "cursor"),
	}, "data", "pagination")
}

func searchResponseSchema() *huma.Schema {
	hit := recordSchema(map[string]*huma.Schema{
		"id":                 stringSchema(),
		"name":               stringSchema(),
		"ticker":             nullable(stringSchema()),
		"type":               stringSchema(),
		"corporation_id":     nullable(intSchema()),
		"corporation_name":   stringSchema(),
		"corporation_ticker": stringSchema(),
		"alliance_id":        nullable(intSchema()),
		"alliance_name":      stringSchema(),
		"alliance_ticker":    stringSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"hits":             arraySchema(hit),
		"query":            stringSchema(),
		"processingTimeMs": intSchema(),
		"total":            intSchema(),
		"entityCounts": {
			Type:                 huma.TypeObject,
			AdditionalProperties: intSchema(),
		},
	}, "hits", "query", "processingTimeMs", "total", "entityCounts")
}

func killlistRowSchema() *huma.Schema {
	schema := recordSchema(map[string]*huma.Schema{
		"killmail_id":                 intSchema(),
		"killmail_hash":               stringSchema(),
		"killmail_time":               timestampSchema(),
		"total_value":                 numberSchema(),
		"attacker_count":              intSchema(),
		"is_npc":                      boolSchema(),
		"is_solo":                     boolSchema(),
		"ship_type_id":                nullable(intSchema()),
		"ship_name":                   nullable(stringSchema()),
		"ship_group_id":               nullable(intSchema()),
		"ship_group_name":             nullable(stringSchema()),
		"ship_market_path":            nullable(stringSchema()),
		"meta_group_id":               nullable(intSchema()),
		"victim_character_id":         nullable(intSchema()),
		"victim_character_name":       nullable(stringSchema()),
		"victim_corporation_id":       nullable(intSchema()),
		"victim_corporation_name":     nullable(stringSchema()),
		"victim_alliance_id":          nullable(intSchema()),
		"victim_alliance_name":        nullable(stringSchema()),
		"victim_faction_id":           nullable(intSchema()),
		"final_blow_character_id":     nullable(intSchema()),
		"final_blow_character_name":   nullable(stringSchema()),
		"final_blow_corporation_id":   nullable(intSchema()),
		"final_blow_corporation_name": nullable(stringSchema()),
		"final_blow_alliance_id":      nullable(intSchema()),
		"final_blow_alliance_name":    nullable(stringSchema()),
		"final_blow_ship_type_id":     nullable(intSchema()),
		"final_blow_ship_name":        nullable(stringSchema()),
		"solar_system_id":             intSchema(),
		"solar_system_name":           nullable(stringSchema()),
		"solar_system_security":       nullable(numberSchema()),
		"region_id":                   nullable(intSchema()),
		"region_name":                 nullable(stringSchema()),
	})
	schema.Required = []string{
		"killmail_id", "killmail_hash", "killmail_time", "total_value",
		"attacker_count", "is_npc", "is_solo", "ship_type_id",
		"ship_name", "ship_group_id", "ship_group_name", "ship_market_path", "meta_group_id",
		"victim_character_id", "victim_character_name",
		"victim_corporation_id", "victim_corporation_name",
		"victim_alliance_id", "victim_alliance_name",
		"victim_faction_id",
		"final_blow_character_id", "final_blow_character_name",
		"final_blow_corporation_id", "final_blow_corporation_name",
		"final_blow_alliance_id", "final_blow_alliance_name",
		"final_blow_ship_type_id", "final_blow_ship_name",
		"solar_system_id", "solar_system_name", "solar_system_security",
		"region_id", "region_name",
	}
	return schema
}

func killmailESISchema() *huma.Schema {
	item := recordSchema(map[string]*huma.Schema{
		"item_type_id":       intSchema(),
		"flag":               intSchema(),
		"quantity_dropped":   intSchema(),
		"quantity_destroyed": intSchema(),
		"singleton":          intSchema(),
	})
	victim := recordSchema(map[string]*huma.Schema{
		"damage_taken": intSchema(),
		"position": responseSchema(map[string]*huma.Schema{
			"x": numberSchema(), "y": numberSchema(), "z": numberSchema(),
		}, "x", "y", "z"),
		"character_id":   intSchema(),
		"corporation_id": intSchema(),
		"alliance_id":    intSchema(),
		"faction_id":     intSchema(),
		"ship_type_id":   intSchema(),
		"items":          arraySchema(item),
	})
	attacker := recordSchema(map[string]*huma.Schema{
		"damage_done":     intSchema(),
		"final_blow":      boolSchema(),
		"security_status": numberSchema(),
		"character_id":    intSchema(),
		"corporation_id":  intSchema(),
		"alliance_id":     intSchema(),
		"faction_id":      intSchema(),
		"ship_type_id":    intSchema(),
		"weapon_type_id":  intSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"killmail_id":     intSchema(),
		"killmail_hash":   stringSchema(),
		"killmail_time":   timestampSchema(),
		"solar_system_id": intSchema(),
		"war_id":          intSchema(),
		"victim":          victim,
		"attackers":       arraySchema(attacker),
	}, "killmail_id", "killmail_hash", "killmail_time", "solar_system_id",
		"victim", "attackers")
}

func killmailFittingSchema() *huma.Schema {
	item := responseSchema(map[string]*huma.Schema{
		"type_id":  intSchema(),
		"name":     stringSchema(),
		"quantity": intSchema(),
	}, "type_id", "name", "quantity")
	properties := map[string]*huma.Schema{
		"killmail_id": intSchema(),
		"ship": responseSchema(map[string]*huma.Schema{
			"type_id": intSchema(),
			"name":    stringSchema(),
		}, "type_id", "name"),
	}
	for _, slot := range []string{
		"subsystem", "high", "mid", "low", "rig", "service", "drone",
		"fighter", "cargo", "fleet", "fighter_bay", "specialized", "other",
	} {
		properties[slot] = arraySchema(item)
	}
	return responseSchema(properties, "killmail_id", "ship")
}

func killmailDetailSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"killmail_id":   intSchema(),
		"killmail_hash": stringSchema(),
		"killmail_time": timestampSchema(),
		"victim": recordSchema(map[string]*huma.Schema{
			"character_id":        nullable(intSchema()),
			"character_name":      nullable(stringSchema()),
			"corporation_id":      nullable(intSchema()),
			"corporation_name":    nullable(stringSchema()),
			"corporation_palette": nullable(stringSchema()),
			"alliance_id":         nullable(intSchema()),
			"alliance_name":       nullable(stringSchema()),
			"ship_type_id":        nullable(intSchema()),
			"ship_name":           nullable(stringSchema()),
			"ship_group_id":       nullable(intSchema()),
			"ship_group_name":     nullable(stringSchema()),
			"ship_market_path":    nullable(stringSchema()),
			"damage_taken":        intSchema(),
			"ship_price":          numberSchema(),
		}),
		"solar_system_id":       intSchema(),
		"solar_system_name":     nullable(stringSchema()),
		"solar_system_security": nullable(numberSchema()),
		"constellation_id":      nullable(intSchema()),
		"constellation_name":    nullable(stringSchema()),
		"region_id":             nullable(intSchema()),
		"region_name":           nullable(stringSchema()),
		"position_x":            nullable(numberSchema()),
		"position_y":            nullable(numberSchema()),
		"position_z":            nullable(numberSchema()),
		"location": nullable(recordSchema(map[string]*huma.Schema{
			"item_id": intSchema(), "item_name": stringSchema(),
			"type_id": intSchema(), "group_id": intSchema(),
			"distance": numberSchema(),
		})),
		"total_value":     numberSchema(),
		"fitted_value":    numberSchema(),
		"dropped_value":   numberSchema(),
		"destroyed_value": numberSchema(),
		"points":          numberSchema(),
		"attacker_count":  intSchema(),
		"is_npc":          boolSchema(),
		"is_solo":         boolSchema(),
		"total_damage":    intSchema(),
		"attackers": arraySchema(recordSchema(map[string]*huma.Schema{
			"character_id":   nullable(intSchema()),
			"corporation_id": nullable(intSchema()),
			"alliance_id":    nullable(intSchema()),
			"ship_type_id":   nullable(intSchema()),
			"weapon_type_id": nullable(intSchema()),
			"damage_done":    intSchema(),
			"final_blow":     boolSchema(),
		})),
		"items": arraySchema(recordSchema(map[string]*huma.Schema{
			"item_index":         intSchema(),
			"type_id":            intSchema(),
			"type_name":          nullable(stringSchema()),
			"quantity_dropped":   intSchema(),
			"quantity_destroyed": intSchema(),
			"slot":               stringSchema(),
			"price":              numberSchema(),
			"total_value":        numberSchema(),
		})),
		"siblings": arraySchema(recordSchema(map[string]*huma.Schema{
			"killmail_id":   intSchema(),
			"ship_type_id":  nullable(intSchema()),
			"ship_group_id": nullable(intSchema()),
			"ship_name":     nullable(stringSchema()),
			"total_value":   numberSchema(),
			"killmail_time": timestampSchema(),
		})),
	}, "killmail_id", "killmail_hash", "killmail_time", "victim",
		"solar_system_id", "total_value", "attackers", "items", "siblings")
}

func characterListSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"character_id":    intSchema(),
		"name":            stringSchema(),
		"corporation_id":  nullable(intSchema()),
		"alliance_id":     nullable(intSchema()),
		"faction_id":      nullable(intSchema()),
		"security_status": nullable(numberSchema()),
		"last_active":     nullable(timestampSchema()),
	})
}

func corporationListSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"corporation_id": intSchema(),
		"name":           stringSchema(),
		"ticker":         stringSchema(),
		"alliance_id":    nullable(intSchema()),
		"faction_id":     nullable(intSchema()),
		"member_count":   nullable(intSchema()),
		"date_founded":   nullable(timestampSchema()),
	})
}

func allianceListSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"alliance_id":       intSchema(),
		"name":              stringSchema(),
		"ticker":            stringSchema(),
		"faction_id":        nullable(intSchema()),
		"corporation_count": nullable(intSchema()),
		"member_count":      nullable(intSchema()),
		"date_founded":      nullable(timestampSchema()),
	})
}

func entityDetailSchema(
	key string,
	entity *huma.Schema,
	historyKey string,
) *huma.Schema {
	properties := map[string]*huma.Schema{
		key:           entity,
		"stats":       scalarStatsSchema(),
		"recentStats": recentStatsSchema(),
	}
	required := []string{key, "stats", "recentStats"}
	if key == "character" {
		properties["topShips"] = arraySchema(topShipSchema())
		properties["topSystems"] = arraySchema(topSystemSchema())
		required = append(required, "topShips", "topSystems")
	}
	if historyKey != "" {
		properties[historyKey] = arraySchema(recordSchema(map[string]*huma.Schema{
			"start_date": timestampSchema(),
			"kills":      intSchema(),
			"losses":     intSchema(),
		}))
		required = append(required, historyKey)
	}
	return responseSchema(properties, required...)
}

func scalarStatsSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"kills":          intSchema(),
		"losses":         intSchema(),
		"solo_kills":     intSchema(),
		"npc_losses":     intSchema(),
		"isk_destroyed":  numberSchema(),
		"isk_lost":       numberSchema(),
		"points":         numberSchema(),
		"final_blows":    intSchema(),
		"damage_dealt":   intSchema(),
		"damage_taken":   intSchema(),
		"efficiency":     numberSchema(),
		"isk_efficiency": numberSchema(),
	})
}

func recentStatsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
	}, "kills", "losses", "isk_destroyed", "isk_lost")
}

func topShipSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"ship_type_id": intSchema(),
		"ship_name":    stringSchema(),
		"kills":        intSchema(),
		"losses":       intSchema(),
	})
}

func topSystemSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"solar_system_id": intSchema(),
		"system_name":     stringSchema(),
		"kills":           intSchema(),
		"losses":          intSchema(),
	})
}

func entityStatsResponseSchema() *huma.Schema {
	schema := scalarStatsSchema()
	schema.Properties["character_id"] = intSchema()
	schema.Properties["corporation_id"] = intSchema()
	schema.Properties["alliance_id"] = intSchema()
	schema.Properties["period"] = stringSchema()
	schema.Properties["topMembers"] = arraySchema(recordSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(),
		"kills": intSchema(), "losses": intSchema(),
	}))
	schema.Properties["topShips"] = arraySchema(topShipSchema())
	schema.Properties["topSystems"] = arraySchema(topSystemSchema())
	return schema
}

func characterIntelSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"character_id":   intSchema(),
		"days":           intSchema(),
		"playstyle":      recordSchema(map[string]*huma.Schema{}),
		"dominant_style": stringSchema(),
		"tags":           arraySchema(stringSchema()),
		"fc": recordSchema(map[string]*huma.Schema{
			"likelihood": stringSchema(), "monitor_appearances": intSchema(),
		}),
		"capital_pilot":     boolSchema(),
		"is_logi":           boolSchema(),
		"ships_flown":       arraySchema(recordSchema(map[string]*huma.Schema{})),
		"ships_lost":        arraySchema(recordSchema(map[string]*huma.Schema{})),
		"targets":           arraySchema(recordSchema(map[string]*huma.Schema{})),
		"fleet_partners":    arraySchema(recordSchema(map[string]*huma.Schema{})),
		"groups_flown_with": arraySchema(recordSchema(map[string]*huma.Schema{})),
		"awox_kills":        intSchema(),
		"cyno_deaths":       intSchema(),
		"bait":              stringSchema(),
		"bait_count":        intSchema(),
		"bridge_score":      intSchema(),
	}, "character_id", "days", "playstyle", "dominant_style", "tags",
		"fc", "capital_pilot", "is_logi", "ships_flown", "ships_lost",
		"targets", "fleet_partners", "groups_flown_with", "awox_kills",
		"cyno_deaths", "bait", "bait_count", "bridge_score")
}

func characterAnalysisSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"character_id":      intSchema(),
		"total_kills":       intSchema(),
		"total_losses":      intSchema(),
		"efficiency":        numberSchema(),
		"gang_probability":  numberSchema(),
		"average_gang_size": numberSchema(),
		"last_5_ships": arraySchema(recordSchema(map[string]*huma.Schema{
			"ship_type_id": intSchema(),
			"ship_name":    nullable(stringSchema()),
			"kill_count":   intSchema(),
			"last_loss":    nullable(timestampSchema()),
		})),
		"cyno_probability": numberSchema(),
	}, "character_id", "total_kills", "total_losses", "efficiency",
		"gang_probability", "average_gang_size", "last_5_ships",
		"cyno_probability")
}

func batchStatsSchema() *huma.Schema {
	entry := scalarStatsSchema()
	entry.Properties["id"] = intSchema()
	entry.Properties["name"] = stringSchema()
	entry.Properties["topShips"] = arraySchema(topShipSchema())
	return responseSchema(map[string]*huma.Schema{
		"period":  stringSchema(),
		"results": arraySchema(entry),
	}, "period", "results")
}

func coalitionStatsSchema() *huma.Schema {
	side := recordSchema(map[string]*huma.Schema{
		"label":                stringSchema(),
		"entity_counts":        recordSchema(map[string]*huma.Schema{}),
		"overall":              scalarStatsSchema(),
		"vs_opponent":          scalarStatsSchema(),
		"top_ships_used":       arraySchema(topShipSchema()),
		"active_systems_count": intSchema(),
		"active_regions_count": intSchema(),
	})
	return responseSchema(map[string]*huma.Schema{
		"mode":        stringSchema(),
		"period_days": intSchema(),
		"from":        dateSchema(),
		"to":          dateSchema(),
		"sideA":       side,
		"sideB":       side,
		"clashed_systems": responseSchema(map[string]*huma.Schema{
			"count":      intSchema(),
			"system_ids": arraySchema(intSchema()),
		}, "count", "system_ids"),
		"daily": arraySchema(recordSchema(map[string]*huma.Schema{
			"date": dateSchema(),
		})),
	}, "mode", "period_days", "from", "to", "sideA", "sideB",
		"clashed_systems", "daily")
}

func globalStatsEntrySchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"id":            intSchema(),
		"name":          stringSchema(),
		"ticker":        stringSchema(),
		"sec":           numberSchema(),
		"kills":         intSchema(),
		"losses":        intSchema(),
		"isk_destroyed": numberSchema(),
		"isk_lost":      numberSchema(),
		"efficiency":    numberSchema(),
		"killmail_id":   intSchema(),
		"killmail_hash": stringSchema(),
		"killmail_time": timestampSchema(),
		"total_value":   numberSchema(),
	})
}

func shipFittingsSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"ship_type_id":  intSchema(),
		"window_days":   intSchema(),
		"module_filter": arraySchema(intSchema()),
		"is_rare_hull":  boolSchema(),
		"hull_cost":     numberSchema(),
		"families": arraySchema(recordSchema(map[string]*huma.Schema{
			"family_hash":        stringSchema(),
			"canonical_fit_hash": stringSchema(),
			"total_uses":         intSchema(),
			"canonical_uses":     intSchema(),
			"variant_count":      intSchema(),
			"last_used":          timestampSchema(),
			"fit_cost":           numberSchema(),
			"modules":            arraySchema(recordSchema(map[string]*huma.Schema{})),
			"top_alliances":      arraySchema(recordSchema(map[string]*huma.Schema{})),
			"context":            fittingFamilyContextSchema(),
		})),
	}, "ship_type_id", "window_days", "module_filter", "is_rare_hull",
		"hull_cost", "families")
}

func battleSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"battle_id":            intSchema(),
		"solar_system_id":      intSchema(),
		"system_name":          nullable(stringSchema()),
		"system_security":      nullable(numberSchema()),
		"region_id":            nullable(intSchema()),
		"region_name":          nullable(stringSchema()),
		"start_time":           timestampSchema(),
		"end_time":             timestampSchema(),
		"duration_minutes":     numberSchema(),
		"kill_count":           intSchema(),
		"total_isk_destroyed":  numberSchema(),
		"is_multi_party":       boolSchema(),
		"is_custom":            boolSchema(),
		"entity_kills":         stringSchema(),
		"entity_losses":        stringSchema(),
		"entity_isk_destroyed": numberSchema(),
	})
}

func battlePageSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"data": arraySchema(battleSchema()),
		"pagination": responseSchema(map[string]*huma.Schema{
			"page": intSchema(), "limit": intSchema(), "hasMore": boolSchema(),
		}, "page", "limit", "hasMore"),
	}, "data", "pagination")
}

func battleDetailSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"battle": battleSchema(),
		"teams": arraySchema(recordSchema(map[string]*huma.Schema{
			"team_index":          intSchema(),
			"total_kills":         intSchema(),
			"total_losses":        intSchema(),
			"total_isk_destroyed": numberSchema(),
			"total_isk_lost":      numberSchema(),
			"members": arraySchema(recordSchema(map[string]*huma.Schema{
				"corporation_id": intSchema(),
				"alliance_id":    nullable(intSchema()),
				"kills":          intSchema(),
				"losses":         intSchema(),
				"isk_destroyed":  numberSchema(),
				"isk_lost":       numberSchema(),
			})),
		})),
	}, "battle", "teams")
}

func warListSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"war_id":                   intSchema(),
		"declared":                 timestampSchema(),
		"started":                  nullable(timestampSchema()),
		"finished":                 nullable(timestampSchema()),
		"mutual":                   boolSchema(),
		"aggressor_alliance_id":    nullable(intSchema()),
		"aggressor_corporation_id": nullable(intSchema()),
		"defender_alliance_id":     nullable(intSchema()),
		"defender_corporation_id":  nullable(intSchema()),
		"aggressor_ships_killed":   intSchema(),
		"defender_ships_killed":    intSchema(),
	})
}

func warDetailSchema() *huma.Schema {
	entity := responseSchema(map[string]*huma.Schema{
		"id": intSchema(), "name": stringSchema(), "ticker": stringSchema(),
		"type": stringSchema(), "isk_destroyed": numberSchema(),
		"ships_killed": intSchema(),
	}, "id", "name", "ticker", "type", "isk_destroyed", "ships_killed")
	return responseSchema(map[string]*huma.Schema{
		"war": responseSchema(map[string]*huma.Schema{
			"war_id":          intSchema(),
			"declared":        timestampSchema(),
			"started":         nullable(timestampSchema()),
			"finished":        nullable(timestampSchema()),
			"retracted":       nullable(timestampSchema()),
			"mutual":          boolSchema(),
			"open_for_allies": boolSchema(),
			"aggressor":       entity,
			"defender":        entity,
			"allies": arraySchema(responseSchema(map[string]*huma.Schema{
				"id": intSchema(), "name": stringSchema(), "type": stringSchema(),
			}, "id", "name", "type")),
		}, "war_id", "declared", "started", "finished", "retracted", "mutual",
			"open_for_allies", "aggressor", "defender", "allies"),
		"stats": responseSchema(map[string]*huma.Schema{
			"total_kills": intSchema(),
			"total_value": numberSchema(),
			"top_ships":   arraySchema(topShipSchema()),
		}, "total_kills", "total_value", "top_ships"),
	}, "war", "stats")
}

func feedIndexSchema() *huma.Schema {
	stringMap := &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: stringSchema(),
	}
	return responseSchema(map[string]*huma.Schema{
		"name":        stringSchema(),
		"description": stringSchema(),
		"endpoints": {
			Type: huma.TypeObject,
			AdditionalProperties: recordSchema(map[string]*huma.Schema{
				"description": stringSchema(),
				"params":      stringMap,
				"headers":     stringMap,
				"example":     stringSchema(),
			}),
		},
		"topics": {
			Type:                 huma.TypeObject,
			AdditionalProperties: arraySchema(stringSchema()),
		},
		"note": stringSchema(),
	}, "name", "description", "endpoints", "topics", "note")
}

func feedPollSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"data": arraySchema(responseSchema(map[string]*huma.Schema{
			"seq":           intSchema(),
			"killmail_id":   intSchema(),
			"killmail_hash": stringSchema(),
			"data":          killmailESISchema(),
		}, "seq", "killmail_id", "killmail_hash", "data")),
		"latest":  nullable(intSchema()),
		"hasMore": boolSchema(),
		"next":    stringSchema(),
		"last":    stringSchema(),
	}, "data", "latest", "hasMore", "next", "last")
}

func sdeResponseSchema(operationID string) *huma.Schema {
	switch operationID {
	case "sde-systems":
		return cursorPageSchema(sdeSystemSchema())
	case "sde-system":
		return sdeSystemSchema()
	case "sde-regions":
		return dataSchema(arraySchema(sdeRegionSchema()))
	case "sde-region":
		return sdeRegionSchema()
	case "sde-constellations":
		return dataSchema(arraySchema(sdeConstellationSchema()))
	case "sde-constellation":
		return sdeConstellationSchema()
	case "sde-types":
		return cursorPageSchema(sdeTypeSchema())
	case "sde-type":
		return sdeTypeSchema()
	case "sde-groups":
		return cursorPageSchema(sdeGroupSchema())
	case "sde-group":
		return sdeGroupSchema()
	case "sde-categories", "sde-market-groups", "sde-meta-groups",
		"sde-flags", "sde-factions", "sde-races", "sde-bloodlines",
		"sde-npc-corporations", "sde-station-operations",
		"sde-sovereignty":
		return dataSchema(arraySchema(sdeGenericRecordSchema(operationID)))
	case "sde-category", "sde-market-group", "sde-faction", "sde-race",
		"sde-bloodline", "sde-celestial", "sde-npc-corporation",
		"sde-station-operation", "sde-sovereignty-system":
		return sdeGenericRecordSchema(operationID)
	case "sde-type-dogma":
		return responseSchema(map[string]*huma.Schema{
			"type_id":    intSchema(),
			"attributes": arraySchema(sdeGenericRecordSchema("dogma-attribute")),
			"effects":    arraySchema(sdeGenericRecordSchema("dogma-effect")),
		}, "type_id", "attributes", "effects")
	case "sde-type-materials":
		return nestedSDESchema("type_id", "materials", "material")
	case "sde-type-insurance":
		return nestedSDESchema("type_id", "levels", "insurance")
	case "sde-system-jumps":
		return nestedSDESchema("solar_system_id", "jumps", "jump")
	case "sde-system-celestials":
		return nestedSDESchema("solar_system_id", "celestials", "celestial")
	case "sde-sovereignty-history":
		return nestedSDESchema("system_id", "history", "sovereignty-history")
	case "sde-stations":
		return cursorPageSchema(sdeStationSchema())
	case "sde-station":
		return sdeStationSchema()
	case "sde-structures":
		return cursorPageSchema(sdeStructureSchema())
	case "sde-structure":
		return sdeStructureSchema()
	case "sde-prices":
		return responseSchema(map[string]*huma.Schema{
			"type_id":   intSchema(),
			"region_id": numberSchema(),
			"prices": arraySchema(recordSchema(map[string]*huma.Schema{
				"type_id": intSchema(), "region_id": intSchema(),
				"date": dateSchema(), "average": numberSchema(),
				"highest": numberSchema(), "lowest": numberSchema(),
				"order_count": intSchema(), "volume": intSchema(),
			})),
		}, "type_id", "region_id", "prices")
	case "sde-custom-prices":
		return responseSchema(map[string]*huma.Schema{
			"data": arraySchema(recordSchema(map[string]*huma.Schema{
				"type_id": intSchema(), "type_name": nullable(stringSchema()),
				"valid_until": dateSchema(), "price": numberSchema(),
			})),
			"count": intSchema(),
		}, "data", "count")
	case "sde-system-kills", "sde-region-kills":
		return cursorPageSchema(killlistRowSchema())
	}
	return nil
}

func sdeSystemSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"solar_system_id":    intSchema(),
		"system_name":        stringSchema(),
		"constellation_id":   intSchema(),
		"constellation_name": nullable(stringSchema()),
		"region_id":          intSchema(),
		"region_name":        nullable(stringSchema()),
		"security":           numberSchema(),
		"security_class":     nullable(stringSchema()),
	})
}

func sdeRegionSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"region_id":    intSchema(),
		"name":         stringSchema(),
		"description":  nullable(stringSchema()),
		"faction_id":   nullable(intSchema()),
		"faction_name": nullable(stringSchema()),
	})
}

func sdeConstellationSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"constellation_id":   intSchema(),
		"constellation_name": stringSchema(),
		"region_id":          intSchema(),
		"region_name":        nullable(stringSchema()),
		"faction_id":         nullable(intSchema()),
	})
}

func sdeTypeSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"type_id":         intSchema(),
		"name":            stringSchema(),
		"description":     nullable(stringSchema()),
		"group_id":        intSchema(),
		"group_name":      nullable(stringSchema()),
		"category_id":     intSchema(),
		"category_name":   nullable(stringSchema()),
		"meta_group_id":   nullable(intSchema()),
		"market_group_id": nullable(intSchema()),
		"mass":            nullable(numberSchema()),
		"volume":          nullable(numberSchema()),
		"capacity":        nullable(numberSchema()),
		"base_price":      nullable(numberSchema()),
		"published":       boolSchema(),
	})
}

func sdeGroupSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"group_id":      intSchema(),
		"name":          stringSchema(),
		"category_id":   intSchema(),
		"category_name": nullable(stringSchema()),
		"published":     boolSchema(),
		"icon_id":       nullable(intSchema()),
	})
}

func sdeStationSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"station_id":      intSchema(),
		"station_name":    stringSchema(),
		"type_id":         intSchema(),
		"corporation_id":  intSchema(),
		"solar_system_id": intSchema(),
		"region_id":       intSchema(),
		"security":        numberSchema(),
	})
}

func sdeStructureSchema() *huma.Schema {
	return recordSchema(map[string]*huma.Schema{
		"structure_id":    intSchema(),
		"name":            stringSchema(),
		"owner_id":        intSchema(),
		"solar_system_id": intSchema(),
		"region_id":       intSchema(),
		"type_id":         intSchema(),
	})
}

func nestedSDESchema(idKey, rowsKey, recordType string) *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		idKey:   intSchema(),
		rowsKey: arraySchema(sdeGenericRecordSchema(recordType)),
	}, idKey, rowsKey)
}

func sdeGenericRecordSchema(operationID string) *huma.Schema {
	properties := map[string]*huma.Schema{
		"id":          intSchema(),
		"name":        stringSchema(),
		"description": nullable(stringSchema()),
	}
	for _, key := range []string{
		"category_id", "market_group_id", "meta_group_id", "flag_id",
		"faction_id", "race_id", "bloodline_id", "corporation_id",
		"item_id", "type_id", "system_id", "solar_system_id",
		"alliance_id", "attribute_id", "effect_id", "material_type_id",
	} {
		properties[key] = nullable(intSchema())
	}
	properties["published"] = boolSchema()
	properties["date_added"] = timestampSchema()
	properties["operation"] = stringSchema()
	return &huma.Schema{
		Type:  huma.TypeObject,
		Title: strings.TrimPrefix(operationID, "sde-"),
		Description: "Static data record. Known stable fields are documented; " +
			"source-table-specific fields are preserved as additional properties.",
		Properties:           properties,
		AdditionalProperties: true,
	}
}
