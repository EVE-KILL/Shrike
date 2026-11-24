package api

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/testfixtures"
	"github.com/redis/go-redis/v9"
)

func TestPreparedKillCategoryBoardsServeFromValkey(t *testing.T) {
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" && host != "localhost" {
		t.Fatal("leaderboard tests require a disposable loopback Valkey")
	}
	ctx, pool := localAPITestPool(t, "shrike_kill_category_top_")
	if err := testfixtures.SeedBattleGenerator(ctx, pool); err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(&redis.Options{Addr: address, DB: 13})
	defer func() { _ = client.Close() }()
	defer func() { _ = client.Del(ctx, killCategoryBoardCursor, killCategoryBoardPrefix+"highsec").Err() }()

	// The first two registry entries are latest and highsec.
	if err := client.Set(ctx, killCategoryBoardCursor, 1, 0).Err(); err != nil {
		t.Fatal(err)
	}
	const killID int64 = 2000000001
	now := time.Now().UTC()
	for _, table := range []string{"killmails", "killmail_attackers"} {
		if _, err := pool.Exec(ctx, "UPDATE "+table+" SET killmail_time = $1 WHERE killmail_id = $2", now, killID); err != nil {
			t.Fatal(err)
		}
	}
	if kind, err := PrepareNextKillCategoryTop(ctx, pool, client); err != nil || kind != "highsec" {
		t.Fatalf("prepared %q, err = %v", kind, err)
	}

	service := New(Options{DB: pool, Cache: client})
	defer service.Close()
	reader := service.Homepage()
	boards := reader.KillCategoryTopCached(ctx, "highsec")
	if len(boards) != 3 || len(boards["alliances"]) != 1 || boards["alliances"][0].ID != 70121 ||
		len(boards["characters"]) != 2 || len(boards["corporations"]) != 2 {
		t.Fatalf("highsec prepared boards = %#v", boards)
	}
	if other := reader.KillCategoryTopCached(ctx, "lowsec"); len(other) != 0 {
		t.Fatalf("lowsec reused highsec boards: %#v", other)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM killmail_attackers WHERE killmail_id = $1", killID); err != nil {
		t.Fatal(err)
	}
	entries, err := reader.TopKills(ctx, "highsec", "alliances", KillCategoryTopDays, KillCategoryTopLimit)
	if err != nil || len(entries) != 1 || entries[0].ID != 70121 {
		t.Fatalf("fragment missed prepared data: %#v, err = %v", entries, err)
	}
	entries, err = reader.TopKills(ctx, "highsec", "alliances", KillCategoryTopDays, KillCategoryTopLimit-1)
	if err != nil || len(entries) != 0 {
		t.Fatalf("nondefault limit reused prepared data: %#v, err = %v", entries, err)
	}
}
