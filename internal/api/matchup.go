package api

import (
	"context"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/errgroup"
)

const (
	matchupWindowDays = 90
	matchupTopFits    = 3
	matchupMinSample  = 10
)

var matchupPodTypeIDs = map[int64]struct{}{670: {}, 33328: {}}

func registerMatchupRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "ship-matchup",
		Method:      http.MethodGet,
		Path:        "/matchup",
		Summary:     "Solo ship matchup",
		Description: "Compares both directions of a solo hull matchup over 90 days.",
		Tags:        []string{"ships", "statistics"},
	}, routeJSONCache(
		opts,
		time.Hour,
		"public, max-age=300, s-maxage=3600, stale-while-revalidate=3600",
		matchupHandler(opts),
	))
}

func matchupHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		attackerID, attackerErr := parseID(req.Query.Get("attacker"))
		victimID, victimErr := parseID(req.Query.Get("victim"))
		if attackerErr != nil || victimErr != nil ||
			attackerID <= 0 || victimID <= 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				"attacker and victim must be positive ship type IDs",
			)
		}
		if _, pod := matchupPodTypeIDs[attackerID]; pod {
			return jsonPayload(emptyMatchup(attackerID, victimID)), nil
		}
		if _, pod := matchupPodTypeIDs[victimID]; pod {
			return jsonPayload(emptyMatchup(attackerID, victimID)), nil
		}

		cutoff := time.Now().UTC().Add(-matchupWindowDays * 24 * time.Hour)
		mirror := attackerID == victimID
		var attackerWins, victimWins int64
		group, groupCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			value, err := loadMatchupWins(
				groupCtx, opts.DB, victimID, attackerID, cutoff,
			)
			attackerWins = value
			return err
		})
		if !mirror {
			group.Go(func() error {
				value, err := loadMatchupWins(
					groupCtx, opts.DB, attackerID, victimID, cutoff,
				)
				victimWins = value
				return err
			})
		}
		if err := group.Wait(); err != nil {
			return legacyPayload{}, err
		}

		sample := attackerWins + victimWins
		if mirror {
			sample = attackerWins
		}
		enough := sample >= matchupMinSample
		winRate := 0.0
		switch {
		case mirror:
			winRate = 50
		case sample > 0:
			winRate = roundMatchup(float64(attackerWins) / float64(sample) * 100)
		}

		topFits := []map[string]any{}
		if enough {
			families, err := loadMatchupFitFamilies(
				ctx, opts.DB, attackerID, victimID, cutoff,
			)
			if err != nil {
				return legacyPayload{}, err
			}
			topFits, err = loadMatchupFitModules(
				ctx, opts.DB, families, attackerWins,
			)
			if err != nil {
				return legacyPayload{}, err
			}
		}
		return jsonPayload(matchupResult(
			attackerID, victimID, attackerWins, victimWins,
			sample, winRate, enough, topFits,
		)), nil
	}
}

func loadMatchupWins(
	ctx context.Context,
	db Database,
	loserID, winnerID int64,
	cutoff time.Time,
) (int64, error) {
	row, err := queryMap(ctx, db, `
		SELECT COUNT(*)::bigint AS wins
		FROM killmails killmail
		WHERE killmail.is_solo IS TRUE
		  AND killmail.killmail_time >= $3
		  AND killmail.victim_ship_type_id = $1
		  AND EXISTS (
			  SELECT 1
			  FROM killmail_attackers attacker
			  WHERE attacker.killmail_id = killmail.killmail_id
			    AND attacker.character_id IS NOT NULL
			    AND attacker.ship_type_id = $2
		  )`,
		loserID, winnerID, cutoff,
	)
	if err != nil {
		return 0, err
	}
	wins, _ := int64Value(row["wins"])
	return wins, nil
}

