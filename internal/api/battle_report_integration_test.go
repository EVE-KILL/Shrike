package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/battle"
	"github.com/eve-kill/shrike/internal/testfixtures"
)

func TestPreparedBattleReportKeepsCanonicalIDAndRefreshesAtomically(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_battle_report_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	detected := battle.Detected{
		SolarSystemID: 30000142, RegionID: 10000002, Start: start,
		End: start.Add(40 * time.Minute), DurationMinutes: 40,
		Teams: [2]battle.Team{
			{Entries: []battle.TeamEntry{{CorporationID: 70111, AllianceID: 70121}}},
			{Entries: []battle.TeamEntry{{CorporationID: 70112, AllianceID: 70122}}},
		},
	}
	firstID, err := battle.Store(ctx, pool, &detected, nil)
	if err != nil {
		t.Fatal(err)
	}
	later := detected
	later.Start = start.Add(time.Hour)
	later.End = later.Start.Add(30 * time.Minute)
	later.DurationMinutes = 30
	laterID, err := battle.Store(ctx, pool, &later, nil)
	if err != nil {
		t.Fatal(err)
	}
	window, row, err := loadSavedConflictBattleWindow(ctx, pool, int32(firstID))
	if err != nil || window == nil {
		t.Fatalf("load original battle: %v", err)
	}
	detail, err := buildBattleDetail(ctx, Options{DB: pool}, *window, row, true)
	if err != nil {
		t.Fatal(err)
	}
	originalJSON, err := json.Marshal(normalizeJSON(detail))
	if err != nil {
		t.Fatal(err)
	}
	legacyReplay, err := loadBattleReplay(ctx, Options{DB: pool}, window)
	if err != nil {
		t.Fatal(err)
	}

	if count, err := PrepareBattleReports(ctx, pool, firstID, laterID); err != nil || count != 2 {
		t.Fatalf("prepared %d reports: %v", count, err)
	}
	reader := newCachedHomepageReader(pool, nil, nil, "")
	original, err := reader.BattleReport(ctx, firstID)
	if err != nil || original.Losses.LossCount != 2 || len(original.Losses.Filtered) != 0 {
		t.Fatalf("prepared summary = %+v, err = %v", original.Losses, err)
	}
	var oldShape, preparedShape map[string]any
	if err := json.Unmarshal(originalJSON, &oldShape); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(original.Detail, &preparedShape); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oldShape, preparedShape) {
		t.Fatal("prepared report changed the public detail response")
	}
	preparedReplay, err := battleReplayHandler(Options{DB: pool}, false)(ctx, &legacyRequest{
		Params: map[string]string{"id": fmt.Sprint(firstID)},
	})
	if err != nil {
		t.Fatal(err)
	}
	oldReplayJSON, err := json.Marshal(normalizeJSON(legacyReplay.Body))
	if err != nil {
		t.Fatal(err)
	}
	newReplayJSON, err := json.Marshal(normalizeJSON(preparedReplay.Body))
	if err != nil {
		t.Fatal(err)
	}
	var oldReplay, newReplay map[string]any
	if err := json.Unmarshal(oldReplayJSON, &oldReplay); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(newReplayJSON, &newReplay); err != nil || !reflect.DeepEqual(oldReplay, newReplay) {
		t.Fatalf("prepared replay changed the response: %v", err)
	}
	if id, err := battle.Store(ctx, pool, &detected, nil); err != nil || id != firstID {
		t.Fatalf("identical re-detection returned ID %d, want %d, err = %v", id, firstID, err)
	}
	if count, err := PrepareBattleReports(ctx, pool, firstID, laterID); err != nil || count != 0 {
		t.Fatalf("identical re-detection queued %d reports: %v", count, err)
	}

	detected.End = later.End
	detected.DurationMinutes = 90
	reusedID, err := battle.Store(ctx, pool, &detected, nil)
	if err != nil || reusedID != firstID {
		t.Fatalf("re-detection returned ID %d, want %d, err = %v", reusedID, firstID, err)
	}
	stale, err := reader.BattleReport(ctx, laterID)
	if err != nil || stale.Losses.LossCount != 2 {
		t.Fatalf("merged link could not read the existing snapshot: %+v, %v", stale.Losses, err)
	}
	if count, err := PrepareBattleReports(ctx, pool, firstID, laterID); err != nil || count != 1 {
		t.Fatalf("prepared %d revised reports: %v", count, err)
	}
	updated, err := reader.BattleReport(ctx, laterID)
	if err != nil || updated.Losses.LossCount != 3 || len(updated.Losses.Biggest) != 3 {
		t.Fatalf("merged report was not refreshed: %+v, %v", updated.Losses, err)
	}
	var identity struct {
		ID int64 `json:"battle_id"`
	}
	if err := json.Unmarshal(updated.Detail, &identity); err != nil || identity.ID != firstID {
		t.Fatalf("merged report identity = %d, err = %v", identity.ID, err)
	}
	if count, err := PrepareBattleReports(ctx, pool, firstID, laterID); err != nil || count != 0 {
		t.Fatalf("already prepared reports still queued: %d, %v", count, err)
	}
}

