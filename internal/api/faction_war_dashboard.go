package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/errgroup"
)

const (
	factionWarDefaultDays  = 30
	factionWarMaximumDays  = 365
	factionWarDefaultLimit = 500
	factionWarMaximumLimit = 500
)

type factionWarSide struct {
	ID     int32  `json:"id"`
	Name   string `json:"name"`
	CorpID int32  `json:"corpId"`
	Key    string `json:"-"`
}

type factionWarMatchup struct {
	Slug       string
	ListingKey string
	Side1      factionWarSide
	Side2      factionWarSide
}

// factionWarMatchups is the sole definition of faction-war opponents used by
// every site endpoint. The previous implementation carried four subtly
// different copies of this data.
var factionWarMatchups = map[string]factionWarMatchup{
	"caldari-vs-gallente": {
		Slug:       "caldari-vs-gallente",
		ListingKey: "caldariVsGallente",
		Side1: factionWarSide{
			ID: 500001, Name: "Caldari State", CorpID: 1000035, Key: "caldari",
		},
		Side2: factionWarSide{
			ID: 500004, Name: "Gallente Federation", CorpID: 1000120, Key: "gallente",
		},
	},
	"amarr-vs-minmatar": {
		Slug:       "amarr-vs-minmatar",
		ListingKey: "amarrVsMinmatar",
		Side1: factionWarSide{
			ID: 500003, Name: "Amarr Empire", CorpID: 1000084, Key: "amarr",
		},
		Side2: factionWarSide{
			ID: 500002, Name: "Minmatar Republic", CorpID: 1000051, Key: "minmatar",
		},
	},
}

var factionWarMatchupOrder = []string{
	"caldari-vs-gallente",
	"amarr-vs-minmatar",
}

func registerFactionWarDashboardRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "faction-wars-dashboard",
		Method:      http.MethodGet,
		Path:        "/faction-wars",
		Summary:     "Faction-war matchups",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		factionWarsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-dashboard-detail",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}",
		Summary:     "Faction-war combat summary",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		factionWarBaseHandler(opts),
	))

	// The dashboard route is the consolidated replacement for the historical
	// base + overview request pair. Keep both old paths during the site
	// migration, but give web/ one request that returns their data together.
	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-dashboard",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}/dashboard",
		Summary:     "Complete faction-war dashboard",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		factionWarDashboardHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-overview",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}/overview",
		Summary:     "Faction-war warzone overview",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		factionWarOverviewHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-systems",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}/systems",
		Summary:     "Faction-war systems and map data",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		factionWarSystemsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-members",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}/members",
		Summary:     "Active faction-war members",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=60",
		factionWarMembersHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "faction-war-intel",
		Method:      http.MethodGet,
		Path:        "/faction-war/{matchup}/intel",
		Summary:     "Faction-war activity intelligence",
		Tags:        []string{"faction-war"},
	}, routeJSONCache(
		opts,
		10*time.Minute,
		"public, max-age=60, s-maxage=600, stale-while-revalidate=60",
		factionWarIntelHandler(opts),
	))
}

func factionWarMatchupFromRequest(req *legacyRequest) (factionWarMatchup, error) {
	matchup, ok := factionWarMatchups[req.Param("matchup")]
	if !ok {
		return factionWarMatchup{}, apiError(
			http.StatusNotFound,
			"Unknown faction matchup",
		)
	}
	return matchup, nil
}

func parseFactionWarBoundedInt(raw string, fallback, minimum, maximum int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value <= float64(minimum) {
		return minimum
	}
	if value >= float64(maximum) {
		return maximum
	}
	return int(math.Floor(value))
}

func parseFactionWarDays(query url.Values) int {
	return parseFactionWarBoundedInt(
		query.Get("days"),
		factionWarDefaultDays,
		1,
		factionWarMaximumDays,
	)
}

func factionWarsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		factionIDs := make([]any, 0, len(factionWarMatchupOrder)*2)
		for _, slug := range factionWarMatchupOrder {
			matchup := factionWarMatchups[slug]
			factionIDs = append(factionIDs, matchup.Side1.ID, matchup.Side2.ID)
		}
		rows, err := queryMapsConcurrent(
			ctx,
			opts.DB,
			databaseQuery{
				SQL: `
					SELECT victim_faction_id AS faction_id,
					       COUNT(*)::bigint AS losses,
					       COALESCE(SUM(total_value), 0)::double precision AS isk_lost
					FROM killmails
					WHERE victim_faction_id IN ($1, $2, $3, $4)
					  AND killmail_time >= NOW() - INTERVAL '30 days'
					GROUP BY victim_faction_id`,
				Args: factionIDs,
			},
			databaseQuery{
				SQL: `
					SELECT faction_id, pilots, systems_controlled
					FROM fw_faction_stats
					WHERE faction_id IN ($1, $2, $3, $4)`,
				Args: factionIDs,
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		losses := make(map[int32]factionWarLossStats)
		for _, row := range rows[0] {
			id := int32(factionWarInt(row, "faction_id"))
			losses[id] = factionWarLossStats{
				Losses:  factionWarInt(row, "losses"),
				ISKLost: factionWarFloat(row, "isk_lost"),
			}
		}
		esi := make(map[int32]factionWarESIStats)
		for _, row := range rows[1] {
			id := int32(factionWarInt(row, "faction_id"))
			esi[id] = factionWarESIStats{
				Pilots:            factionWarInt(row, "pilots"),
				SystemsControlled: factionWarInt(row, "systems_controlled"),
			}
		}

		response := make(map[string]any, len(factionWarMatchupOrder))
		for _, slug := range factionWarMatchupOrder {
			matchup := factionWarMatchups[slug]
			response[matchup.ListingKey] = map[string]any{
				matchup.Side1.Key: factionWarListingSide(
					matchup.Side1,
					losses[matchup.Side1.ID],
					losses[matchup.Side2.ID],
					esi[matchup.Side1.ID],
				),
				matchup.Side2.Key: factionWarListingSide(
					matchup.Side2,
					losses[matchup.Side2.ID],
					losses[matchup.Side1.ID],
					esi[matchup.Side2.ID],
				),
			}
		}
		return jsonPayload(response), nil
	}
}