func loadMatchupFitFamilies(
	ctx context.Context,
	db Database,
	attackerID, victimID int64,
	cutoff time.Time,
) ([]map[string]any, error) {
	return queryMaps(ctx, db, `
		WITH fit_uses AS (
			SELECT killmail_fit.fit_hash, fitting.family_hash,
			       COUNT(*)::int AS uses,
			       MAX(killmail_fit.kill_time) AS last_used
			FROM killmail_fittings killmail_fit
			JOIN fittings fitting
			  ON fitting.fit_hash = killmail_fit.fit_hash
			WHERE killmail_fit.ship_type_id = $1
			  AND killmail_fit.kill_time >= $3
			  AND EXISTS (
				  SELECT 1 FROM killmails killmail
				  WHERE killmail.killmail_id = killmail_fit.killmail_id
				    AND killmail.is_solo IS TRUE
			  )
			  AND EXISTS (
				  SELECT 1 FROM killmail_attackers attacker
				  WHERE attacker.killmail_id = killmail_fit.killmail_id
				    AND attacker.character_id IS NOT NULL
				    AND attacker.ship_type_id = $2
			  )
			GROUP BY killmail_fit.fit_hash, fitting.family_hash
		),
		canonical AS (
			SELECT DISTINCT ON (family_hash)
			       family_hash, fit_hash AS canonical_fit_hash,
			       last_used AS canonical_last_used
			FROM fit_uses
			ORDER BY family_hash, uses DESC, fit_hash
		),
		family_totals AS (
			SELECT family_hash, SUM(uses)::int AS total_uses
			FROM fit_uses
			GROUP BY family_hash
		)
		SELECT canonical.family_hash, canonical.canonical_fit_hash,
		       family_totals.total_uses AS uses
		FROM canonical
		JOIN family_totals
		  ON family_totals.family_hash = canonical.family_hash
		ORDER BY family_totals.total_uses DESC,
		         canonical.canonical_last_used DESC
		LIMIT $4`,
		victimID, attackerID, cutoff, matchupTopFits,
	)
}

func loadMatchupFitModules(
	ctx context.Context,
	db Database,
	families []map[string]any,
	attackerWins int64,
) ([]map[string]any, error) {
	if len(families) == 0 {
		return []map[string]any{}, nil
	}
	hashes := make([]string, 0, len(families))
	for _, family := range families {
		hash := strings.TrimSpace(matchupString(family, "canonical_fit_hash"))
		if hash != "" {
			hashes = append(hashes, hash)
		}
	}
	rows, err := queryMaps(ctx, db, `
		SELECT item.fit_hash, item.slot_group, item.ordinal,
		       item.type_id, type.name
		FROM fitting_items item
		LEFT JOIN inv_types type ON type.type_id = item.type_id
		WHERE item.fit_hash = ANY($1::text[])
		  AND item.slot_group BETWEEN 1 AND 5
		ORDER BY item.fit_hash, item.slot_group, item.ordinal`,
		hashes,
	)
	if err != nil {
		return nil, err
	}
	modules := make(map[string][]map[string]any, len(hashes))
	for _, row := range rows {
		hash := matchupString(row, "fit_hash")
		modules[hash] = append(modules[hash], map[string]any{
			"slot_group": matchupInt(row, "slot_group"),
			"type_id":    matchupInt(row, "type_id"),
			"name":       row["name"],
		})
	}

	result := make([]map[string]any, 0, len(families))
	for _, family := range families {
		hash := matchupString(family, "canonical_fit_hash")
		uses := matchupInt(family, "uses")
		percentage := 0.0
		if attackerWins > 0 {
			percentage = roundMatchup(
				float64(uses) / float64(attackerWins) * 100,
			)
		}
		items := modules[hash]
		if items == nil {
			items = []map[string]any{}
		}
		result = append(result, map[string]any{
			"family_hash": matchupString(family, "family_hash"),
			"uses":        uses,
			"pct":         percentage,
			"modules":     items,
		})
	}
	return result, nil
}

func emptyMatchup(attackerID, victimID int64) map[string]any {
	return matchupResult(
		attackerID, victimID, 0, 0, 0, 0, false, []map[string]any{},
	)
}

func matchupResult(
	attackerID, victimID, attackerWins, victimWins, sample int64,
	winRate float64,
	enough bool,
	topFits []map[string]any,
) map[string]any {
	if topFits == nil {
		topFits = []map[string]any{}
	}
	return map[string]any{
		"attacker_ship_type_id": attackerID,
		"victim_ship_type_id":   victimID,
		"window_days":           matchupWindowDays,
		"min_sample":            matchupMinSample,
		"mirror":                attackerID == victimID,
		"attacker_wins":         attackerWins,
		"victim_wins":           victimWins,
		"sample":                sample,
		"attacker_win_rate":     winRate,
		"enough":                enough,
		"top_fits":              topFits,
	}
}

func roundMatchup(value float64) float64 {
	return math.Round(value*10) / 10
}

func matchupInt(row map[string]any, key string) int64 {
	value, _ := int64Value(row[key])
	return value
}

func matchupString(row map[string]any, key string) string {
	value, _ := stringValue(row[key])
	return value
}
