package mcpserver

import (
	"context"
	"fmt"
	"time"
)

type KillmailInput struct {
	KillmailID int64 `json:"killmail_id" minimum:"1" jsonschema:"Numeric killmail identifier."`
}

type KillmailLocation struct {
	ID                int64    `json:"id"`
	Name              *string  `json:"name"`
	Security          *float64 `json:"security"`
	ConstellationID   *int64   `json:"constellation_id"`
	ConstellationName *string  `json:"constellation_name"`
	RegionID          *int64   `json:"region_id"`
	RegionName        *string  `json:"region_name"`
}

type KillmailVictim struct {
	CharacterID       *int64  `json:"character_id"`
	CharacterName     *string `json:"character_name"`
	CorporationID     *int64  `json:"corporation_id"`
	CorporationName   *string `json:"corporation_name"`
	CorporationTicker *string `json:"corporation_ticker"`
	AllianceID        *int64  `json:"alliance_id"`
	AllianceName      *string `json:"alliance_name"`
	AllianceTicker    *string `json:"alliance_ticker"`
	FactionID         *int64  `json:"faction_id"`
	FactionName       *string `json:"faction_name"`
	ShipTypeID        *int64  `json:"ship_type_id"`
	ShipName          *string `json:"ship_name"`
	ShipGroupID       *int64  `json:"ship_group_id"`
	ShipGroupName     *string `json:"ship_group_name"`
	DamageTaken       int64   `json:"damage_taken"`
}

type KillmailAttacker struct {
	CharacterID       *int64   `json:"character_id"`
	CharacterName     *string  `json:"character_name"`
	CorporationID     *int64   `json:"corporation_id"`
	CorporationName   *string  `json:"corporation_name"`
	CorporationTicker *string  `json:"corporation_ticker"`
	AllianceID        *int64   `json:"alliance_id"`
	AllianceName      *string  `json:"alliance_name"`
	AllianceTicker    *string  `json:"alliance_ticker"`
	FactionID         *int64   `json:"faction_id"`
	FactionName       *string  `json:"faction_name"`
	ShipTypeID        *int64   `json:"ship_type_id"`
	ShipName          *string  `json:"ship_name"`
	WeaponTypeID      *int64   `json:"weapon_type_id"`
	WeaponName        *string  `json:"weapon_name"`
	DamageDone        int64    `json:"damage_done"`
	FinalBlow         bool     `json:"final_blow"`
	SecurityStatus    *float64 `json:"security_status"`
}

type KillmailOutput struct {
	KillmailID     int64              `json:"killmail_id"`
	URL            string             `json:"url"`
	Time           time.Time          `json:"time"`
	Hash           string             `json:"hash"`
	TotalValue     float64            `json:"total_value"`
	FittedValue    float64            `json:"fitted_value"`
	DroppedValue   float64            `json:"dropped_value"`
	DestroyedValue float64            `json:"destroyed_value"`
	Points         int64              `json:"points"`
	AttackerCount  int64              `json:"attacker_count"`
	IsNPC          bool               `json:"is_npc"`
	IsSolo         bool               `json:"is_solo"`
	WarID          *int64             `json:"war_id"`
	System         KillmailLocation   `json:"system"`
	Victim         KillmailVictim     `json:"victim"`
	FinalBlow      *KillmailAttacker  `json:"final_blow"`
	Attackers      []KillmailAttacker `json:"attackers"`
}

func registerKillmailTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name:        "killmail",
		Title:       "Get killmail",
		Description: "Return full victim, attacker, final blow, location, value, and point details for one killmail.",
	}, func(ctx context.Context, input KillmailInput) (KillmailOutput, error) {
		return loadKillmail(ctx, registry.deps, input)
	})
}

