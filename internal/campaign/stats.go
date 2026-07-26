package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type campaignStatsBlob struct {
	Totals             campaignStatsTotals             `json:"totals"`
	TopKillersBySide   map[string][]campaignTopKiller  `json:"topKillersBySide"`
	TopVictimsBySide   map[string][]campaignTopVictim  `json:"topVictimsBySide"`
	ShipClassesBySide  map[string][]campaignShipClass  `json:"shipClassesBySide"`
	TopKillersOverall  []campaignTopKiller             `json:"topKillersOverall"`
	TopVictimsOverall  []campaignTopVictim             `json:"topVictimsOverall"`
	ShipClassesOverall []campaignShipClass             `json:"shipClassesOverall"`
	TopSystems         []campaignTopSystem             `json:"topSystems"`
	IntelBySide        map[string]*campaignIntelBucket `json:"intelBySide"`
	MostValuable       []campaignMostValuable          `json:"mostValuable"`
}

type campaignStatsTotals struct {
	KillCount            int64   `json:"killCount"`
	IskDestroyed         float64 `json:"iskDestroyed"`
	CharactersInvolved   int64   `json:"charactersInvolved"`
	CorporationsInvolved int64   `json:"corporationsInvolved"`
	AlliancesInvolved    int64   `json:"alliancesInvolved"`
}

type campaignAffiliation struct {
	CorporationID     *int32  `json:"corporationId"`
	CorporationName   *string `json:"corporationName"`
	CorporationTicker *string `json:"corporationTicker"`
	AllianceID        *int32  `json:"allianceId"`
	AllianceName      *string `json:"allianceName"`
	AllianceTicker    *string `json:"allianceTicker"`
}

type campaignTopKiller struct {
	CharacterID  int32   `json:"characterId"`
	Name         *string `json:"name"`
	Kills        int64   `json:"kills"`
	IskDestroyed float64 `json:"iskDestroyed"`
	campaignAffiliation
}

type campaignTopVictim struct {
	CharacterID int32   `json:"characterId"`
	Name        *string `json:"name"`
	Losses      int64   `json:"losses"`
	IskLost     float64 `json:"iskLost"`
	campaignAffiliation
}

type campaignShipClass struct {
	GroupID int32   `json:"groupId"`
	Name    *string `json:"name"`
	Losses  int64   `json:"losses"`
	IskLost float64 `json:"iskLost"`
}

type campaignTopSystem struct {
	SystemID     int32   `json:"systemId"`
	Name         *string `json:"name"`
	RegionID     *int32  `json:"regionId"`
	RegionName   *string `json:"regionName"`
	Kills        int64   `json:"kills"`
	IskDestroyed float64 `json:"iskDestroyed"`
}

type campaignMostValuable struct {
	KillmailID            int64   `json:"killmailId"`
	Value                 float64 `json:"value"`
	ShipTypeID            *int32  `json:"shipTypeId"`
	ShipName              *string `json:"shipName"`
	VictimCharacterID     *int32  `json:"victimCharacterId"`
	VictimCharacterName   *string `json:"victimCharacterName"`
	VictimCorporationID   *int32  `json:"victimCorporationId"`
	VictimCorporationName *string `json:"victimCorporationName"`
	VictimSide            *int16  `json:"victimSide"`
	KillmailTime          string  `json:"killmailTime"`
}

type campaignIntelPilot struct {
	CharacterID     int32   `json:"characterId"`
	Name            *string `json:"name"`
	CorporationName *string `json:"corporationName"`
	AllianceName    *string `json:"allianceName"`
	ShipTypeID      *int32  `json:"shipTypeId"`
	ShipName        *string `json:"shipName"`
	ShipGroupName   *string `json:"shipGroupName"`
	Damage          int64   `json:"damage"`
	Died            bool    `json:"died"`

	shipGroupID int32
}

type campaignIntelBucket struct {
	FCs            []campaignIntelPilot `json:"fcs"`
	Logistics      []campaignIntelPilot `json:"logistics"`
	LogisticsCount int                  `json:"logisticsCount"`
	Capitals       []campaignIntelPilot `json:"capitals"`
	CapitalsCount  int                  `json:"capitalsCount"`
}

