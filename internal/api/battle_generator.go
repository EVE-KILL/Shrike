package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/battle"
	"github.com/jackc/pgx/v5"
)

type conflictBattleGeneratorWindow struct {
	SystemIDs []int32 `json:"systemIds" minItems:"1" nullable:"false" doc:"Solar systems to scan for killmails."`
	StartTime string  `json:"startTime" format:"date-time" doc:"Window start, as an ISO 8601 timestamp."`
	EndTime   string  `json:"endTime" format:"date-time" doc:"Window end, as an ISO 8601 timestamp."`
}

type conflictBattleGeneratorEntity struct {
	ID   int32  `json:"id"`
	Type string `json:"type"`
}

type conflictBattleGeneratorSide struct {
	Name     string                          `json:"name"`
	Entities []conflictBattleGeneratorEntity `json:"entities" nullable:"false"`
}

type conflictBattleGeneratorPreview struct {
	conflictBattleGeneratorWindow
	Sides []conflictBattleGeneratorSide `json:"sides" nullable:"false"`
}

type conflictBattleSaveCorporation struct {
	CorporationID int32   `json:"corporation_id"`
	Kills         int64   `json:"kills"`
	Losses        int64   `json:"losses"`
	IskDestroyed  float64 `json:"isk_destroyed"`
	IskLost       float64 `json:"isk_lost"`
}

type conflictBattleSaveAlliance struct {
	AllianceID   *int32                          `json:"alliance_id"`
	Corporations []conflictBattleSaveCorporation `json:"corporations" nullable:"false"`
}

type conflictBattleSaveTeam struct {
	TotalKills        int64                        `json:"total_kills"`
	TotalLosses       int64                        `json:"total_losses"`
	TotalIskDestroyed float64                      `json:"total_isk_destroyed"`
	TotalIskLost      float64                      `json:"total_isk_lost"`
	Alliances         []conflictBattleSaveAlliance `json:"alliances" nullable:"false"`
}

type conflictBattleSaveBody struct {
	BattleID          int32                    `json:"battle_id"`
	SolarSystemID     int32                    `json:"solar_system_id"`
	RegionID          *int32                   `json:"region_id"`
	StartTime         string                   `json:"start_time" format:"date-time"`
	EndTime           string                   `json:"end_time" format:"date-time"`
	DurationMinutes   int                      `json:"duration_minutes"`
	KillCount         int                      `json:"kill_count"`
	TotalIskDestroyed float64                  `json:"total_isk_destroyed"`
	IsMultiParty      bool                     `json:"is_multi_party"`
	Teams             []conflictBattleSaveTeam `json:"teams" minItems:"2" maxItems:"2" nullable:"false"`
}

func registerBattleGeneratorRoutes(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "battle-generator-entities",
		Method:      http.MethodPost,
		Path:        "/battle/generator/entities",
		Summary:     "Entities present in a custom battle window",
		Tags:        []string{"battles"},
	}, conflictMaximumBodyBytes, battleGeneratorEntitiesHandler(opts))
	registerLegacyJSON(a, huma.Operation{
		OperationID: "battle-generator-preview",
		Method:      http.MethodPost,
		Path:        "/battle/generator/preview",
		Summary:     "Preview user-assigned battle sides",
		Tags:        []string{"battles"},
	}, conflictMaximumBodyBytes, battleGeneratorPreviewHandler(opts))
	registerLegacy(a, documentJSONBody[conflictBattleSaveBody](a, huma.Operation{
		OperationID: "battle-generator-save",
		Method:      http.MethodPost,
		Path:        "/battle/generator/save",
		Summary:     "Save a custom battle or corrected sides",
		Tags:        []string{"battles"},
		Security:    []map[string][]string{{"eveSession": {}}},
	}), battleGeneratorSaveHandler(opts))
}

