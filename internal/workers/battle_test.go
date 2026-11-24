package workers

import (
	"context"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/battle"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
)

func TestLateKillmailRefreshesPreparedBattle(t *testing.T) {
	pool := testPool(t)
	if host := pool.Config().ConnConfig.Host; host != "127.0.0.1" && host != "localhost" {
		t.Skip("requires a disposable loopback database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const systemID int32 = 2_000_000_151
	const killID int64 = 2_000_015_100
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day()-2, 0, 0, 0, 0, time.UTC)
	start := day.Add(12 * time.Hour)
	detected := battle.Detected{SolarSystemID: systemID, Start: start, End: start.Add(time.Hour), DurationMinutes: 60}
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := killmail.Delete(clean, pool, killID); err != nil {
			t.Errorf("cleanup late kill: %v", err)
		}
		if _, err := pool.Exec(clean, `
			DELETE FROM battle_teams WHERE battle_id IN (
				SELECT battle_id FROM battles WHERE solar_system_id = $1
			)`, systemID); err != nil {
			t.Errorf("cleanup battle teams: %v", err)
		}
		if _, err := pool.Exec(clean, "DELETE FROM battles WHERE solar_system_id = $1", systemID); err != nil {
			t.Errorf("cleanup generated report: %v", err)
		}
		if _, err := pool.Exec(clean, "DELETE FROM battle_dirty_days WHERE solar_system_id = $1", systemID); err != nil {
			t.Errorf("cleanup dirty day: %v", err)
		}
	})
	id, err := battle.Store(ctx, pool, &detected)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := api.PrepareBattleReports(ctx, pool, 10); n < 1 || err != nil {
		t.Fatalf("prepare baseline report: count=%d err=%v", n, err)
	}
	inserted, err := killmail.Insert(ctx, pool, &killmail.Parsed{Killmail: killmail.Killmail{
		KillmailID: killID, KillmailTime: start.Add(10 * time.Minute), KillmailHash: "late-battle-fixture", SolarSystemID: systemID,
	}})
	if err != nil || !inserted {
		t.Fatalf("insert late kill: inserted=%v err=%v", inserted, err)
	}
	worker := BattleDetectionWorker{Deps: &Deps{Pool: pool, Log: zerolog.Nop()}}
	job := &river.Job[queue.BattleDetectionArgs]{
		Args: queue.BattleDetectionArgs{SystemID: systemID, From: day, To: day.AddDate(0, 0, 1)},
	}
	if err := worker.Work(ctx, job); err != nil {
		t.Fatal(err)
	}
	var pending, dirty bool
	err = pool.QueryRow(ctx, `
		SELECT report_pending, EXISTS (SELECT 1 FROM battle_dirty_days WHERE solar_system_id = $2)
		FROM battles WHERE battle_id = $1`, id, systemID).Scan(&pending, &dirty)
	if err != nil || !pending || dirty {
		t.Fatalf("late refresh left pending=%v dirty=%v err=%v", pending, dirty, err)
	}
	if n, err := api.PrepareBattleReports(ctx, pool, 10); n < 1 || err != nil {
		t.Fatalf("publish refreshed report: count=%d err=%v", n, err)
	}
	reader := api.New(api.Options{DB: pool})
	defer reader.Close()
	report, err := reader.Homepage().BattleReport(ctx, id)
	if err != nil || report.Losses.LossCount != 1 {
		t.Fatalf("refreshed report lost the late kill, count=%d err=%v", report.Losses.LossCount, err)
	}
}