const (
	campaignIntelMonitorGroup = 1972
	campaignIntelLogisticsCap = 40
	campaignIntelCapitalsCap  = 60
)

var campaignIntelLogisticsGroups = map[int32]bool{832: true, 1527: true}
var campaignIntelCapitalGroups = map[int32]bool{
	547: true, 659: true, 30: true, 485: true, 1538: true, 4594: true,
}

// buildStats assembles the read-side JSON blob. The page serves this directly,
// so every expensive grouping belongs here in the write path rather than in an
// HTTP request.
func buildStats(
	ctx context.Context,
	tx pgx.Tx,
	campaignID string,
	area bool,
) ([]byte, *time.Time, error) {
	blob := campaignStatsBlob{
		TopKillersBySide:   make(map[string][]campaignTopKiller),
		TopVictimsBySide:   make(map[string][]campaignTopVictim),
		ShipClassesBySide:  make(map[string][]campaignShipClass),
		TopKillersOverall:  make([]campaignTopKiller, 0),
		TopVictimsOverall:  make([]campaignTopVictim, 0),
		ShipClassesOverall: make([]campaignShipClass, 0),
		TopSystems:         make([]campaignTopSystem, 0),
		IntelBySide:        make(map[string]*campaignIntelBucket),
		MostValuable:       make([]campaignMostValuable, 0),
	}

	var last sql.NullTime
	if err := tx.QueryRow(ctx, `
        SELECT
            count(*)::bigint,
            coalesce(sum(adj_value), 0),
            max(killmail_time),
            (
                SELECT count(DISTINCT character_id)
                FROM (
                    SELECT attacker.character_id
                    FROM campaign_scratch_killmails scratch
                    JOIN killmail_attackers attacker USING (killmail_id)
                    WHERE scratch.campaign_id = $1 AND attacker.character_id IS NOT NULL
                    UNION
                    SELECT victim_character_id
                    FROM campaign_scratch_killmails
                    WHERE campaign_id = $1 AND victim_character_id IS NOT NULL
                ) characters
            ),
            (
                SELECT count(DISTINCT corporation_id)
                FROM (
                    SELECT attacker.corporation_id
                    FROM campaign_scratch_killmails scratch
                    JOIN killmail_attackers attacker USING (killmail_id)
                    WHERE scratch.campaign_id = $1 AND attacker.corporation_id IS NOT NULL
                    UNION
                    SELECT victim_corporation_id
                    FROM campaign_scratch_killmails
                    WHERE campaign_id = $1 AND victim_corporation_id IS NOT NULL
                ) corporations
            ),
            (
                SELECT count(DISTINCT alliance_id)
                FROM (
                    SELECT attacker.alliance_id
                    FROM campaign_scratch_killmails scratch
                    JOIN killmail_attackers attacker USING (killmail_id)
                    WHERE scratch.campaign_id = $1 AND attacker.alliance_id IS NOT NULL
                    UNION
                    SELECT victim_alliance_id
                    FROM campaign_scratch_killmails
                    WHERE campaign_id = $1 AND victim_alliance_id IS NOT NULL
                ) alliances
            )
        FROM campaign_scratch_killmails
        WHERE campaign_id = $1`,
		campaignID,
	).Scan(
		&blob.Totals.KillCount,
		&blob.Totals.IskDestroyed,
		&last,
		&blob.Totals.CharactersInvolved,
		&blob.Totals.CorporationsInvolved,
		&blob.Totals.AlliancesInvolved,
	); err != nil {
		return nil, nil, fmt.Errorf("campaign totals: %w", err)
	}

	if err := loadTopKillers(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}
	if err := loadTopVictims(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}
	if err := loadShipClasses(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}
	if area {
		if err := loadAreaBreakdowns(ctx, tx, campaignID, &blob); err != nil {
			return nil, nil, err
		}
	}
	if err := loadTopSystems(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}
	if err := loadMostValuable(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}
	if err := loadIntel(ctx, tx, campaignID, &blob); err != nil {
		return nil, nil, err
	}

	encoded, err := json.Marshal(blob)
	if err != nil {
		return nil, nil, err
	}
	if !last.Valid {
		return encoded, nil, nil
	}
	return encoded, &last.Time, nil
}

