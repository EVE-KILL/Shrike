package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/battle"
	"github.com/jackc/pgx/v5"
)

type PreparedBattleReport struct {
	Detail json.RawMessage
	Losses battle.LossSummary
}

func (r *HomepageReader) BattleReport(ctx context.Context, id int64) (PreparedBattleReport, error) {
	return loadPreparedBattleReport(ctx, r.db, id, true)
}

func loadPreparedBattleReport(ctx context.Context, db Database, id int64, withLosses bool) (PreparedBattleReport, error) {
	var result PreparedBattleReport
	var summary []byte
	columns := "report, NULL::jsonb"
	if withLosses {
		columns = "report, loss_summary"
	}
	err := db.QueryRow(ctx, `
		SELECT `+columns+` FROM battle_report_snapshots
		WHERE battle_id = COALESCE(
			(SELECT canonical_battle_id FROM battle_aliases WHERE battle_id = $1), $1
		)`, id).Scan(&result.Detail, &summary)
	if errors.Is(err, pgx.ErrNoRows) {
		return PreparedBattleReport{}, nil
	}
	if err != nil {
		return PreparedBattleReport{}, err
	}
	if withLosses {
		err = json.Unmarshal(summary, &result.Losses)
	}
	return result, err
}

type pendingBattleReport struct {
	id        int32
	updatedAt *time.Time
}

func loadPreparedBattleEvents(ctx context.Context, db Database, id int32) (json.RawMessage, int32, error) {
	var events json.RawMessage
	var system int32
	err := db.QueryRow(ctx, `
		SELECT loss_events, (report->>'solar_system_id')::int FROM battle_report_snapshots
		WHERE battle_id = COALESCE(
			(SELECT canonical_battle_id FROM battle_aliases WHERE battle_id = $1), $1
		)`, id).Scan(&events, &system)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, nil
	}
	return events, system, err
}

// PrepareBattleReports builds a bounded batch, latest changes first. If detection or a
// manual correction changes the source while a report is being prepared, the
// previous snapshot stays readable and the new revision remains pending.
func PrepareBattleReports(ctx context.Context, db MutationDatabase, limit int) (int, error) {
	rows, err := db.Query(ctx, `
		SELECT battle_id, updated_at FROM battles WHERE report_pending
		ORDER BY updated_at DESC NULLS LAST, start_time DESC, battle_id DESC LIMIT $1`, limit)
	if err != nil {
		return 0, err
	}
	pending, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (pendingBattleReport, error) {
		var candidate pendingBattleReport
		err := row.Scan(&candidate.id, &candidate.updatedAt)
		return candidate, err
	})
	if err != nil {
		return 0, err
	}
	count := 0
	var failures []error
	for _, candidate := range pending {
		ok, err := prepareBattleReport(ctx, db, candidate)
		if err != nil {
			failures = append(failures, fmt.Errorf("prepare battle %d: %w", candidate.id, err))
		}
		if ok {
			count++
		}
		if ctx.Err() != nil {
			break
		}
	}
	return count, errors.Join(failures...)
}

func prepareBattleReport(ctx context.Context, db MutationDatabase, candidate pendingBattleReport) (bool, error) {
	window, row, err := loadSavedConflictBattleWindow(ctx, db, candidate.id)
	if err != nil || window == nil || window.BattleID != candidate.id {
		return false, err
	}
	options := Options{DB: db}
	detail, err := buildBattleDetail(ctx, options, *window, row, true)
	if err != nil {
		return false, err
	}
	lossRows, err := loadBattleLossRows(ctx, options, window, true)
	if err != nil {
		return false, err
	}
	events, err := json.Marshal(normalizeJSON(lossRows))
	if err != nil {
		return false, err
	}
	var losses []battle.ReportLoss
	if err := json.Unmarshal(events, &losses); err != nil {
		return false, err
	}
	raw, err := json.Marshal(normalizeJSON(detail))
	if err != nil {
		return false, err
	}
	var sides struct {
		Teams []battle.ReportSide `json:"team_entities"`
	}
	if err := json.Unmarshal(raw, &sides); err != nil {
		return false, err
	}
	summary := battle.SummarizeLosses(window.Start.Format(time.RFC3339Nano), window.End.Format(time.RFC3339Nano), sides.Teams, losses, battle.LossFilter{})
	for _, rows := range [][]battle.ReportLoss{summary.Biggest, summary.Supers} {
		for i := range rows {
			rows[i].X, rows[i].Y, rows[i].Z = nil, nil, nil
		}
	}
	tag, err := db.Exec(ctx, `
		WITH publish AS (
			UPDATE battles SET report_pending = false
			WHERE battle_id = $1 AND report_pending
			  AND updated_at IS NOT DISTINCT FROM $2::timestamptz
			RETURNING battle_id
		)
		INSERT INTO battle_report_snapshots (battle_id, report, loss_summary, loss_events)
		SELECT battle_id, $3::jsonb, $4::jsonb, $5::jsonb FROM publish
		ON CONFLICT (battle_id) DO UPDATE
		SET report = EXCLUDED.report, loss_summary = EXCLUDED.loss_summary, loss_events = EXCLUDED.loss_events`,
		candidate.id, candidate.updatedAt, raw, summary, events)
	return tag.RowsAffected() != 0, err
}