func parseConflictGeneratorWindow(
	value conflictBattleGeneratorWindow,
) ([]int32, time.Time, time.Time, error) {
	if len(value.SystemIDs) == 0 {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "Missing systemIds, startTime, or endTime")
	}
	if len(value.SystemIDs) > conflictMaximumSystems {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "Maximum 5 systems")
	}
	systems := uniqueConflictIDs(value.SystemIDs)
	if len(systems) != len(value.SystemIDs) {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "System IDs must be unique")
	}
	for _, id := range systems {
		if id <= 0 {
			return nil, time.Time{}, time.Time{},
				apiError(http.StatusBadRequest, "Invalid system IDs")
		}
	}
	start, err := parseJavaScriptDate(value.StartTime)
	if err != nil {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "Invalid startTime")
	}
	end, err := parseJavaScriptDate(value.EndTime)
	if err != nil {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "Invalid endTime")
	}
	if !start.Before(end) {
		return nil, time.Time{}, time.Time{},
			apiError(http.StatusBadRequest, "startTime must be before endTime")
	}
	return systems, start.UTC(), end.UTC(), nil
}

func battleGeneratorEntitiesHandler(opts Options) bodyHandler[conflictBattleGeneratorWindow] {
	return func(
		ctx context.Context, req *legacyRequest, body *conflictBattleGeneratorWindow,
	) (legacyPayload, error) {
		setConflictNoStore(req.Huma)
		systemIDs, start, end, err := parseConflictGeneratorWindow(*body)
		if err != nil {
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB, `
			WITH battle_kills AS MATERIALIZED (
				SELECT killmail_id, victim_corporation_id,
				       victim_alliance_id
				FROM killmails
				WHERE solar_system_id = ANY($1::int[])
				  AND killmail_time >= $2 AND killmail_time <= $3
			),
			entities AS (
				SELECT victim_corporation_id AS corporation_id,
				       victim_alliance_id AS alliance_id
				FROM battle_kills
				WHERE victim_corporation_id IS NOT NULL
				UNION
				SELECT attacker.corporation_id, attacker.alliance_id
				FROM killmail_attackers attacker
				JOIN battle_kills kill
				  ON kill.killmail_id = attacker.killmail_id
				WHERE attacker.corporation_id IS NOT NULL
				  AND attacker.character_id IS NOT NULL
				  AND attacker.killmail_time >= $2
				  AND attacker.killmail_time <= $3
			)
			SELECT entity.corporation_id, corporation.name AS corporation_name,
			       COALESCE(entity.alliance_id, corporation.alliance_id)
			           AS alliance_id,
			       alliance.name AS alliance_name
			FROM entities entity
			LEFT JOIN corporations corporation
			  ON corporation.corporation_id = entity.corporation_id
			LEFT JOIN alliances alliance
			  ON alliance.alliance_id =
			     COALESCE(entity.alliance_id, corporation.alliance_id)
			ORDER BY alliance.name NULLS LAST, corporation.name`,
			systemIDs, start, end,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		count, err := queryMap(ctx, opts.DB, `
			SELECT COUNT(*)::int AS count
			FROM killmails
			WHERE solar_system_id = ANY($1::int[])
			  AND killmail_time >= $2 AND killmail_time <= $3`,
			systemIDs, start, end,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		allianceByID := map[int64]map[string]any{}
		corporations := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			allianceID, hasAlliance := int64Value(row["alliance_id"])
			if hasAlliance && allianceID != 0 {
				if _, exists := allianceByID[allianceID]; !exists {
					allianceByID[allianceID] = map[string]any{
						"id":   allianceID,
						"name": conflictString(row, "alliance_name"),
						"type": "alliance",
					}
				}
			}
			corporations = append(corporations, map[string]any{
				"id":            conflictInt(row, "corporation_id"),
				"name":          conflictString(row, "corporation_name"),
				"type":          "corporation",
				"alliance_id":   conflictNullableID(row, "alliance_id"),
				"alliance_name": conflictNullableString(row, "alliance_name"),
			})
		}
		alliances := make([]map[string]any, 0, len(allianceByID))
		for _, row := range allianceByID {
			alliances = append(alliances, row)
		}
		sort.Slice(alliances, func(i, j int) bool {
			return conflictString(alliances[i], "name") <
				conflictString(alliances[j], "name")
		})
		return conflictNoStorePayload(map[string]any{
			"alliances": alliances, "corporations": corporations,
			"killCount": conflictInt(count, "count"),
		}), nil
	}
}

func battleGeneratorPreviewHandler(opts Options) bodyHandler[conflictBattleGeneratorPreview] {
	return func(
		ctx context.Context, req *legacyRequest, body *conflictBattleGeneratorPreview,
	) (legacyPayload, error) {
		setConflictNoStore(req.Huma)
		systemIDs, start, end, err := parseConflictGeneratorWindow(
			body.conflictBattleGeneratorWindow,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(body.Sides) != 2 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Exactly two sides are required",
			)
		}
		assignment := battle.TeamAssignment{
			CorpTeam: map[int32]int{}, CorpAlliance: map[int32]int32{},
		}
		allianceTeams := map[int32]int{}
		for index, side := range body.Sides {
			for _, entity := range side.Entities {
				if entity.ID <= 0 {
					return legacyPayload{}, apiError(
						http.StatusBadRequest, "Invalid side entity",
					)
				}
				switch entity.Type {
				case "corporation":
					if previous, exists := assignment.CorpTeam[entity.ID]; exists &&
						previous != index {
						return legacyPayload{}, apiError(
							http.StatusBadRequest,
							"A corporation cannot be on both sides",
						)
					}
					assignment.CorpTeam[entity.ID] = index
				case "alliance":
					if previous, exists := allianceTeams[entity.ID]; exists &&
						previous != index {
						return legacyPayload{}, apiError(
							http.StatusBadRequest,
							"An alliance cannot be on both sides",
						)
					}
					allianceTeams[entity.ID] = index
				default:
					return legacyPayload{}, apiError(
						http.StatusBadRequest, "Invalid side entity type",
					)
				}
			}
		}
		kills, err := loadConflictBattleKillmails(
			ctx, opts.DB, systemIDs, start, end, false,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(kills) == 0 {
			return legacyPayload{}, apiError(
				http.StatusNotFound,
				"No killmails found in the specified window",
			)
		}
		attackers, err := loadConflictBattleAttackers(
			ctx, opts.DB, systemIDs, start, end,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		for _, kill := range kills {
			if kill.VictimCorporationID != 0 && kill.VictimAllianceID != 0 {
				assignment.CorpAlliance[kill.VictimCorporationID] =
					kill.VictimAllianceID
			}
		}
		for _, killAttackers := range attackers {
			for _, attacker := range killAttackers {
				if attacker.CorporationID != 0 && attacker.AllianceID != 0 {
					assignment.CorpAlliance[attacker.CorporationID] =
						attacker.AllianceID
				}
			}
		}
		for corporationID, allianceID := range assignment.CorpAlliance {
			if _, directlyAssigned := assignment.CorpTeam[corporationID]; directlyAssigned {
				continue
			}
			if team, exists := allianceTeams[allianceID]; exists {
				assignment.CorpTeam[corporationID] = team
			}
		}
		teams := battle.ComputeTeamStats(kills, attackers, assignment)
		window := conflictBattleWindow{
			SystemIDs: systemIDs, SystemID: systemIDs[0],
			RegionID: kills[0].RegionID, Start: start, End: end,
			IsCustom: true, Assignment: assignment,
			TeamIndices: []int{0, 1},
		}
		names, err := loadBattleNames(
			ctx, opts.DB, window, teams, kills, attackers,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		formatted := formatBattleTeams(teams, names)
		for index := range formatted {
			formatted[index]["name"] = body.Sides[index].Name
			if body.Sides[index].Name == "" {
				formatted[index]["name"] = fmt.Sprintf("Team %d", index+1)
			}
		}
		participants, err := loadBattleParticipantCounts(
			ctx, opts.DB, systemIDs, start, end,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		teamEntities := make([]map[string]any, 0, len(formatted))
		for _, team := range formatted {
			corps, alliances := battleTeamEntities(team)
			teamEntities = append(teamEntities, map[string]any{
				"corps": corps, "alliances": alliances,
			})
		}
		var totalISK float64
		for _, kill := range kills {
			totalISK += kill.TotalValue
		}
		systems := make([]map[string]any, 0, len(systemIDs))
		for _, id := range systemIDs {
			if system := names.Systems[id]; system != nil {
				systems = append(systems, map[string]any{
					"id": id, "name": conflictString(system, "name"),
					"security": conflictFloat(system, "security"),
				})
			}
		}
		primary := names.Systems[systemIDs[0]]
		result := map[string]any{
			"solar_system_id":       systemIDs[0],
			"solar_system_name":     conflictNullableString(primary, "name"),
			"solar_system_security": conflictNullableFloat(primary, "security"),
			"region_id":             nullableConflictInt32(kills[0].RegionID),
			"region_name":           conflictNullableString(names.Region, "name"),
			"start_time":            start, "end_time": end,
			"duration_minutes": int(math.Round(end.Sub(start).Minutes())),
			"kill_count":       len(kills), "total_isk_destroyed": totalISK,
			"is_multi_party": false, "is_custom": true,
			"characters_involved":   conflictInt(participants, "characters"),
			"corporations_involved": conflictInt(participants, "corporations"),
			"alliances_involved":    conflictInt(participants, "alliances"),
			"total_damage":          conflictInt(participants, "total_damage"),
			"teams":                 formatted, "team_entities": teamEntities,
			"systems": systems,
		}
		return conflictNoStorePayload(result), nil
	}
}

func battleGeneratorSaveHandler(opts Options) legacyHandler {
	auth := newAuthService(opts)
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setConflictNoStore(req.Huma)
		principal, err := auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[conflictBattleSaveBody](req, conflictMaximumBodyBytes)
		if err != nil {
			return legacyPayload{}, err
		}
		start, err := parseJavaScriptDate(body.StartTime)
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid start_time")
		}
		end, err := parseJavaScriptDate(body.EndTime)
		if err != nil {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid end_time")
		}
		if body.SolarSystemID <= 0 || !start.Before(end) || len(body.Teams) != 2 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Missing or invalid required fields",
			)
		}
		if body.DurationMinutes < 0 || body.KillCount < 0 ||
			body.TotalIskDestroyed < 0 ||
			math.IsNaN(body.TotalIskDestroyed) ||
			math.IsInf(body.TotalIskDestroyed, 0) {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid battle totals",
			)
		}
		if err := validateConflictSaveTeams(body.Teams); err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

		battleID := body.BattleID
		if battleID > 0 {
			var exists int32
			err = tx.QueryRow(ctx,
				`SELECT battle_id FROM battles WHERE battle_id = $1 FOR UPDATE`,
				battleID,
			).Scan(&exists)
			if errors.Is(err, pgx.ErrNoRows) {
				return legacyPayload{}, apiError(http.StatusNotFound, "Battle not found")
			}
			if err != nil {
				return legacyPayload{}, err
			}
			if _, err = tx.Exec(ctx, `
				DELETE FROM battle_team_members member
				USING battle_teams team
				WHERE member.battle_team_id = team.id
				  AND team.battle_id = $1`,
				battleID,
			); err != nil {
				return legacyPayload{}, err
			}
			if _, err = tx.Exec(ctx,
				`DELETE FROM battle_teams WHERE battle_id = $1`,
				battleID,
			); err != nil {
				return legacyPayload{}, err
			}
			_, err = tx.Exec(ctx, `
				UPDATE battles
				SET solar_system_id = $2, region_id = $3,
				    start_time = $4, end_time = $5,
				    duration_minutes = $6, kill_count = $7,
				    total_isk_destroyed = $8, is_multi_party = $9,
				    updated_at = NOW()
				WHERE battle_id = $1`,
				battleID, body.SolarSystemID, body.RegionID,
				start, end, body.DurationMinutes, body.KillCount,
				body.TotalIskDestroyed, body.IsMultiParty,
			)
			if err != nil {
				return legacyPayload{}, err
			}
		} else {
			err = tx.QueryRow(ctx, `
				INSERT INTO battles (
					solar_system_id, region_id, start_time, end_time,
					duration_minutes, kill_count, total_isk_destroyed,
					is_multi_party, is_custom, created_by_character_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9)
				RETURNING battle_id`,
				body.SolarSystemID, body.RegionID, start, end,
				body.DurationMinutes, body.KillCount,
				body.TotalIskDestroyed, body.IsMultiParty,
				principal.CharacterID,
			).Scan(&battleID)
			if err != nil {
				return legacyPayload{}, err
			}
		}
		if err := writeConflictBattleTeams(ctx, tx, battleID, body.Teams); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		invalidateConflictBattleCache(context.WithoutCancel(ctx), opts)
		return conflictNoStorePayload(map[string]any{"battle_id": battleID}), nil
	}
}