type factionWarLossStats struct {
	Losses  int64
	ISKLost float64
}

type factionWarESIStats struct {
	Pilots            int64
	SystemsControlled int64
}

func factionWarListingSide(
	side factionWarSide,
	own factionWarLossStats,
	opponent factionWarLossStats,
	esi factionWarESIStats,
) map[string]any {
	return map[string]any{
		"faction_id":         side.ID,
		"name":               side.Name,
		"kills":              opponent.Losses,
		"isk_destroyed":      opponent.ISKLost,
		"losses":             own.Losses,
		"isk_lost":           own.ISKLost,
		"pilots":             esi.Pilots,
		"systems_controlled": esi.SystemsControlled,
	}
}

func factionWarBaseHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		days := parseFactionWarDays(req.Query)
		body, err := loadFactionWarBase(ctx, opts.DB, matchup, days)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

func loadFactionWarBase(
	ctx context.Context,
	db Database,
	matchup factionWarMatchup,
	days int,
) (map[string]any, error) {
	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := queryMapsConcurrent(
		ctx,
		db,
		databaseQuery{
			SQL: `
				SELECT victim_faction_id AS faction_id,
				       COUNT(*)::bigint AS losses,
				       COALESCE(SUM(total_value), 0)::double precision AS isk_lost
				FROM killmails
				WHERE victim_faction_id IN ($1, $2)
				  AND killmail_time >= $3
				GROUP BY victim_faction_id`,
			Args: []any{matchup.Side1.ID, matchup.Side2.ID, since},
		},
		databaseQuery{
			SQL: `
				SELECT k.victim_ship_type_id AS ship_type_id,
				       t.name AS ship_name,
				       COUNT(*)::bigint AS count
				FROM killmails k
				LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
				WHERE k.victim_faction_id IN ($1, $2)
				  AND k.killmail_time >= $3
				GROUP BY k.victim_ship_type_id, t.name
				ORDER BY count DESC, k.victim_ship_type_id
				LIMIT 10`,
			Args: []any{matchup.Side1.ID, matchup.Side2.ID, since},
		},
	)
	if err != nil {
		return nil, err
	}

	stats := make(map[int32]factionWarLossStats, 2)
	for _, row := range rows[0] {
		id := int32(factionWarInt(row, "faction_id"))
		stats[id] = factionWarLossStats{
			Losses:  factionWarInt(row, "losses"),
			ISKLost: factionWarFloat(row, "isk_lost"),
		}
	}
	side1 := stats[matchup.Side1.ID]
	side2 := stats[matchup.Side2.ID]

	topShips := make([]map[string]any, 0, len(rows[1]))
	for _, row := range rows[1] {
		name := factionWarString(row, "ship_name")
		if name == "" {
			name = "Unknown"
		}
		topShips = append(topShips, map[string]any{
			"ship_type_id": factionWarInt(row, "ship_type_id"),
			"ship_name":    name,
			"kills":        factionWarInt(row, "count"),
		})
	}

	return map[string]any{
		"matchup": matchup.Slug,
		"days":    days,
		"side1": factionWarCombatSide(
			matchup.Side1,
			side1,
			side2,
		),
		"side2": factionWarCombatSide(
			matchup.Side2,
			side2,
			side1,
		),
		"topShips": topShips,
	}, nil
}

func factionWarCombatSide(
	side factionWarSide,
	own factionWarLossStats,
	opponent factionWarLossStats,
) map[string]any {
	return map[string]any{
		"id":              side.ID,
		"name":            side.Name,
		"corpId":          side.CorpID,
		"kills":           opponent.Losses,
		"isk_destroyed":   opponent.ISKLost,
		"losses":          own.Losses,
		"isk_lost":        own.ISKLost,
		"topCharacters":   []any{},
		"topCorporations": []any{},
		"topAlliances":    []any{},
	}
}

func factionWarOverviewHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := loadFactionWarOverview(
			ctx,
			opts.DB,
			matchup,
			parseFactionWarPeriod(req.Query),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(body), nil
	}
}

func parseFactionWarPeriod(query url.Values) string {
	switch query.Get("period") {
	case "yesterday", "last_week", "active_total":
		return query.Get("period")
	default:
		return "last_week"
	}
}

