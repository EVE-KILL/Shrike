package wars

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNeedsWarReplayMatchesProcessingState(t *testing.T) {
	const warID int32 = 12345

	tests := []struct {
		name  string
		state storedWarState
		want  bool
	}{
		{"not stored", storedWarState{}, true},
		{"unassigned", storedWarState{completed: killmail.AllEffects}, true},
		{"different war", storedWarState{warID: 999, completed: killmail.AllEffects}, true},
		{"same war no ledger", storedWarState{warID: warID}, true},
		{"same war effect pending", storedWarState{
			warID: warID, completed: killmail.AllEffectsExceptWar,
		}, true},
		{"same war effect complete", storedWarState{
			warID: warID, completed: killmail.EffectWarInteractions,
		}, false},
	}

	for _, tc := range tests {
		if got := needsWarReplay(warID, tc.state); got != tc.want {
			t.Errorf("%s: needsWarReplay = %v, want %v", tc.name, got, tc.want)
		}
	}
}

const warStoreTestDSN = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"

func warStorePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = warStoreTestDSN
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStoreAlliesDoesNotDuplicateLogicalAllies(t *testing.T) {
	pool := warStorePool(t)
	ctx := context.Background()
	const warID int32 = 2_140_002_000

	cleanup := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM war_allies WHERE war_id = $1`, warID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM wars WHERE war_id = $1`, warID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := pool.Exec(ctx, `INSERT INTO wars (war_id) VALUES ($1)`, warID); err != nil {
		t.Fatal(err)
	}

	var war esi.War
	if err := json.Unmarshal([]byte(`{
        "allies": [
            {"alliance_id": 99000001},
            {"corporation_id": 98000001}
        ]
    }`), &war); err != nil {
		t.Fatal(err)
	}

	first, err := StoreAllies(ctx, pool, warID, war)
	if err != nil {
		t.Fatal(err)
	}
	second, err := StoreAllies(ctx, pool, warID, war)
	if err != nil {
		t.Fatal(err)
	}
	if first != 2 || second != 0 {
		t.Errorf("writes = %d then %d, want 2 then 0", first, second)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM war_allies WHERE war_id = $1`, warID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("%d ally rows after storing the same response twice, want 2", count)
	}
}