func conflictNoStorePayload(body any) legacyPayload {
	headers := make(http.Header)
	headers.Set("Cache-Control", "private, no-store")
	headers.Set("Pragma", "no-cache")
	return legacyPayload{Body: body, Headers: headers}
}

func setConflictNoStore(ctx huma.Context) {
	ctx.SetHeader("Cache-Control", "private, no-store")
	ctx.SetHeader("Pragma", "no-cache")
}

func validateConflictSaveTeams(teams []conflictBattleSaveTeam) error {
	seen := map[int32]int{}
	for teamIndex, team := range teams {
		values := []float64{
			team.TotalIskDestroyed, team.TotalIskLost,
		}
		if team.TotalKills < 0 || team.TotalLosses < 0 {
			return apiError(http.StatusBadRequest, "Invalid team totals")
		}
		for _, value := range values {
			if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return apiError(http.StatusBadRequest, "Invalid team totals")
			}
		}
		for _, alliance := range team.Alliances {
			if alliance.AllianceID != nil && *alliance.AllianceID <= 0 {
				return apiError(http.StatusBadRequest, "Invalid alliance ID")
			}
			for _, corporation := range alliance.Corporations {
				if corporation.CorporationID <= 0 {
					return apiError(http.StatusBadRequest, "Invalid corporation ID")
				}
				if previous, exists := seen[corporation.CorporationID]; exists {
					return apiError(
						http.StatusBadRequest,
						fmt.Sprintf(
							"Corporation %d appears more than once (first on team %d)",
							corporation.CorporationID,
							previous+1,
						),
					)
				}
				seen[corporation.CorporationID] = teamIndex
				if corporation.Kills < 0 || corporation.Losses < 0 ||
					corporation.IskDestroyed < 0 || corporation.IskLost < 0 ||
					math.IsNaN(corporation.IskDestroyed) ||
					math.IsNaN(corporation.IskLost) ||
					math.IsInf(corporation.IskDestroyed, 0) ||
					math.IsInf(corporation.IskLost, 0) {
					return apiError(
						http.StatusBadRequest, "Invalid corporation totals",
					)
				}
			}
		}
	}
	return nil
}