func loadFactionWarOverview(
	ctx context.Context,
	db Database,
	matchup factionWarMatchup,
	period string,
) (map[string]any, error) {
	side1, side2 := matchup.Side1.ID, matchup.Side2.ID
	rows, err := queryMapsConcurrent(
		ctx,
		db,
		databaseQuery{
			SQL: `
				SELECT faction_id, pilots, systems_controlled,
				       kills_yesterday, kills_last_week, kills_total,
				       vp_yesterday, vp_last_week, vp_total
				FROM fw_faction_stats
				WHERE faction_id IN ($1, $2)`,
			Args: []any{side1, side2},
		},
		databaseQuery{
			SQL: `
				SELECT occupier_faction_id, contested, COUNT(*)::int AS cnt,
				       COALESCE(SUM(victory_points), 0)::bigint AS total_vp,
				       COALESCE(SUM(victory_points_threshold), 0)::bigint AS total_threshold
				FROM fw_systems
				WHERE owner_faction_id IN ($1, $2)
				GROUP BY occupier_faction_id, contested
				ORDER BY occupier_faction_id, contested`,
			Args: []any{side1, side2},
		},
		databaseQuery{
			SQL: `
				WITH deduped AS (
					SELECT DISTINCT ON (
						(h.flipped_at AT TIME ZONE 'UTC')::date,
						h.solar_system_id
					)
						h.solar_system_id,
						h.old_occupier_faction_id,
						h.new_occupier_faction_id,
						h.flipped_at,
						(h.flipped_at AT TIME ZONE 'UTC')::date AS day
					FROM fw_system_history h
					WHERE h.solar_system_id IN (
						SELECT solar_system_id
						FROM fw_systems
						WHERE owner_faction_id IN ($1, $2)
					)
					  AND h.flipped_at >= NOW() - INTERVAL '7 days'
					ORDER BY (h.flipped_at AT TIME ZONE 'UTC')::date DESC,
					         h.solar_system_id,
					         h.flipped_at DESC
				)
				SELECT TO_CHAR(d.day, 'YYYY-MM-DD') AS day,
				       d.solar_system_id, s.system_name,
				       d.old_occupier_faction_id, f1.name AS old_faction_name,
				       d.new_occupier_faction_id, f2.name AS new_faction_name,
				       d.flipped_at
				FROM deduped d
				JOIN solar_systems s ON s.solar_system_id = d.solar_system_id
				JOIN factions f1 ON f1.faction_id = d.old_occupier_faction_id
				JOIN factions f2 ON f2.faction_id = d.new_occupier_faction_id
				ORDER BY d.flipped_at DESC`,
			Args: []any{side1, side2},
		},
		databaseQuery{
			SQL: `
				WITH ranked AS (
					SELECT fl.entity_id AS character_id, fl.amount,
					       c.name AS character_name, c.faction_id,
					       c.corporation_id, co.name AS corporation_name,
					       ROW_NUMBER() OVER (
						       PARTITION BY c.faction_id
						       ORDER BY fl.amount DESC, fl.entity_id
					       ) AS faction_rank
					FROM fw_leaderboards fl
					JOIN characters c ON c.character_id = fl.entity_id
					LEFT JOIN corporations co ON co.corporation_id = c.corporation_id
					WHERE fl.entity_type = 'character'
					  AND fl.metric = 'kills'
					  AND fl.period = $3
					  AND c.faction_id IN ($1, $2)
				)
				SELECT character_id, amount, character_name, faction_id,
				       corporation_id, corporation_name
				FROM ranked
				WHERE faction_rank <= 10
				ORDER BY amount DESC, character_id`,
			Args: []any{side1, side2, period},
		},
		databaseQuery{
			SQL: `
				WITH ranked AS (
					SELECT fl.entity_id AS corporation_id, fl.amount,
					       co.name AS corporation_name, co.faction_id,
					       ROW_NUMBER() OVER (
						       PARTITION BY co.faction_id
						       ORDER BY fl.amount DESC, fl.entity_id
					       ) AS faction_rank
					FROM fw_leaderboards fl
					JOIN corporations co ON co.corporation_id = fl.entity_id
					WHERE fl.entity_type = 'corporation'
					  AND fl.metric = 'kills'
					  AND fl.period = $3
					  AND co.faction_id IN ($1, $2)
				)
				SELECT corporation_id, amount, corporation_name, faction_id
				FROM ranked
				WHERE faction_rank <= 10
				ORDER BY amount DESC, corporation_id`,
			Args: []any{side1, side2, period},
		},
	)
	if err != nil {
		return nil, err
	}

	stats := make(map[int32]map[string]any, 2)
	for _, row := range rows[0] {
		stats[int32(factionWarInt(row, "faction_id"))] = map[string]any{
			"pilots":             factionWarInt(row, "pilots"),
			"systems_controlled": factionWarInt(row, "systems_controlled"),
			"kills_yesterday":    factionWarInt(row, "kills_yesterday"),
			"kills_last_week":    factionWarInt(row, "kills_last_week"),
			"kills_total":        factionWarInt(row, "kills_total"),
			"vp_yesterday":       factionWarInt(row, "vp_yesterday"),
			"vp_last_week":       factionWarInt(row, "vp_last_week"),
			"vp_total":           factionWarInt(row, "vp_total"),
		}
	}

	breakdown := map[int32]*factionWarSystemBreakdown{
		side1: {},
		side2: {},
	}
	var totalSystems int64
	for _, row := range rows[1] {
		id := int32(factionWarInt(row, "occupier_faction_id"))
		count := factionWarInt(row, "cnt")
		totalSystems += count
		bucket := breakdown[id]
		if bucket == nil {
			continue
		}
		bucket.Total += count
		bucket.TotalVP += factionWarInt(row, "total_vp")
		bucket.TotalThreshold += factionWarInt(row, "total_threshold")
		switch factionWarString(row, "contested") {
		case "uncontested":
			bucket.Uncontested += count
		case "contested":
			bucket.Contested += count
		case "vulnerable":
			bucket.Vulnerable += count
		case "captured":
			bucket.Captured += count
		}
	}

	flipDays := make([]map[string]any, 0)
	flipDayIndexes := make(map[string]int)
	for _, row := range rows[2] {
		day := factionWarString(row, "day")
		index, ok := flipDayIndexes[day]
		if !ok {
			index = len(flipDays)
			flipDayIndexes[day] = index
			flipDays = append(flipDays, map[string]any{
				"day":   day,
				"items": []map[string]any{},
			})
		}
		items := flipDays[index]["items"].([]map[string]any)
		items = append(items, map[string]any{
			"solar_system_id":  factionWarInt(row, "solar_system_id"),
			"system_name":      factionWarNullableString(row, "system_name"),
			"old_faction_id":   factionWarInt(row, "old_occupier_faction_id"),
			"old_faction_name": factionWarNullableString(row, "old_faction_name"),
			"new_faction_id":   factionWarInt(row, "new_occupier_faction_id"),
			"new_faction_name": factionWarNullableString(row, "new_faction_name"),
			"flipped_at":       row["flipped_at"],
		})
		flipDays[index]["items"] = items
	}

	characters := map[int32][]map[string]any{
		side1: {},
		side2: {},
	}
	for _, row := range rows[3] {
		id := int32(factionWarInt(row, "faction_id"))
		if _, ok := characters[id]; !ok {
			continue
		}
		characters[id] = append(characters[id], map[string]any{
			"id":               factionWarInt(row, "character_id"),
			"name":             factionWarNullableString(row, "character_name"),
			"kills":            factionWarInt(row, "amount"),
			"corporation_name": factionWarNullableString(row, "corporation_name"),
		})
	}

	corporations := map[int32][]map[string]any{
		side1: {},
		side2: {},
	}
	for _, row := range rows[4] {
		id := int32(factionWarInt(row, "faction_id"))
		if _, ok := corporations[id]; !ok {
			continue
		}
		corporations[id] = append(corporations[id], map[string]any{
			"id":    factionWarInt(row, "corporation_id"),
			"name":  factionWarNullableString(row, "corporation_name"),
			"kills": factionWarInt(row, "amount"),
		})
	}

	var side1Stats any
	if value, ok := stats[side1]; ok {
		side1Stats = value
	}
	var side2Stats any
	if value, ok := stats[side2]; ok {
		side2Stats = value
	}

	return map[string]any{
		"factionStats": map[string]any{
			"side1": side1Stats,
			"side2": side2Stats,
		},
		"warzone": map[string]any{
			"total_systems": totalSystems,
			"side1":         breakdown[side1],
			"side2":         breakdown[side2],
		},
		"flipDays": flipDays,
		"leaderboards": map[string]any{
			"characters": map[string]any{
				"side1": characters[side1],
				"side2": characters[side2],
			},
			"corporations": map[string]any{
				"side1": corporations[side1],
				"side2": corporations[side2],
			},
		},
	}, nil
}