func loadTopKillers(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT ranked.side_index, ranked.character_id, character.name,
               ranked.kills, ranked.isk_destroyed,
               ranked.corporation_id, corporation.name, corporation.ticker,
               ranked.alliance_id, alliance.name, alliance.ticker
        FROM (
            SELECT grouped.*,
                   row_number() OVER (
                       PARTITION BY grouped.side_index
                       ORDER BY grouped.kills DESC, grouped.isk_destroyed DESC
                   ) AS rank
            FROM (
                SELECT matched.side_index,
                       matched.character_id,
                       count(*)::bigint AS kills,
                       sum(matched.adj_value) AS isk_destroyed,
                       mode() WITHIN GROUP (ORDER BY matched.corporation_id) AS corporation_id,
                       mode() WITHIN GROUP (ORDER BY matched.alliance_id) AS alliance_id
                FROM (
                    SELECT DISTINCT entity.side_index,
                           attacker.character_id,
                           attacker.corporation_id,
                           attacker.alliance_id,
                           scratch.killmail_id,
                           scratch.adj_value
                    FROM campaign_scratch_killmails scratch
                    JOIN killmail_attackers attacker USING (killmail_id)
                    JOIN campaign_side_entities entity
                      ON entity.campaign_id = $1
                     AND (
                            (entity.entity_type = $2 AND entity.entity_id = attacker.character_id)
                         OR (entity.entity_type = $3 AND entity.entity_id = attacker.corporation_id)
                         OR (entity.entity_type = $4 AND entity.entity_id = attacker.alliance_id)
                     )
                    WHERE scratch.campaign_id = $1
                      AND attacker.character_id IS NOT NULL
                      AND scratch.victim_side IS DISTINCT FROM entity.side_index
                ) matched
                GROUP BY matched.side_index, matched.character_id
            ) grouped
        ) ranked
        LEFT JOIN characters character
          ON character.character_id = ranked.character_id
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = ranked.corporation_id
        LEFT JOIN alliances alliance
          ON alliance.alliance_id = ranked.alliance_id
        WHERE ranked.rank <= 10
        ORDER BY ranked.side_index, ranked.rank`,
		campaignID,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var side int16
		var item campaignTopKiller
		var name, corpName, corpTicker, allianceName, allianceTicker sql.NullString
		var corpID, allianceID sql.NullInt64
		if err := rows.Scan(
			&side,
			&item.CharacterID,
			&name,
			&item.Kills,
			&item.IskDestroyed,
			&corpID,
			&corpName,
			&corpTicker,
			&allianceID,
			&allianceName,
			&allianceTicker,
		); err != nil {
			return err
		}
		item.Name = nullString(name)
		item.campaignAffiliation = affiliation(
			corpID, corpName, corpTicker,
			allianceID, allianceName, allianceTicker,
		)
		key := strconv.Itoa(int(side))
		blob.TopKillersBySide[key] = append(blob.TopKillersBySide[key], item)
	}
	return rows.Err()
}

func loadTopVictims(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT ranked.victim_side, ranked.victim_character_id, character.name,
               ranked.losses, ranked.isk_lost,
               ranked.corporation_id, corporation.name, corporation.ticker,
               ranked.alliance_id, alliance.name, alliance.ticker
        FROM (
            SELECT scratch.victim_side,
                   scratch.victim_character_id,
                   count(*)::bigint AS losses,
                   sum(scratch.adj_value) AS isk_lost,
                   mode() WITHIN GROUP (ORDER BY scratch.victim_corporation_id) AS corporation_id,
                   mode() WITHIN GROUP (ORDER BY scratch.victim_alliance_id) AS alliance_id,
                   row_number() OVER (
                       PARTITION BY scratch.victim_side
                       ORDER BY count(*) DESC, sum(scratch.adj_value) DESC
                   ) AS rank
            FROM campaign_scratch_killmails scratch
            WHERE scratch.campaign_id = $1
              AND scratch.victim_side IS NOT NULL
              AND scratch.victim_character_id IS NOT NULL
            GROUP BY scratch.victim_side, scratch.victim_character_id
        ) ranked
        LEFT JOIN characters character
          ON character.character_id = ranked.victim_character_id
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = ranked.corporation_id
        LEFT JOIN alliances alliance
          ON alliance.alliance_id = ranked.alliance_id
        WHERE ranked.rank <= 10
        ORDER BY ranked.victim_side, ranked.rank`,
		campaignID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var side int16
		var item campaignTopVictim
		var name, corpName, corpTicker, allianceName, allianceTicker sql.NullString
		var corpID, allianceID sql.NullInt64
		if err := rows.Scan(
			&side,
			&item.CharacterID,
			&name,
			&item.Losses,
			&item.IskLost,
			&corpID,
			&corpName,
			&corpTicker,
			&allianceID,
			&allianceName,
			&allianceTicker,
		); err != nil {
			return err
		}
		item.Name = nullString(name)
		item.campaignAffiliation = affiliation(
			corpID, corpName, corpTicker,
			allianceID, allianceName, allianceTicker,
		)
		key := strconv.Itoa(int(side))
		blob.TopVictimsBySide[key] = append(blob.TopVictimsBySide[key], item)
	}
	return rows.Err()
}