func writeConflictBattleTeams(
	ctx context.Context,
	tx pgx.Tx,
	battleID int32,
	teams []conflictBattleSaveTeam,
) error {
	for index, team := range teams {
		var teamID int32
		err := tx.QueryRow(ctx, `
			INSERT INTO battle_teams (
				battle_id, team_index, total_kills, total_losses,
				total_isk_destroyed, total_isk_lost
			) VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			battleID, index, team.TotalKills, team.TotalLosses,
			team.TotalIskDestroyed, team.TotalIskLost,
		).Scan(&teamID)
		if err != nil {
			return err
		}
		memberRows := make([][]any, 0)
		for _, group := range team.Alliances {
			for _, corporation := range group.Corporations {
				memberRows = append(memberRows, []any{
					teamID,
					corporation.CorporationID,
					group.AllianceID,
					corporation.Kills,
					corporation.Losses,
					corporation.IskDestroyed,
					corporation.IskLost,
				})
			}
		}
		if len(memberRows) > 0 {
			if _, err := tx.CopyFrom(
				ctx,
				pgx.Identifier{"battle_team_members"},
				[]string{
					"battle_team_id", "corporation_id", "alliance_id",
					"kills", "losses", "isk_destroyed", "isk_lost",
				},
				pgx.CopyFromRows(memberRows),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func invalidateConflictBattleCache(ctx context.Context, opts Options) {
	if opts.responseCache == nil {
		return
	}
	opts.responseCache.DeleteMatching(ctx, "shrike:web-api:*:*battle*")
}