type factionWarSystemBreakdown struct {
	Total          int64 `json:"total"`
	Uncontested    int64 `json:"uncontested"`
	Contested      int64 `json:"contested"`
	Vulnerable     int64 `json:"vulnerable"`
	Captured       int64 `json:"captured"`
	TotalVP        int64 `json:"total_vp"`
	TotalThreshold int64 `json:"total_threshold"`
}

func factionWarDashboardHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		days := parseFactionWarDays(req.Query)
		period := parseFactionWarPeriod(req.Query)

		var base, overview map[string]any
		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() (err error) {
			base, err = loadFactionWarBase(groupCtx, opts.DB, matchup, days)
			return err
		})
		group.Go(func() (err error) {
			overview, err = loadFactionWarOverview(groupCtx, opts.DB, matchup, period)
			return err
		})
		if err := group.Wait(); err != nil {
			return legacyPayload{}, err
		}
		base["overview"] = overview
		return jsonPayload(base), nil
	}
}

func factionWarSystemsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		side1, side2 := matchup.Side1.ID, matchup.Side2.ID
		rows, err := queryMapsConcurrent(
			ctx,
			opts.DB,
			databaseQuery{
				SQL: `
					SELECT fw.solar_system_id, s.system_name,
					       s.x2d, s.z2d, s.security,
					       s.region_id, r.name AS region_name,
					       s.constellation_id, c.constellation_name,
					       fw.owner_faction_id, fw.occupier_faction_id,
					       fw.contested, fw.victory_points,
					       fw.victory_points_threshold
					FROM fw_systems fw
					JOIN solar_systems s ON s.solar_system_id = fw.solar_system_id
					LEFT JOIN regions r ON r.region_id = s.region_id
					LEFT JOIN constellations c ON c.constellation_id = s.constellation_id
					WHERE fw.owner_faction_id IN ($1, $2)
					ORDER BY s.system_name`,
				Args: []any{side1, side2},
			},
			databaseQuery{
				SQL: `
					SELECT j.from_solar_system_id, j.to_solar_system_id
					FROM solar_system_jumps j
					WHERE j.from_solar_system_id IN (
						SELECT solar_system_id
						FROM fw_systems
						WHERE owner_faction_id IN ($1, $2)
					)
					  AND j.to_solar_system_id IN (
						SELECT solar_system_id
						FROM fw_systems
						WHERE owner_faction_id IN ($1, $2)
					)`,
				Args: []any{side1, side2},
			},
			databaseQuery{
				SQL: `
					SELECT solar_system_id, COUNT(*)::int AS kills_24h,
					       COALESCE(SUM(total_value), 0)::double precision AS isk_24h
					FROM killmails
					WHERE solar_system_id IN (
						SELECT solar_system_id
						FROM fw_systems
						WHERE owner_faction_id IN ($1, $2)
					)
					  AND killmail_time >= NOW() - INTERVAL '24 hours'
					GROUP BY solar_system_id`,
				Args: []any{side1, side2},
			},
			databaseQuery{
				SQL: `
					SELECT solar_system_id AS system_id, group_id, x, z
					FROM celestials
					WHERE solar_system_id IN (
						SELECT solar_system_id
						FROM fw_systems
						WHERE owner_faction_id IN ($1, $2)
					)
					  AND group_id IN (6, 7)`,
				Args: []any{side1, side2},
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		activity := make(map[int64]factionWarSystemActivity)
		for _, row := range rows[2] {
			activity[factionWarInt(row, "solar_system_id")] = factionWarSystemActivity{
				Kills: factionWarInt(row, "kills_24h"),
				ISK:   factionWarFloat(row, "isk_24h"),
			}
		}

		systems := make([]map[string]any, 0, len(rows[0]))
		for _, row := range rows[0] {
			id := factionWarInt(row, "solar_system_id")
			current := activity[id]
			systems = append(systems, map[string]any{
				"solar_system_id":          id,
				"system_name":              factionWarNullableString(row, "system_name"),
				"x":                        factionWarFloat(row, "x2d"),
				"y":                        factionWarFloat(row, "z2d"),
				"security":                 factionWarFloat(row, "security"),
				"region_id":                factionWarInt(row, "region_id"),
				"region_name":              factionWarNullableString(row, "region_name"),
				"constellation_id":         factionWarInt(row, "constellation_id"),
				"constellation_name":       factionWarNullableString(row, "constellation_name"),
				"owner_faction_id":         factionWarInt(row, "owner_faction_id"),
				"occupier_faction_id":      factionWarInt(row, "occupier_faction_id"),
				"contested":                factionWarNullableString(row, "contested"),
				"victory_points":           factionWarInt(row, "victory_points"),
				"victory_points_threshold": factionWarInt(row, "victory_points_threshold"),
				"kills_24h":                current.Kills,
				"isk_24h":                  current.ISK,
			})
		}

		jumps := make([][]int64, 0, len(rows[1]))
		for _, row := range rows[1] {
			jumps = append(jumps, []int64{
				factionWarInt(row, "from_solar_system_id"),
				factionWarInt(row, "to_solar_system_id"),
			})
		}

		celestials := make([]map[string]any, 0, len(rows[3]))
		for _, row := range rows[3] {
			celestials = append(celestials, map[string]any{
				"system_id": factionWarInt(row, "system_id"),
				"group_id":  factionWarInt(row, "group_id"),
				"x":         factionWarFloat(row, "x"),
				"z":         factionWarFloat(row, "z"),
			})
		}

		return jsonPayload(map[string]any{
			"systems":    systems,
			"jumps":      jumps,
			"celestials": celestials,
		}), nil
	}
}