func loadKillmail(
	ctx context.Context,
	deps Dependencies,
	input KillmailInput,
) (KillmailOutput, error) {
	if input.KillmailID <= 0 {
		return KillmailOutput{}, fmt.Errorf("invalid killmail_id")
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT
			k.killmail_id, k.killmail_time, k.killmail_hash,
			k.solar_system_id, k.constellation_id, k.region_id,
			k.victim_character_id, k.victim_corporation_id,
			k.victim_alliance_id, k.victim_faction_id,
			k.victim_ship_type_id, k.victim_ship_group_id,
			k.victim_damage_taken, k.total_value, k.fitted_value,
			k.dropped_value, k.destroyed_value, k.points,
			k.attacker_count, k.is_npc, k.is_solo, k.war_id,
			character.name AS victim_character_name,
			corporation.name AS victim_corporation_name,
			corporation.ticker AS victim_corporation_ticker,
			alliance.name AS victim_alliance_name,
			alliance.ticker AS victim_alliance_ticker,
			faction.name AS victim_faction_name,
			ship.name AS victim_ship_name,
			ship_group.name AS victim_ship_group_name,
			system.system_name AS solar_system_name,
			system.security AS solar_system_security,
			region.name AS region_name,
			constellation.constellation_name
		FROM killmails k
		LEFT JOIN characters character
		       ON character.character_id = k.victim_character_id
		LEFT JOIN corporations corporation
		       ON corporation.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = k.victim_alliance_id
		LEFT JOIN factions faction ON faction.faction_id = k.victim_faction_id
		LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN inv_groups ship_group
		       ON ship_group.group_id = k.victim_ship_group_id
		LEFT JOIN solar_systems system ON system.solar_system_id = k.solar_system_id
		LEFT JOIN regions region ON region.region_id = k.region_id
		LEFT JOIN constellations constellation
		       ON constellation.constellation_id = k.constellation_id
		WHERE k.killmail_id = $1
		LIMIT 1`, input.KillmailID)
	if err != nil {
		return KillmailOutput{}, fmt.Errorf("load killmail: %w", err)
	}
	killmail := firstMap(rows)
	if killmail == nil {
		return KillmailOutput{}, fmt.Errorf("killmail %d not found", input.KillmailID)
	}
	attackerRows, err := queryMaps(ctx, deps.DB, `
		SELECT
			attacker.attacker_index, attacker.damage_done, attacker.final_blow,
			attacker.security_status, attacker.character_id,
			attacker.corporation_id, attacker.alliance_id, attacker.faction_id,
			attacker.ship_type_id, attacker.weapon_type_id,
			character.name AS character_name,
			corporation.name AS corporation_name,
			corporation.ticker AS corporation_ticker,
			alliance.name AS alliance_name, alliance.ticker AS alliance_ticker,
			faction.name AS faction_name, ship.name AS ship_name,
			weapon.name AS weapon_name
		FROM killmail_attackers attacker
		LEFT JOIN characters character ON character.character_id = attacker.character_id
		LEFT JOIN corporations corporation
		       ON corporation.corporation_id = attacker.corporation_id
		LEFT JOIN alliances alliance ON alliance.alliance_id = attacker.alliance_id
		LEFT JOIN factions faction ON faction.faction_id = attacker.faction_id
		LEFT JOIN inv_types ship ON ship.type_id = attacker.ship_type_id
		LEFT JOIN inv_types weapon ON weapon.type_id = attacker.weapon_type_id
		WHERE attacker.killmail_id = $1
		ORDER BY attacker.damage_done DESC NULLS LAST, attacker.attacker_index`,
		input.KillmailID)
	if err != nil {
		return KillmailOutput{}, fmt.Errorf("load killmail attackers: %w", err)
	}
	attackers := make([]KillmailAttacker, 0, len(attackerRows))
	var finalBlow *KillmailAttacker
	for _, row := range attackerRows {
		attacker := KillmailAttacker{
			CharacterID:       nullableInt64(row["character_id"]),
			CharacterName:     nullableString(row["character_name"]),
			CorporationID:     nullableInt64(row["corporation_id"]),
			CorporationName:   nullableString(row["corporation_name"]),
			CorporationTicker: nullableString(row["corporation_ticker"]),
			AllianceID:        nullableInt64(row["alliance_id"]),
			AllianceName:      nullableString(row["alliance_name"]),
			AllianceTicker:    nullableString(row["alliance_ticker"]),
			FactionID:         nullableInt64(row["faction_id"]),
			FactionName:       nullableString(row["faction_name"]),
			ShipTypeID:        nullableInt64(row["ship_type_id"]),
			ShipName:          nullableString(row["ship_name"]),
			WeaponTypeID:      nullableInt64(row["weapon_type_id"]),
			WeaponName:        nullableString(row["weapon_name"]),
			DamageDone:        valueInt64(row["damage_done"]),
			FinalBlow:         valueBool(row["final_blow"]),
			SecurityStatus:    nullableFloat64(row["security_status"]),
		}
		attackers = append(attackers, attacker)
		if attacker.FinalBlow && finalBlow == nil {
			copy := attacker
			finalBlow = &copy
		}
	}
	timeValue := nullableTime(killmail["killmail_time"])
	if timeValue == nil {
		return KillmailOutput{}, fmt.Errorf("killmail %d has no time", input.KillmailID)
	}
	return KillmailOutput{
		KillmailID:     valueInt64(killmail["killmail_id"]),
		URL:            killmailURL(deps.BaseURL, input.KillmailID),
		Time:           *timeValue,
		Hash:           valueString(killmail["killmail_hash"]),
		TotalValue:     valueFloat64(killmail["total_value"]),
		FittedValue:    valueFloat64(killmail["fitted_value"]),
		DroppedValue:   valueFloat64(killmail["dropped_value"]),
		DestroyedValue: valueFloat64(killmail["destroyed_value"]),
		Points:         valueInt64(killmail["points"]),
		AttackerCount:  valueInt64(killmail["attacker_count"]),
		IsNPC:          valueBool(killmail["is_npc"]),
		IsSolo:         valueBool(killmail["is_solo"]),
		WarID:          nullableInt64(killmail["war_id"]),
		System: KillmailLocation{
			ID:                valueInt64(killmail["solar_system_id"]),
			Name:              nullableString(killmail["solar_system_name"]),
			Security:          nullableFloat64(killmail["solar_system_security"]),
			ConstellationID:   nullableInt64(killmail["constellation_id"]),
			ConstellationName: nullableString(killmail["constellation_name"]),
			RegionID:          nullableInt64(killmail["region_id"]),
			RegionName:        nullableString(killmail["region_name"]),
		},
		Victim: KillmailVictim{
			CharacterID:       nullableInt64(killmail["victim_character_id"]),
			CharacterName:     nullableString(killmail["victim_character_name"]),
			CorporationID:     nullableInt64(killmail["victim_corporation_id"]),
			CorporationName:   nullableString(killmail["victim_corporation_name"]),
			CorporationTicker: nullableString(killmail["victim_corporation_ticker"]),
			AllianceID:        nullableInt64(killmail["victim_alliance_id"]),
			AllianceName:      nullableString(killmail["victim_alliance_name"]),
			AllianceTicker:    nullableString(killmail["victim_alliance_ticker"]),
			FactionID:         nullableInt64(killmail["victim_faction_id"]),
			FactionName:       nullableString(killmail["victim_faction_name"]),
			ShipTypeID:        nullableInt64(killmail["victim_ship_type_id"]),
			ShipName:          nullableString(killmail["victim_ship_name"]),
			ShipGroupID:       nullableInt64(killmail["victim_ship_group_id"]),
			ShipGroupName:     nullableString(killmail["victim_ship_group_name"]),
			DamageTaken:       valueInt64(killmail["victim_damage_taken"]),
		},
		FinalBlow: finalBlow,
		Attackers: attackers,
	}, nil
}
