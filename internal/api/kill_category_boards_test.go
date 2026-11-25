package api

import (
	"context"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/testfixtures"
	"github.com/redis/go-redis/v9"
)

func TestPreparedKillCategoryBoardsSurviveColdCacheAndFailedRefresh(t *testing.T) {
	ctx, pool := localAPITestPool(t, "shrike_kill_category_top_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	const killID int64 = 2000000001
	now := time.Now().UTC()
	for _, table := range []string{"killmails", "killmail_attackers"} {
		if _, err := pool.Exec(ctx, "UPDATE "+table+" SET killmail_time = $1 WHERE killmail_id = $2", now, killID); err != nil {
			t.Fatal(err)
		}
	}
	if err := PrepareKillCategoryTop(ctx, pool, nil, "highsec"); err != nil {
		t.Fatalf("prepare highsec: %v", err)
	}

	reader := NewHomepageReader(pool)
	boards := reader.KillCategoryTopCached(ctx, "highsec")
	if len(boards) != 3 || len(boards["alliances"]) != 1 || boards["alliances"][0].ID != 70121 ||
		len(boards["characters"]) != 2 || len(boards["corporations"]) != 2 {
		t.Fatalf("highsec prepared boards = %#v", boards)
	}
	if other := reader.KillCategoryTopCached(ctx, "lowsec"); len(other) != 0 {
		t.Fatalf("lowsec reused highsec boards: %#v", other)
	}
	if _, err := reader.TopKills(ctx, "lowsec", "alliances", KillCategoryTopDays, KillCategoryTopLimit); err == nil {
		t.Fatal("the default leaderboard read unexpectedly succeeded without a snapshot")
	}
	for entity, entries := range boards {
		legacy, err := loadTopKillEntries(ctx, pool, "highsec", entity, KillCategoryTopLimit, now.AddDate(0, 0, -KillCategoryTopDays))
		if err != nil || len(legacy) != len(entries) {
			t.Fatalf("shared %s leaderboard has %d entries, legacy has %d, err = %v", entity, len(entries), len(legacy), err)
		}
		counts := make(map[int64]float64, len(legacy))
		for _, entry := range legacy {
			counts[entry.ID] = entry.Count
		}
		for _, entry := range entries {
			if entry.Count != counts[entry.ID] {
				t.Fatalf("shared %s leaderboard count for %d = %v, want %v", entity, entry.ID, entry.Count, counts[entry.ID])
			}
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM killmail_attackers WHERE killmail_id = $1", killID); err != nil {
		t.Fatal(err)
	}
	failed, cancel := context.WithCancel(ctx)
	cancel()
	if err := PrepareKillCategoryTop(failed, pool, nil, "highsec"); err == nil {
		t.Fatal("a cancelled refresh unexpectedly succeeded")
	}
	entries, err := reader.TopKills(ctx, "highsec", "alliances", KillCategoryTopDays, KillCategoryTopLimit)
	if err != nil || len(entries) != 1 || entries[0].ID != 70121 {
		t.Fatalf("fragment missed prepared data: %#v, err = %v", entries, err)
	}
	entries, err = reader.TopKills(ctx, "highsec", "alliances", KillCategoryTopDays, KillCategoryTopLimit-1)
	if err != nil || len(entries) != 0 {
		t.Fatalf("nondefault limit reused prepared data: %#v, err = %v", entries, err)
	}
	response, err := topKillsHandler(Options{DB: pool})(ctx, &legacyRequest{
		Query: url.Values{"type": {"highsec"}, "dataType": {"alliances"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries, ok := response.Body.(map[string]any)["entries"].([]TopEntry); !ok || len(entries) != 1 || entries[0].ID != 70121 {
		t.Fatalf("default public API missed the saved board: %#v", response.Body)
	}
	t.Run("Valkey refill", func(t *testing.T) {
		address := os.Getenv("TEST_REDIS_ADDR")
		if address == "" {
			t.Skip("TEST_REDIS_ADDR is not set")
		}
		host, _, err := net.SplitHostPort(address)
		if err != nil || host != "127.0.0.1" && host != "localhost" {
			t.Fatal("leaderboard tests require a disposable loopback Valkey")
		}
		client := redis.NewClient(&redis.Options{Addr: address, DB: 13})
		defer func() { _ = client.Close() }()
		key := killCategoryBoardPrefix + "highsec"
		if err := client.Del(ctx, key).Err(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = client.Del(ctx, key).Err() }()
		reader := &HomepageReader{db: pool, redis: client}
		if entries := reader.KillCategoryTopCached(ctx, "highsec")["alliances"]; len(entries) != 1 || entries[0].ID != 70121 {
			t.Fatalf("cold Valkey read lost the saved board: %#v", entries)
		}
		if n, err := client.Exists(ctx, key).Result(); err != nil || n != 1 {
			t.Fatalf("snapshot did not refill Valkey: exists=%d, err=%v", n, err)
		}
		if entries := reader.KillCategoryTopCached(ctx, "lowsec"); len(entries) != 0 {
			t.Fatalf("lowsec reused a warm highsec board: %#v", entries)
		}
	})
}