type factionWarSystemActivity struct {
	Kills int64
	ISK   float64
}

type factionWarMemberOptions struct {
	Days          int
	Limit         int
	Side          string
	Sort          string
	CorporationID *int32
	AllianceID    *int32
}

func parseFactionWarMemberOptions(query url.Values) factionWarMemberOptions {
	side := query.Get("side")
	switch side {
	case "aggressor", "defender":
	default:
		side = "combined"
	}
	sortBy := query.Get("sort")
	switch sortBy {
	case "kills", "losses", "isk":
	default:
		sortBy = "activity"
	}
	return factionWarMemberOptions{
		Days: parseFactionWarDays(query),
		Limit: parseFactionWarBoundedInt(
			query.Get("limit"),
			factionWarDefaultLimit,
			1,
			factionWarMaximumLimit,
		),
		Side:          side,
		Sort:          sortBy,
		CorporationID: parseFactionWarOptionalID(query.Get("corporationId")),
		AllianceID:    parseFactionWarOptionalID(query.Get("allianceId")),
	}
}

func parseFactionWarOptionalID(raw string) *int32 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil || value <= 0 {
		return nil
	}
	id := int32(value)
	return &id
}

func factionWarMembersHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		options := parseFactionWarMemberOptions(req.Query)
		since := time.Now().UTC().Add(-time.Duration(options.Days) * 24 * time.Hour)
		query, args := factionWarMembersQuery(matchup, options, since)
		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}

		members := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			characterID := factionWarInt(row, "character_id")
			characterName := factionWarString(row, "character_name")
			if characterName == "" {
				characterName = fmt.Sprintf("Character %d", characterID)
			}
			side := "defender"
			if factionWarInt(row, "side") == 1 {
				side = "aggressor"
			}
			members = append(members, map[string]any{
				"character_id":       characterID,
				"character_name":     characterName,
				"side":               side,
				"corporation_id":     factionWarNullableID(row, "corporation_id"),
				"corporation_name":   factionWarNullableString(row, "corp_name"),
				"corporation_ticker": factionWarNullableString(row, "corp_ticker"),
				"alliance_id":        factionWarNullableID(row, "alliance_id"),
				"alliance_name":      factionWarNullableString(row, "alliance_name"),
				"alliance_ticker":    factionWarNullableString(row, "alliance_ticker"),
				"kills":              factionWarInt(row, "kills"),
				"losses":             factionWarInt(row, "losses"),
				"isk_destroyed":      factionWarFloat(row, "isk_destroyed"),
				"isk_lost":           factionWarFloat(row, "isk_lost"),
				"top_ship_type_id":   factionWarNullableID(row, "top_ship_type_id"),
				"top_ship_name":      factionWarNullableString(row, "top_ship_name"),
				"top_ship_count":     factionWarInt(row, "top_ship_count"),
			})
		}

		return jsonPayload(map[string]any{
			"matchup": matchup.Slug,
			"days":    options.Days,
			"side":    options.Side,
			"count":   len(members),
			"limit":   options.Limit,
			"members": members,
		}), nil
	}
}