func TestKillmailBattleSharesLookupAcrossTabsAndResolvesSavedReport(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_killmail_battle_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	const killID int64 = 2000000001
	start := time.Date(2026, 9, 24, 12, 10, 0, 0, time.UTC)
	for _, table := range []string{"killmails", "killmail_attackers"} {
		if _, err := pool.Exec(ctx, "UPDATE "+table+" SET killmail_time = $1 WHERE killmail_id = 2000000002", start.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	shared := newFakeResponseCacheBackend()
	opts := Options{DB: pool, responseCache: newResponseCache(shared, 1<<20)}
	defer opts.responseCache.Close()
	req := &legacyRequest{Params: map[string]string{"id": fmt.Sprint(killID)}}
	if _, err := killmailBattleHandler(opts)(ctx, req); err != nil {
		t.Fatalf("load killmail battle summary: %v", err)
	}
	// With only the shared cache available, another replica can resolve the
	// same killmail for a tab without rerunning detection.
	other := Options{responseCache: newResponseCache(shared, 1<<20)}
	defer other.responseCache.Close()
	window, err := resolveConflictBattleWindow(ctx, other, req, true)
	if err != nil || window == nil {
		t.Fatalf("load killmail battle from the shared cache: %v", err)
	}
	if len(window.Kills) != 2 || len(window.Attackers) != 2 || len(window.Attackers[killID]) != 3 {
		t.Fatalf("killmail battle included attackers outside the refined fight: %+v", window.Attackers)
	}
	if _, _, err := readKillmailBattle(ctx, opts, 2000000004); err == nil {
		t.Fatal("an isolated killmail unexpectedly resolved to a battle")
	}

	id, err := battle.Store(ctx, pool, &battle.Detected{
		SolarSystemID: 30000142, RegionID: 10000002, Start: start,
		End: start.Add(5 * time.Minute), DurationMinutes: 5,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reader := newCachedHomepageReader(pool, nil, opts.responseCache, opts.Commit)
	opts.responseCache.DeleteMatching(ctx, reader.cacheKey(fmt.Sprintf("killmail-battle:%d", killID)))
	got, err := resolveConflictBattleWindow(ctx, opts, req, true)
	if err != nil || got == nil || got.BattleID != int32(id) {
		t.Fatalf("killmail battle after its cache expires = %+v, %v; want battle %d", got, err, id)
	}
}

func TestKillmailPageAndPublicAPIShareDetailCache(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_killmail_cache_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	service := New(Options{DB: pool, ResponseCacheBytes: 1024 * 1024})
	t.Cleanup(service.Close)
	const id = 2000000001
	page, found, err := service.Homepage().KillmailPage(ctx, id)
	if err != nil || !found || page.TotalValue != 150_000_000 {
		t.Fatalf("source killmail = %v, found = %v, err = %v", page.TotalValue, found, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE killmails SET total_value = 1 WHERE killmail_id = $1", id); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/killmail/2000000001", "/api/killmails/2000000001"} {
		w := httptest.NewRecorder()
		service.Site().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		var data KillmailPageData
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || w.Code != http.StatusOK || data.TotalValue != page.TotalValue {
			t.Fatalf("%s did not reuse the detail document: status %d, ISK %v, err = %v", path, w.Code, data.TotalValue, err)
		}
	}
}
