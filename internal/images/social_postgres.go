package images

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
)

type SocialDatabase interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresSocialLoader struct {
	DB SocialDatabase
}

func (l PostgresSocialLoader) LoadKillmailSocial(
	ctx context.Context,
	id int64,
) (SocialKillmail, bool, error) {
	if l.DB == nil {
		return SocialKillmail{}, false, errors.New("social image database is nil")
	}
	var (
		totalValue float64
		systemName sql.NullString
		regionName sql.NullString

		victimCharacterID     sql.NullInt64
		victimCharacterName   sql.NullString
		victimCorporationID   sql.NullInt64
		victimCorporationName sql.NullString
		victimAllianceID      sql.NullInt64
		victimAllianceName    sql.NullString
		victimShipTypeID      sql.NullInt64
		victimShipName        sql.NullString

		finalCharacterID     sql.NullInt64
		finalCharacterName   sql.NullString
		finalCorporationID   sql.NullInt64
		finalCorporationName sql.NullString
		finalAllianceID      sql.NullInt64
		finalAllianceName    sql.NullString
		finalShipTypeID      sql.NullInt64
		finalShipName        sql.NullString
	)
	err := l.DB.QueryRow(ctx, `
		SELECT
			COALESCE(k.total_value, 0),
			system.system_name,
			region.name,
			k.victim_character_id,
			victim_character.name,
			k.victim_corporation_id,
			victim_corporation.name,
			k.victim_alliance_id,
			victim_alliance.name,
			k.victim_ship_type_id,
			victim_ship.name,
			final_blow.character_id,
			final_character.name,
			final_blow.corporation_id,
			final_corporation.name,
			final_blow.alliance_id,
			final_alliance.name,
			final_blow.ship_type_id,
			final_ship.name
		FROM killmails k
		LEFT JOIN LATERAL (
			SELECT character_id, corporation_id, alliance_id, ship_type_id
			FROM killmail_attackers
			WHERE killmail_id = k.killmail_id
			  AND killmail_time = k.killmail_time
			  AND final_blow = true
			ORDER BY attacker_index
			LIMIT 1
		) final_blow ON true
		LEFT JOIN characters victim_character
			ON victim_character.character_id = k.victim_character_id
		LEFT JOIN corporations victim_corporation
			ON victim_corporation.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances victim_alliance
			ON victim_alliance.alliance_id = k.victim_alliance_id
		LEFT JOIN inv_types victim_ship
			ON victim_ship.type_id = k.victim_ship_type_id
		LEFT JOIN characters final_character
			ON final_character.character_id = final_blow.character_id
		LEFT JOIN corporations final_corporation
			ON final_corporation.corporation_id = final_blow.corporation_id
		LEFT JOIN alliances final_alliance
			ON final_alliance.alliance_id = final_blow.alliance_id
		LEFT JOIN inv_types final_ship
			ON final_ship.type_id = final_blow.ship_type_id
		LEFT JOIN solar_systems system
			ON system.solar_system_id = k.solar_system_id
		LEFT JOIN regions region
			ON region.region_id = k.region_id
		WHERE k.killmail_id = $1
		ORDER BY k.killmail_time DESC
		LIMIT 1`,
		id,
	).Scan(
		&totalValue,
		&systemName,
		&regionName,
		&victimCharacterID,
		&victimCharacterName,
		&victimCorporationID,
		&victimCorporationName,
		&victimAllianceID,
		&victimAllianceName,
		&victimShipTypeID,
		&victimShipName,
		&finalCharacterID,
		&finalCharacterName,
		&finalCorporationID,
		&finalCorporationName,
		&finalAllianceID,
		&finalAllianceName,
		&finalShipTypeID,
		&finalShipName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SocialKillmail{}, false, nil
	}
	if err != nil {
		return SocialKillmail{}, false, err
	}
	result := SocialKillmail{
		TotalValue:      totalValue,
		SolarSystemName: systemName.String,
		RegionName:      regionName.String,
		Victim: SocialParty{
			CharacterID:     victimCharacterID.Int64,
			CharacterName:   victimCharacterName.String,
			CorporationID:   victimCorporationID.Int64,
			CorporationName: victimCorporationName.String,
			AllianceID:      victimAllianceID.Int64,
			AllianceName:    victimAllianceName.String,
			ShipTypeID:      victimShipTypeID.Int64,
			ShipName:        victimShipName.String,
		},
	}
	if finalCharacterID.Valid || finalCorporationID.Valid ||
		finalAllianceID.Valid || finalShipTypeID.Valid {
		result.FinalBlow = &SocialParty{
			CharacterID:     finalCharacterID.Int64,
			CharacterName:   finalCharacterName.String,
			CorporationID:   finalCorporationID.Int64,
			CorporationName: finalCorporationName.String,
			AllianceID:      finalAllianceID.Int64,
			AllianceName:    finalAllianceName.String,
			ShipTypeID:      finalShipTypeID.Int64,
			ShipName:        finalShipName.String,
		}
	}
	return result, true, nil
}