func factionWarMembersQuery(
	matchup factionWarMatchup,
	options factionWarMemberOptions,
	since time.Time,
) (string, []any) {
	args := []any{matchup.Side1.ID, matchup.Side2.ID, since}
	memberFilters := make([]string, 0, 3)
	switch options.Side {
	case "aggressor":
		memberFilters = append(memberFilters, "COALESCE(k.side, l.side) = 1")
	case "defender":
		memberFilters = append(memberFilters, "COALESCE(k.side, l.side) = 2")
	}
	if options.CorporationID != nil {
		args = append(args, *options.CorporationID)
		memberFilters = append(
			memberFilters,
			fmt.Sprintf("ch.corporation_id = $%d", len(args)),
		)
	}
	if options.AllianceID != nil {
		args = append(args, *options.AllianceID)
		memberFilters = append(
			memberFilters,
			fmt.Sprintf("ch.alliance_id = $%d", len(args)),
		)
	}

	orderBy := "(COALESCE(k.kills, 0) + COALESCE(l.losses, 0)) DESC"
	switch options.Sort {
	case "kills":
		orderBy = "COALESCE(k.kills, 0) DESC"
	case "losses":
		orderBy = "COALESCE(l.losses, 0) DESC"
	case "isk":
		orderBy = "(COALESCE(k.isk_destroyed, 0) + COALESCE(l.isk_lost, 0)) DESC"
	}
	where := ""
	if len(memberFilters) > 0 {
		where = "WHERE " + strings.Join(memberFilters, " AND ")
	}
	args = append(args, options.Limit)
	limitPlaceholder := len(args)

	query := fmt.Sprintf(`
		WITH att AS MATERIALIZED (
			SELECT a.character_id,
			       CASE WHEN a.faction_id = $1 THEN 1 ELSE 2 END AS side,
			       a.ship_type_id,
			       k.killmail_id,
			       k.total_value
			FROM killmails k
			JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
			WHERE k.victim_faction_id IN ($1, $2)
			  AND k.killmail_time >= $3
			  AND a.character_id IS NOT NULL
			  AND a.faction_id IN ($1, $2)
		),
		kill_stats AS (
			SELECT character_id, side,
			       COUNT(DISTINCT killmail_id)::int AS kills,
			       COALESCE(SUM(total_value), 0)::double precision AS isk_destroyed
			FROM att
			GROUP BY character_id, side
		),
		top_ship AS (
			SELECT DISTINCT ON (character_id, side)
			       character_id, side, ship_type_id, ship_count
			FROM (
				SELECT character_id, side, ship_type_id, COUNT(*)::int AS ship_count
				FROM att
				WHERE ship_type_id IS NOT NULL
				GROUP BY character_id, side, ship_type_id
			) ships
			ORDER BY character_id, side, ship_count DESC, ship_type_id
		),
		loss_stats AS (
			SELECT victim_character_id AS character_id,
			       CASE WHEN victim_faction_id = $1 THEN 1 ELSE 2 END AS side,
			       COUNT(*)::int AS losses,
			       COALESCE(SUM(total_value), 0)::double precision AS isk_lost
			FROM killmails
			WHERE victim_faction_id IN ($1, $2)
			  AND killmail_time >= $3
			  AND victim_character_id IS NOT NULL
			GROUP BY victim_character_id, side
		)
		SELECT
			COALESCE(k.character_id, l.character_id) AS character_id,
			COALESCE(k.side, l.side) AS side,
			COALESCE(k.kills, 0)::int AS kills,
			COALESCE(l.losses, 0)::int AS losses,
			COALESCE(k.isk_destroyed, 0)::double precision AS isk_destroyed,
			COALESCE(l.isk_lost, 0)::double precision AS isk_lost,
			ts.ship_type_id AS top_ship_type_id,
			COALESCE(ts.ship_count, 0)::int AS top_ship_count,
			t.name AS top_ship_name,
			ch.name AS character_name,
			ch.corporation_id,
			ch.alliance_id,
			co.name AS corp_name,
			co.ticker AS corp_ticker,
			al.name AS alliance_name,
			al.ticker AS alliance_ticker
		FROM kill_stats k
		FULL OUTER JOIN loss_stats l
			ON k.character_id = l.character_id AND k.side = l.side
		LEFT JOIN top_ship ts
			ON ts.character_id = COALESCE(k.character_id, l.character_id)
		   AND ts.side = COALESCE(k.side, l.side)
		LEFT JOIN inv_types t ON t.type_id = ts.ship_type_id
		LEFT JOIN characters ch
			ON ch.character_id = COALESCE(k.character_id, l.character_id)
		LEFT JOIN corporations co ON co.corporation_id = ch.corporation_id
		LEFT JOIN alliances al ON al.alliance_id = ch.alliance_id
		%s
		ORDER BY %s, COALESCE(k.character_id, l.character_id)
		LIMIT $%d`,
		where,
		orderBy,
		limitPlaceholder,
	)
	return query, args
}

func factionWarIntelHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		matchup, err := factionWarMatchupFromRequest(req)
		if err != nil {
			return legacyPayload{}, err
		}
		days := parseFactionWarDays(req.Query)
		side1, side2 := matchup.Side1.ID, matchup.Side2.ID
		since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		args := []any{side1, side2, since}
		rows, err := queryMapsConcurrent(
			ctx,
			opts.DB,
			databaseQuery{
				SQL: `
					SELECT COUNT(*)::int AS kills,
					       COALESCE(SUM(total_value), 0)::double precision AS isk_destroyed,
					       COUNT(DISTINCT solar_system_id)::int AS systems,
					       COUNT(DISTINCT constellation_id)::int AS constellations,
					       COUNT(DISTINCT region_id)::int AS regions
					FROM killmails
					WHERE victim_faction_id IN ($1, $2)
					  AND killmail_time >= $3`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT k.solar_system_id AS system_id,
					       s.system_name, s.security,
					       s.region_id, r.name AS region_name,
					       COUNT(*)::int AS kills,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
					LEFT JOIN regions r ON r.region_id = s.region_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					GROUP BY k.solar_system_id, s.system_name, s.security,
					         s.region_id, r.name
					ORDER BY kills DESC, k.solar_system_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT k.constellation_id,
					       c.constellation_name,
					       c.region_id, r.name AS region_name,
					       COUNT(*)::int AS kills,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN constellations c ON c.constellation_id = k.constellation_id
					LEFT JOIN regions r ON r.region_id = c.region_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					  AND k.constellation_id IS NOT NULL
					GROUP BY k.constellation_id, c.constellation_name,
					         c.region_id, r.name
					ORDER BY kills DESC, k.constellation_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT k.region_id, r.name AS region_name,
					       COUNT(*)::int AS kills,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN regions r ON r.region_id = k.region_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					  AND k.region_id IS NOT NULL
					GROUP BY k.region_id, r.name
					ORDER BY kills DESC, k.region_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT k.victim_ship_type_id AS ship_type_id,
					       t.name AS ship_name,
					       t.group_id, g.name AS group_name,
					       COUNT(*)::int AS count,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
					LEFT JOIN inv_groups g ON g.group_id = t.group_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					  AND k.victim_ship_type_id IS NOT NULL
					GROUP BY k.victim_ship_type_id, t.name, t.group_id, g.name
					ORDER BY count DESC, k.victim_ship_type_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT a.ship_type_id,
					       t.name AS ship_name,
					       t.group_id, g.name AS group_name,
					       COUNT(*)::int AS count
					FROM killmails k
					JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
					LEFT JOIN inv_types t ON t.type_id = a.ship_type_id
					LEFT JOIN inv_groups g ON g.group_id = t.group_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					  AND a.ship_type_id IS NOT NULL
					GROUP BY a.ship_type_id, t.name, t.group_id, g.name
					ORDER BY count DESC, a.ship_type_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT t.group_id, g.name AS group_name,
					       COUNT(*)::int AS count,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
					LEFT JOIN inv_groups g ON g.group_id = t.group_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					  AND t.group_id IS NOT NULL
					GROUP BY t.group_id, g.name
					ORDER BY count DESC, t.group_id
					LIMIT 20`,
				Args: args,
			},
			databaseQuery{
				SQL: `
					SELECT CASE
						       WHEN s.security >= 0.5 THEN 'highsec'
						       WHEN s.security > 0 AND s.security < 0.5 THEN 'lowsec'
						       WHEN s.security <= 0 AND s.security > -1 THEN 'nullsec'
						       WHEN s.security <= -1 THEN 'wormhole'
						       ELSE 'unknown'
					       END AS sec_class,
					       COUNT(*)::int AS kills,
					       COALESCE(SUM(k.total_value), 0)::double precision AS isk_destroyed
					FROM killmails k
					LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
					WHERE k.victim_faction_id IN ($1, $2)
					  AND k.killmail_time >= $3
					GROUP BY sec_class
					ORDER BY kills DESC, sec_class`,
				Args: args,
			},
		)
		if err != nil {
			return legacyPayload{}, err
		}

		summary := map[string]any{}
		if len(rows[0]) > 0 {
			summary = rows[0][0]
		}
		return jsonPayload(map[string]any{
			"matchup": matchup.Slug,
			"days":    days,
			"summary": map[string]any{
				"kills":          factionWarInt(summary, "kills"),
				"isk_destroyed":  factionWarFloat(summary, "isk_destroyed"),
				"systems":        factionWarInt(summary, "systems"),
				"constellations": factionWarInt(summary, "constellations"),
				"regions":        factionWarInt(summary, "regions"),
			},
			"top_systems":           factionWarIntelSystems(rows[1]),
			"top_constellations":    factionWarIntelConstellations(rows[2]),
			"top_regions":           factionWarIntelRegions(rows[3]),
			"ships_destroyed":       factionWarIntelShips(rows[4], true),
			"ships_used":            factionWarIntelShips(rows[5], false),
			"ship_groups_destroyed": factionWarIntelShipGroups(rows[6]),
			"security_breakdown":    factionWarIntelSecurity(rows[7]),
		}), nil
	}
}

func factionWarIntelSystems(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := factionWarInt(row, "system_id")
		name := factionWarString(row, "system_name")
		if name == "" {
			name = fmt.Sprintf("System %d", id)
		}
		result = append(result, map[string]any{
			"system_id":     id,
			"system_name":   name,
			"security":      factionWarNullableFloat(row, "security"),
			"region_id":     factionWarNullableID(row, "region_id"),
			"region_name":   factionWarNullableString(row, "region_name"),
			"kills":         factionWarInt(row, "kills"),
			"isk_destroyed": factionWarFloat(row, "isk_destroyed"),
		})
	}
	return result
}

func factionWarIntelConstellations(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := factionWarInt(row, "constellation_id")
		name := factionWarString(row, "constellation_name")
		if name == "" {
			name = fmt.Sprintf("Constellation %d", id)
		}
		result = append(result, map[string]any{
			"constellation_id":   id,
			"constellation_name": name,
			"region_id":          factionWarNullableID(row, "region_id"),
			"region_name":        factionWarNullableString(row, "region_name"),
			"kills":              factionWarInt(row, "kills"),
			"isk_destroyed":      factionWarFloat(row, "isk_destroyed"),
		})
	}
	return result
}

func factionWarIntelRegions(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := factionWarInt(row, "region_id")
		name := factionWarString(row, "region_name")
		if name == "" {
			name = fmt.Sprintf("Region %d", id)
		}
		result = append(result, map[string]any{
			"region_id":     id,
			"region_name":   name,
			"kills":         factionWarInt(row, "kills"),
			"isk_destroyed": factionWarFloat(row, "isk_destroyed"),
		})
	}
	return result
}

func factionWarIntelShips(
	rows []map[string]any,
	includeISK bool,
) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := factionWarInt(row, "ship_type_id")
		name := factionWarString(row, "ship_name")
		if name == "" {
			name = fmt.Sprintf("Type %d", id)
		}
		item := map[string]any{
			"ship_type_id": id,
			"ship_name":    name,
			"group_id":     factionWarNullableID(row, "group_id"),
			"group_name":   factionWarNullableString(row, "group_name"),
			"count":        factionWarInt(row, "count"),
		}
		if includeISK {
			item["isk_destroyed"] = factionWarFloat(row, "isk_destroyed")
		}
		result = append(result, item)
	}
	return result
}

func factionWarIntelShipGroups(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := factionWarInt(row, "group_id")
		name := factionWarString(row, "group_name")
		if name == "" {
			name = fmt.Sprintf("Group %d", id)
		}
		result = append(result, map[string]any{
			"group_id":      id,
			"group_name":    name,
			"count":         factionWarInt(row, "count"),
			"isk_destroyed": factionWarFloat(row, "isk_destroyed"),
		})
	}
	return result
}

func factionWarIntelSecurity(rows []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, map[string]any{
			"sec_class":     factionWarNullableString(row, "sec_class"),
			"kills":         factionWarInt(row, "kills"),
			"isk_destroyed": factionWarFloat(row, "isk_destroyed"),
		})
	}
	return result
}

func factionWarInt(row map[string]any, key string) int64 {
	value, _ := int64Value(row[key])
	return value
}

func factionWarFloat(row map[string]any, key string) float64 {
	value, _ := float64Value(row[key])
	return value
}

func factionWarString(row map[string]any, key string) string {
	value, _ := stringValue(row[key])
	return value
}

func factionWarNullableString(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := stringValue(row[key])
	if !ok {
		return nil
	}
	return value
}

func factionWarNullableID(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := int64Value(row[key])
	if !ok || value == 0 {
		return nil
	}
	return value
}

func factionWarNullableFloat(row map[string]any, key string) any {
	if row[key] == nil {
		return nil
	}
	value, ok := float64Value(row[key])
	if !ok {
		return nil
	}
	return value
}
