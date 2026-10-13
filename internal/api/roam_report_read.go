package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5"
)

// RoamReport reads mutable reports from the primary, without the HTTP or JSON
// response round trip. The API uses the same enrichment for legacy reports.
func (r *HomepageReader) RoamReport(ctx context.Context, id string) (RoamReport, bool, error) {
	return readRoamReport(ctx, r.db, id)
}

// RoamReportKilllist reuses the report already loaded for the page. API callers
// use the same killlist query after loading their saved report.
func (r *HomepageReader) RoamReportKilllist(ctx context.Context, report RoamReport, query url.Values) (KilllistPage, error) {
	payload, err := loadRoamReportKilllist(ctx, r.db, report, query)
	if err != nil {
		return KilllistPage{}, err
	}
	return decodeKilllistPayload(payload)
}

func loadSavedRoamReport(ctx context.Context, db Database, id string) (roamReport, bool, error) {
	if !validRoamID(id) {
		return roamReport{}, false, nil
	}
	if db == nil {
		return roamReport{}, false, apiError(http.StatusServiceUnavailable, "API database is not configured")
	}
	var raw []byte
	if err := db.QueryRow(ctx, `SELECT report FROM roam_reports WHERE id = $1`, id).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return roamReport{}, false, nil
		}
		return roamReport{}, false, err
	}
	var report roamReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return roamReport{}, false, fmt.Errorf("decode roam report: %w", err)
	}
	return report, true, nil
}

func readRoamReport(ctx context.Context, db Database, id string) (roamReport, bool, error) {
	report, found, err := loadSavedRoamReport(ctx, db, id)
	if err != nil || !found {
		return report, found, err
	}
	if total := report.Summary.IskDestroyed + report.Summary.IskLost; total > 0 {
		report.Summary.Efficiency = math.Round(report.Summary.IskDestroyed/total*10000) / 100
	} else {
		report.Summary.Efficiency = 0
	}
	if len(report.Roster) > 0 && report.Roster[0].Ships == nil {
		killIDs := make([]int32, 0)
		for _, engagement := range report.Engagements {
			for _, kill := range engagement.Killmails {
				killIDs = append(killIDs, kill.KillmailID)
			}
		}
		pilotIDs := make([]int32, 0, len(report.Roster))
		for _, pilot := range report.Roster {
			pilotIDs = append(pilotIDs, pilot.CharacterID)
		}
		var attackers []roamAttacker
		if len(killIDs) > 0 {
			attackers, err = loadRoamAttackers(ctx, db, killIDs, pilotIDs)
			if err != nil {
				return roamReport{}, false, err
			}
		}
		addRoamShips(&report, attackers)
	}
	return report, true, nil
}

func loadRoamReportKilllist(ctx context.Context, db Database, report roamReport, query url.Values) (legacyPayload, error) {
	role := query.Get("role")
	if role == "" {
		role = "kills"
	}
	if role != "kills" && role != "losses" && role != "all" {
		return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid role")
	}
	killIDs := make([]int32, 0)
	for _, engagement := range report.Engagements {
		for _, kill := range engagement.Killmails {
			if role == "all" || role == "kills" && kill.Role == "kill" || role == "losses" && kill.Role == "loss" {
				killIDs = append(killIDs, kill.KillmailID)
			}
		}
	}
	if len(killIDs) == 0 {
		return jsonPayload(KilllistPage{Kills: []KilllistRow{}, TotalPages: 1}), nil
	}
	return loadConflictKilllist(ctx, db, query, []string{"k.killmail_id = ANY($1::int[])"}, []any{killIDs})
}