func loadShipClasses(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT ranked.victim_side, ranked.victim_ship_group_id,
               group_row.name, ranked.losses, ranked.isk_lost
        FROM (
            SELECT scratch.victim_side,
                   scratch.victim_ship_group_id,
                   count(*)::bigint AS losses,
                   sum(scratch.adj_value) AS isk_lost,
                   row_number() OVER (
                       PARTITION BY scratch.victim_side
                       ORDER BY count(*) DESC
                   ) AS rank
            FROM campaign_scratch_killmails scratch
            WHERE scratch.campaign_id = $1
              AND scratch.victim_side IS NOT NULL
              AND scratch.victim_ship_group_id IS NOT NULL
            GROUP BY scratch.victim_side, scratch.victim_ship_group_id
        ) ranked
        LEFT JOIN inv_groups group_row
          ON group_row.group_id = ranked.victim_ship_group_id
        WHERE ranked.rank <= 12
        ORDER BY ranked.victim_side, ranked.rank`,
		campaignID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var side int16
		var item campaignShipClass
		var name sql.NullString
		if err := rows.Scan(
			&side,
			&item.GroupID,
			&name,
			&item.Losses,
			&item.IskLost,
		); err != nil {
			return err
		}
		item.Name = nullString(name)
		key := strconv.Itoa(int(side))
		blob.ShipClassesBySide[key] = append(blob.ShipClassesBySide[key], item)
	}
	return rows.Err()
}

func loadAreaBreakdowns(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT grouped.character_id, character.name,
               grouped.kills, grouped.isk_destroyed,
               grouped.corporation_id, corporation.name, corporation.ticker,
               grouped.alliance_id, alliance.name, alliance.ticker
        FROM (
            SELECT matched.character_id,
                   count(*)::bigint AS kills,
                   sum(matched.adj_value) AS isk_destroyed,
                   mode() WITHIN GROUP (ORDER BY matched.corporation_id) AS corporation_id,
                   mode() WITHIN GROUP (ORDER BY matched.alliance_id) AS alliance_id
            FROM (
                SELECT DISTINCT attacker.character_id,
                       attacker.corporation_id,
                       attacker.alliance_id,
                       scratch.killmail_id,
                       scratch.adj_value
                FROM campaign_scratch_killmails scratch
                JOIN killmail_attackers attacker USING (killmail_id)
                WHERE scratch.campaign_id = $1
                  AND attacker.character_id IS NOT NULL
            ) matched
            GROUP BY matched.character_id
            ORDER BY count(*) DESC, sum(matched.adj_value) DESC
            LIMIT 10
        ) grouped
        LEFT JOIN characters character USING (character_id)
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = grouped.corporation_id
        LEFT JOIN alliances alliance
          ON alliance.alliance_id = grouped.alliance_id
        ORDER BY grouped.kills DESC, grouped.isk_destroyed DESC`,
		campaignID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item campaignTopKiller
		var name, corpName, corpTicker, allianceName, allianceTicker sql.NullString
		var corpID, allianceID sql.NullInt64
		if err := rows.Scan(
			&item.CharacterID, &name, &item.Kills, &item.IskDestroyed,
			&corpID, &corpName, &corpTicker,
			&allianceID, &allianceName, &allianceTicker,
		); err != nil {
			rows.Close()
			return err
		}
		item.Name = nullString(name)
		item.campaignAffiliation = affiliation(
			corpID, corpName, corpTicker,
			allianceID, allianceName, allianceTicker,
		)
		blob.TopKillersOverall = append(blob.TopKillersOverall, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	rows, err = tx.Query(ctx, `
        SELECT grouped.character_id, character.name,
               grouped.losses, grouped.isk_lost,
               grouped.corporation_id, corporation.name, corporation.ticker,
               grouped.alliance_id, alliance.name, alliance.ticker
        FROM (
            SELECT scratch.victim_character_id AS character_id,
                   count(*)::bigint AS losses,
                   sum(scratch.adj_value) AS isk_lost,
                   mode() WITHIN GROUP (ORDER BY scratch.victim_corporation_id) AS corporation_id,
                   mode() WITHIN GROUP (ORDER BY scratch.victim_alliance_id) AS alliance_id
            FROM campaign_scratch_killmails scratch
            WHERE scratch.campaign_id = $1
              AND scratch.victim_character_id IS NOT NULL
            GROUP BY scratch.victim_character_id
            ORDER BY count(*) DESC, sum(scratch.adj_value) DESC
            LIMIT 10
        ) grouped
        LEFT JOIN characters character USING (character_id)
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = grouped.corporation_id
        LEFT JOIN alliances alliance
          ON alliance.alliance_id = grouped.alliance_id
        ORDER BY grouped.losses DESC, grouped.isk_lost DESC`,
		campaignID,
	)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item campaignTopVictim
		var name, corpName, corpTicker, allianceName, allianceTicker sql.NullString
		var corpID, allianceID sql.NullInt64
		if err := rows.Scan(
			&item.CharacterID, &name, &item.Losses, &item.IskLost,
			&corpID, &corpName, &corpTicker,
			&allianceID, &allianceName, &allianceTicker,
		); err != nil {
			rows.Close()
			return err
		}
		item.Name = nullString(name)
		item.campaignAffiliation = affiliation(
			corpID, corpName, corpTicker,
			allianceID, allianceName, allianceTicker,
		)
		blob.TopVictimsOverall = append(blob.TopVictimsOverall, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	rows, err = tx.Query(ctx, `
        SELECT scratch.victim_ship_group_id,
               group_row.name,
               count(*)::bigint,
               coalesce(sum(scratch.adj_value), 0)
        FROM campaign_scratch_killmails scratch
        LEFT JOIN inv_groups group_row
          ON group_row.group_id = scratch.victim_ship_group_id
        WHERE scratch.campaign_id = $1
          AND scratch.victim_ship_group_id IS NOT NULL
        GROUP BY scratch.victim_ship_group_id, group_row.name
        ORDER BY count(*) DESC, sum(scratch.adj_value) DESC
        LIMIT 12`,
		campaignID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item campaignShipClass
		var name sql.NullString
		if err := rows.Scan(&item.GroupID, &name, &item.Losses, &item.IskLost); err != nil {
			return err
		}
		item.Name = nullString(name)
		blob.ShipClassesOverall = append(blob.ShipClassesOverall, item)
	}
	return rows.Err()
}

func loadTopSystems(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT scratch.solar_system_id,
               system.system_name,
               system.region_id,
               region.name,
               count(*)::bigint,
               coalesce(sum(scratch.adj_value), 0)
        FROM campaign_scratch_killmails scratch
        LEFT JOIN solar_systems system
          ON system.solar_system_id = scratch.solar_system_id
        LEFT JOIN regions region
          ON region.region_id = system.region_id
        WHERE scratch.campaign_id = $1
        GROUP BY scratch.solar_system_id, system.system_name,
                 system.region_id, region.name
        ORDER BY count(*) DESC
        LIMIT 10`,
		campaignID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item campaignTopSystem
		var name, regionName sql.NullString
		var regionID sql.NullInt64
		if err := rows.Scan(
			&item.SystemID,
			&name,
			&regionID,
			&regionName,
			&item.Kills,
			&item.IskDestroyed,
		); err != nil {
			return err
		}
		item.Name = nullString(name)
		item.RegionID = nullInt32(regionID)
		item.RegionName = nullString(regionName)
		blob.TopSystems = append(blob.TopSystems, item)
	}
	return rows.Err()
}

func loadMostValuable(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        SELECT scratch.killmail_id,
               scratch.adj_value,
               scratch.victim_ship_type_id,
               ship.name,
               scratch.victim_character_id,
               character.name,
               scratch.victim_corporation_id,
               corporation.name,
               scratch.victim_side,
               scratch.killmail_time
        FROM campaign_scratch_killmails scratch
        LEFT JOIN inv_types ship
          ON ship.type_id = scratch.victim_ship_type_id
        LEFT JOIN characters character
          ON character.character_id = scratch.victim_character_id
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = scratch.victim_corporation_id
        WHERE scratch.campaign_id = $1
        ORDER BY scratch.adj_value DESC
        LIMIT 10`,
		campaignID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item campaignMostValuable
		var shipID, characterID, corporationID, side sql.NullInt64
		var shipName, characterName, corporationName sql.NullString
		var killmailTime time.Time
		if err := rows.Scan(
			&item.KillmailID,
			&item.Value,
			&shipID,
			&shipName,
			&characterID,
			&characterName,
			&corporationID,
			&corporationName,
			&side,
			&killmailTime,
		); err != nil {
			return err
		}
		item.ShipTypeID = nullInt32(shipID)
		item.ShipName = nullString(shipName)
		item.VictimCharacterID = nullInt32(characterID)
		item.VictimCharacterName = nullString(characterName)
		item.VictimCorporationID = nullInt32(corporationID)
		item.VictimCorporationName = nullString(corporationName)
		item.VictimSide = nullInt16(side)
		item.KillmailTime = killmailTime.UTC().Format("2006-01-02T15:04:05.000Z")
		blob.MostValuable = append(blob.MostValuable, item)
	}
	return rows.Err()
}

func loadIntel(ctx context.Context, tx pgx.Tx, campaignID string, blob *campaignStatsBlob) error {
	rows, err := tx.Query(ctx, `
        WITH pilots AS (
            SELECT DISTINCT ON (entity.side_index, attacker.character_id)
                   entity.side_index,
                   attacker.character_id,
                   attacker.corporation_id,
                   attacker.alliance_id,
                   attacker.ship_type_id,
                   attacker.ship_group_id,
                   sum(coalesce(attacker.damage_done, 0)) OVER (
                       PARTITION BY entity.side_index, attacker.character_id, attacker.ship_type_id
                   ) AS ship_damage,
                   sum(coalesce(attacker.damage_done, 0)) OVER (
                       PARTITION BY entity.side_index, attacker.character_id
                   ) AS total_damage
            FROM campaign_scratch_killmails scratch
            JOIN killmail_attackers attacker USING (killmail_id)
            JOIN campaign_side_entities entity
              ON entity.campaign_id = $1
             AND (
                    (entity.entity_type = $2 AND entity.entity_id = attacker.character_id)
                 OR (entity.entity_type = $3 AND entity.entity_id = attacker.corporation_id)
                 OR (entity.entity_type = $4 AND entity.entity_id = attacker.alliance_id)
             )
            WHERE scratch.campaign_id = $1
              AND attacker.character_id IS NOT NULL
              AND attacker.ship_type_id IS NOT NULL
            ORDER BY entity.side_index, attacker.character_id, ship_damage DESC
        ),
        deaths AS (
            SELECT DISTINCT victim_character_id
            FROM campaign_scratch_killmails
            WHERE campaign_id = $1 AND victim_character_id IS NOT NULL
        )
        SELECT pilot.side_index,
               pilot.character_id,
               character.name,
               corporation.name,
               alliance.name,
               pilot.ship_type_id,
               ship.name,
               pilot.ship_group_id,
               group_row.name,
               pilot.total_damage::bigint,
               death.victim_character_id IS NOT NULL
        FROM pilots pilot
        LEFT JOIN deaths death
          ON death.victim_character_id = pilot.character_id
        LEFT JOIN characters character
          ON character.character_id = pilot.character_id
        LEFT JOIN corporations corporation
          ON corporation.corporation_id = pilot.corporation_id
        LEFT JOIN alliances alliance
          ON alliance.alliance_id = pilot.alliance_id
        LEFT JOIN inv_types ship
          ON ship.type_id = pilot.ship_type_id
        LEFT JOIN inv_groups group_row
          ON group_row.group_id = pilot.ship_group_id
        WHERE pilot.ship_group_id = ANY($5::int[])`,
		campaignID,
		EntityCharacter,
		EntityCorporation,
		EntityAlliance,
		[]int32{campaignIntelMonitorGroup, 832, 1527, 547, 659, 30, 485, 1538, 4594},
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var side int16
		var pilot campaignIntelPilot
		var name, corporationName, allianceName, shipName, groupName sql.NullString
		var shipTypeID sql.NullInt64
		if err := rows.Scan(
			&side,
			&pilot.CharacterID,
			&name,
			&corporationName,
			&allianceName,
			&shipTypeID,
			&shipName,
			&pilot.shipGroupID,
			&groupName,
			&pilot.Damage,
			&pilot.Died,
		); err != nil {
			return err
		}
		pilot.Name = nullString(name)
		pilot.CorporationName = nullString(corporationName)
		pilot.AllianceName = nullString(allianceName)
		pilot.ShipTypeID = nullInt32(shipTypeID)
		pilot.ShipName = nullString(shipName)
		pilot.ShipGroupName = nullString(groupName)

		key := strconv.Itoa(int(side))
		bucket := blob.IntelBySide[key]
		if bucket == nil {
			bucket = &campaignIntelBucket{
				FCs:       make([]campaignIntelPilot, 0),
				Logistics: make([]campaignIntelPilot, 0),
				Capitals:  make([]campaignIntelPilot, 0),
			}
			blob.IntelBySide[key] = bucket
		}
		switch {
		case pilot.shipGroupID == campaignIntelMonitorGroup:
			bucket.FCs = append(bucket.FCs, pilot)
		case campaignIntelLogisticsGroups[pilot.shipGroupID]:
			bucket.Logistics = append(bucket.Logistics, pilot)
		case campaignIntelCapitalGroups[pilot.shipGroupID]:
			bucket.Capitals = append(bucket.Capitals, pilot)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, bucket := range blob.IntelBySide {
		byDamage := func(pilots []campaignIntelPilot) {
			sort.Slice(pilots, func(i, j int) bool {
				return pilots[i].Damage > pilots[j].Damage
			})
		}
		byDamage(bucket.FCs)
		byDamage(bucket.Logistics)
		byDamage(bucket.Capitals)
		bucket.LogisticsCount = len(bucket.Logistics)
		bucket.CapitalsCount = len(bucket.Capitals)
		if len(bucket.Logistics) > campaignIntelLogisticsCap {
			bucket.Logistics = bucket.Logistics[:campaignIntelLogisticsCap]
		}
		if len(bucket.Capitals) > campaignIntelCapitalsCap {
			bucket.Capitals = bucket.Capitals[:campaignIntelCapitalsCap]
		}
	}
	return nil
}

func affiliation(
	corporationID sql.NullInt64,
	corporationName sql.NullString,
	corporationTicker sql.NullString,
	allianceID sql.NullInt64,
	allianceName sql.NullString,
	allianceTicker sql.NullString,
) campaignAffiliation {
	return campaignAffiliation{
		CorporationID:     nullInt32(corporationID),
		CorporationName:   nullString(corporationName),
		CorporationTicker: nullString(corporationTicker),
		AllianceID:        nullInt32(allianceID),
		AllianceName:      nullString(allianceName),
		AllianceTicker:    nullString(allianceTicker),
	}
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullInt32(value sql.NullInt64) *int32 {
	if !value.Valid {
		return nil
	}
	out := int32(value.Int64)
	return &out
}

func nullInt16(value sql.NullInt64) *int16 {
	if !value.Valid {
		return nil
	}
	out := int16(value.Int64)
	return &out
}
