package api

import "github.com/danielgtaylor/huma/v2"

func roamPilotResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"character_id": intSchema(), "name": stringSchema(),
		"corporation_id": intSchema(), "corporation_name": stringSchema(),
		"corporation_ticker": stringSchema(), "alliance_id": intSchema(),
		"alliance_name": stringSchema(), "alliance_ticker": stringSchema(),
		"kill_participations": intSchema(), "final_blows": intSchema(),
		"losses": intSchema(), "engagements": intSchema(), "damage_done": intSchema(),
	}, "character_id", "name", "corporation_id", "corporation_name",
		"corporation_ticker", "alliance_id", "alliance_name", "alliance_ticker",
		"kill_participations", "final_blows", "losses", "engagements", "damage_done")
}

func roamKillmailResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"killmail_id": intSchema(), "killmail_time": timestampSchema(),
		"solar_system_id": intSchema(), "solar_system_name": stringSchema(),
		"region_id": intSchema(), "region_name": stringSchema(),
		"victim_character_id": intSchema(), "victim_name": stringSchema(),
		"victim_corporation_id": intSchema(), "victim_corporation_name": stringSchema(),
		"victim_alliance_id": intSchema(), "victim_alliance_name": stringSchema(),
		"victim_ship_type_id": intSchema(), "victim_ship_name": stringSchema(),
		"total_value": numberSchema(), "attacker_count": intSchema(),
		"role": stringSchema(), "friendly_fire": boolSchema(),
		"fleet_attacker_ids": arraySchema(intSchema()), "fleet_final_blow": boolSchema(),
		"final_blow_character_name": stringSchema(), "final_blow_corporation_name": stringSchema(),
	}, "killmail_id", "killmail_time", "solar_system_id", "solar_system_name",
		"region_id", "region_name", "victim_character_id", "victim_name",
		"victim_corporation_id", "victim_corporation_name", "victim_alliance_id",
		"victim_alliance_name", "victim_ship_type_id", "victim_ship_name",
		"total_value", "attacker_count", "role", "friendly_fire",
		"fleet_attacker_ids", "fleet_final_blow", "final_blow_character_name",
		"final_blow_corporation_name")
}

func roamEngagementResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"number": intSchema(), "solar_system_id": intSchema(),
		"solar_system_name": stringSchema(), "region_id": intSchema(),
		"region_name": stringSchema(), "start_time": timestampSchema(),
		"end_time": timestampSchema(), "kills": intSchema(), "losses": intSchema(),
		"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
		"pilot_ids": arraySchema(intSchema()),
		"killmails": arraySchema(roamKillmailResponseSchema()),
	}, "number", "solar_system_id", "solar_system_name", "region_id", "region_name",
		"start_time", "end_time", "kills", "losses", "isk_destroyed", "isk_lost",
		"pilot_ids", "killmails")
}

func roamReportResponseSchema() *huma.Schema {
	return responseSchema(map[string]*huma.Schema{
		"id": stringSchema(), "created_at": timestampSchema(),
		"start_time": timestampSchema(), "end_time": timestampSchema(),
		"input_count": intSchema(), "roster": arraySchema(roamPilotResponseSchema()),
		"unresolved": arraySchema(stringSchema()), "truncated": boolSchema(),
		"summary": responseSchema(map[string]*huma.Schema{
			"kills": intSchema(), "losses": intSchema(),
			"friendly_fire_losses": intSchema(), "isk_destroyed": numberSchema(),
			"isk_lost": numberSchema(), "efficiency": numberSchema(),
			"engagements": intSchema(), "systems": intSchema(), "regions": intSchema(),
			"active_pilots": intSchema(), "coordinated_kills": intSchema(),
			"unique_character_targets": intSchema(),
		}, "kills", "losses", "friendly_fire_losses", "isk_destroyed", "isk_lost",
			"efficiency", "engagements", "systems", "regions", "active_pilots",
			"coordinated_kills", "unique_character_targets"),
		"systems": arraySchema(responseSchema(map[string]*huma.Schema{
			"solar_system_id": intSchema(), "solar_system_name": stringSchema(),
			"region_id": intSchema(), "region_name": stringSchema(),
			"first_seen": timestampSchema(), "last_seen": timestampSchema(),
			"engagements": intSchema(), "kills": intSchema(), "losses": intSchema(),
			"isk_destroyed": numberSchema(), "isk_lost": numberSchema(),
		}, "solar_system_id", "solar_system_name", "region_id", "region_name",
			"first_seen", "last_seen", "engagements", "kills", "losses",
			"isk_destroyed", "isk_lost")),
		"targets": arraySchema(responseSchema(map[string]*huma.Schema{
			"corporation_id": intSchema(), "corporation_name": stringSchema(),
			"alliance_id": intSchema(), "alliance_name": stringSchema(),
			"kills": intSchema(), "isk_destroyed": numberSchema(),
		}, "corporation_id", "corporation_name", "alliance_id", "alliance_name",
			"kills", "isk_destroyed")),
		"engagements": arraySchema(roamEngagementResponseSchema()),
	}, "id", "created_at", "start_time", "end_time", "input_count", "roster",
		"unresolved", "truncated", "summary", "systems", "targets", "engagements")
}
