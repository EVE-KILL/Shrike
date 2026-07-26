package entities

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/esi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// These tests write real rows through real SQL, because the interesting part is
// the SQL: which columns an upsert leaves alone, what a 404 records, when the
// history sync marker suppresses a fetch. A mocked database would assert only
// that the code calls the functions it calls.
//
// They run against the local stack's `evekill` database — the schema is large
// enough that creating a scratch database per test would mean running the whole
// baseline migration each time. To stay out of the way of real data, every test
// uses ids in a reserved band and deletes them afterwards.
//
// Skipped when Postgres or Redis is unreachable.

// testIDBase sits above every id CCP has issued, so these rows cannot collide
// with real entities imported from ESI.
const testIDBase int32 = 2_100_000_000

const (
	defaultTestDSN   = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"
	defaultTestRedis = "127.0.0.1:6379"

	// testRedisDB keeps this package's ESI state off the database
	// internal/esi's tests use. Both clear the whole `esi:*` keyspace, and
	// `go test ./...` runs them in parallel.
	testRedisDB = 14
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// reserve returns a unique id in the test band and removes any trace of it,
// before and after.
func reserve(t *testing.T, pool *pgxpool.Pool, offset int32) int32 {
	t.Helper()
	id := testIDBase + offset

	cleanup := func() {
		ctx := context.Background()
		for _, stmt := range []string{
			`DELETE FROM character_corporation_history WHERE character_id = $1`,
			`DELETE FROM corporation_alliance_history WHERE corporation_id = $1`,
			`DELETE FROM characters WHERE character_id = $1`,
			`DELETE FROM corporations WHERE corporation_id = $1`,
			`DELETE FROM alliances WHERE alliance_id = $1`,
		} {
			_, _ = pool.Exec(ctx, stmt, id)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	return id
}

// fakeESIClient points a real ESI client at a fake server, so the pipeline runs
// in full — cache, bucket, coordination — with responses the test controls.
func fakeESIClient(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) *esi.Client {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = defaultTestRedis
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: testRedisDB, MaxRetries: -1, DialTimeout: 2 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		t.Skipf("no test redis at %s: %v", addr, err)
	}

	// The ESI cache is keyed by URL and the fake server's port changes per test,
	// so entries cannot leak between tests. The buckets and the pause flag can,
	// which is why they are cleared.
	clearKeys(t, rdb)
	t.Cleanup(func() {
		clearKeys(t, rdb)
		rdb.Close()
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for prefix, handler := range routes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				handler(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client := esi.NewForTest(srv.URL, "shrike-test/1.0", rdb, rdb)
	t.Cleanup(client.Close)
	return client
}

func clearKeys(t *testing.T, rdb *redis.Client) {
	t.Helper()
	ctx := context.Background()
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, "esi:*", 500).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = rdb.Del(ctx, keys...).Err()
		}
		if next == 0 {
			return
		}
		cursor = next
	}
}

func jsonHandler(body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Expires", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		_, _ = w.Write([]byte(body))
	}
}

func statusHandler(code int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }
}

// --- Characters ---

func TestCharacterUpsertWritesEveryColumn(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 1)
	corpID := reserve(t, pool, 2)

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/characters/": jsonHandler(fmt.Sprintf(`{
            "name": "Test Pilot",
            "description": "a description",
            "birthday": "2015-03-24T11:00:00Z",
            "gender": "male",
            "race_id": 4,
            "bloodline_id": 7,
            "security_status": -1.5,
            "title": "Chief",
            "corporation_id": %d,
            "alliance_id": 0,
            "faction_id": 0
        }`, corpID)),
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.Character(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 200 || res.Name != "Test Pilot" {
		t.Fatalf("result = %+v", res)
	}

	var name, gender, title string
	var raceID, bloodlineID, storedCorp int32
	var allianceID, factionID *int32
	var security float64
	var birthday *time.Time
	var deleted bool
	err = pool.QueryRow(context.Background(), `
        SELECT name, gender, title, race_id, bloodline_id, corporation_id,
               alliance_id, faction_id, security_status, birthday, coalesce(deleted,false)
        FROM characters WHERE character_id = $1`, id).
		Scan(&name, &gender, &title, &raceID, &bloodlineID, &storedCorp,
			&allianceID, &factionID, &security, &birthday, &deleted)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if name != "Test Pilot" || gender != "male" || title != "Chief" {
		t.Errorf("text columns: name=%q gender=%q title=%q", name, gender, title)
	}
	if raceID != 4 || bloodlineID != 7 || storedCorp != corpID {
		t.Errorf("ids: race=%d bloodline=%d corp=%d", raceID, bloodlineID, storedCorp)
	}
	// Zero means absent throughout the codebase; these must be NULL, not 0.
	if allianceID != nil || factionID != nil {
		t.Errorf("absent ids stored as zero: alliance=%v faction=%v", allianceID, factionID)
	}
	if security > -1.49 || security < -1.51 {
		t.Errorf("security = %v, want -1.5", security)
	}
	if birthday == nil || birthday.UTC().Year() != 2015 {
		t.Errorf("birthday = %v", birthday)
	}
	if deleted {
		t.Error("a live character was stored as deleted")
	}

	// A player corporation that is not yet known must be queued for fetching.
	if len(res.Cascade.Corporations) != 1 || res.Cascade.Corporations[0] != corpID {
		t.Errorf("corporation cascade = %v, want [%d]", res.Cascade.Corporations, corpID)
	}
	// History has never been synced, so it must be queued too.
	if len(res.Cascade.CharacterHistories) != 1 {
		t.Errorf("history cascade = %v", res.Cascade.CharacterHistories)
	}
}

// A biomassed character must be recorded, not skipped: the id keeps appearing on
// old killmails forever, and without a row it would be refetched every time.
func TestDeletedCharacterIsRecorded(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 3)

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/characters/": statusHandler(http.StatusNotFound),
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.Character(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Deleted {
		t.Error("a 404 did not record the character as deleted")
	}

	var name string
	var deleted bool
	if err := pool.QueryRow(context.Background(),
		`SELECT name, deleted FROM characters WHERE character_id = $1`, id).Scan(&name, &deleted); err != nil {
		t.Fatalf("no row was written for a deleted character: %v", err)
	}
	if !deleted {
		t.Error("the row does not carry the deleted flag")
	}
	if name == "" {
		t.Error("name is NOT NULL and must be filled with something")
	}
}

// NPC corporations are where every new character starts. Fetching them would
// spend the whole corporation budget on the same few hundred ids.
func TestNPCCorporationIsNotCascaded(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 4)

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		// 1000045 is an NPC corporation, below the player threshold.
		"/latest/characters/": jsonHandler(`{"name":"Newbie","corporation_id":1000045}`),
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.Character(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Cascade.Corporations) != 0 {
		t.Errorf("an NPC corporation was queued for fetching: %v", res.Cascade.Corporations)
	}
}

func TestIsPlayerCorporation(t *testing.T) {
	if IsPlayerCorporation(1000045) {
		t.Error("an NPC corporation was classified as a player corporation")
	}
	if IsPlayerCorporation(PlayerCorporationIDMin - 1) {
		t.Error("the boundary is off by one")
	}
	if !IsPlayerCorporation(PlayerCorporationIDMin) {
		t.Error("the first player corporation id was rejected")
	}
	if !IsPlayerCorporation(98187159) {
		t.Error("a real player corporation was rejected")
	}
}

// --- Corporations ---

// The 2026-07-21 schema expresses tax as a percentage where the old one used a
// fraction. Storing the wrong one shows a 10% corporation as taxing 1000%.
func TestCorporationTaxRateIsNormalised(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 5)

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/corporations/": jsonHandler(`{
            "name": "Test Corp",
            "ticker": "TEST",
            "member_count": 42,
            "tax_rates": {"isk": 10.0, "loyalty_point": 25.0},
            "enlisted_faction_id": 500001,
            "state": "active",
            "type": "player_owned",
            "palette": {"main_color": "#ff0000"}
        }`),
	})

	r := &Refresher{Pool: pool, ESI: client}
	if _, err := r.Corporation(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	var taxRate float64
	var lpTaxRate *float64
	var memberCount, factionID int32
	var state, corpType string
	var palette []byte
	err := pool.QueryRow(context.Background(), `
        SELECT tax_rate, lp_tax_rate, member_count, faction_id, state, type, palette
        FROM corporations WHERE corporation_id = $1`, id).
		Scan(&taxRate, &lpTaxRate, &memberCount, &factionID, &state, &corpType, &palette)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if taxRate < 0.099 || taxRate > 0.101 {
		t.Errorf("tax_rate = %v, want 0.1 — a percentage was stored as a fraction", taxRate)
	}
	if lpTaxRate == nil || *lpTaxRate < 0.249 || *lpTaxRate > 0.251 {
		t.Errorf("lp_tax_rate = %v, want 0.25", lpTaxRate)
	}
	if memberCount != 42 {
		t.Errorf("member_count = %d", memberCount)
	}
	// enlisted_faction_id is the new name for faction_id.
	if factionID != 500001 {
		t.Errorf("faction_id = %d, want the enlisted_faction_id value", factionID)
	}
	if state != "active" || corpType != "player_owned" {
		t.Errorf("state=%q type=%q", state, corpType)
	}
	if len(palette) == 0 || !strings.Contains(string(palette), "main_color") {
		t.Errorf("palette was not stored as jsonb: %s", palette)
	}
}

func TestCorporationLegacyTaxRateIsKept(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 6)

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/corporations/": jsonHandler(`{
            "name": "Old Corp", "ticker": "OLD", "member_count": 1,
            "tax_rate": 0.05, "faction_id": 500002
        }`),
	})

	r := &Refresher{Pool: pool, ESI: client}
	if _, err := r.Corporation(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	var taxRate float64
	var lpTaxRate *float64
	var factionID int32
	if err := pool.QueryRow(context.Background(),
		`SELECT tax_rate, lp_tax_rate, faction_id FROM corporations WHERE corporation_id = $1`, id).
		Scan(&taxRate, &lpTaxRate, &factionID); err != nil {
		t.Fatal(err)
	}

	if taxRate < 0.049 || taxRate > 0.051 {
		t.Errorf("tax_rate = %v, want the 0.05 fraction unchanged", taxRate)
	}
	// The legacy shape has no loyalty-point tax, which is not the same as zero.
	if lpTaxRate != nil {
		t.Errorf("lp_tax_rate = %v, want NULL for the legacy shape", *lpTaxRate)
	}
	if factionID != 500002 {
		t.Errorf("faction_id = %d", factionID)
	}
}

// --- Alliances ---

// creator_id and date_founded are historical facts. ESI has been known to return
// them as null, and overwriting would lose them permanently.
func TestAllianceUpsertPreservesFoundingFacts(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 7)
	ctx := context.Background()

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/alliances/": jsonHandler(`{
            "name": "Test Alliance", "ticker": "TSTA",
            "creator_id": 90000001, "creator_corporation_id": 98000001,
            "executor_corporation_id": 98000002,
            "date_founded": "2010-01-01T00:00:00Z"
        }`),
	})
	r := &Refresher{Pool: pool, ESI: client}
	if _, err := r.Alliance(ctx, id); err != nil {
		t.Fatal(err)
	}

	// A second response that has lost the founding fields must not erase them.
	client2 := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/alliances/": jsonHandler(`{
            "name": "Renamed Alliance", "ticker": "NEW",
            "executor_corporation_id": 98000003
        }`),
	})
	r2 := &Refresher{Pool: pool, ESI: client2}
	if _, err := r2.Alliance(ctx, id); err != nil {
		t.Fatal(err)
	}

	var name, ticker string
	var creatorID, creatorCorp, executor *int32
	var founded *time.Time
	if err := pool.QueryRow(ctx, `
        SELECT name, ticker, creator_id, creator_corporation_id, executor_corporation_id, date_founded
        FROM alliances WHERE alliance_id = $1`, id).
		Scan(&name, &ticker, &creatorID, &creatorCorp, &executor, &founded); err != nil {
		t.Fatal(err)
	}

	if name != "Renamed Alliance" || ticker != "NEW" {
		t.Errorf("mutable fields were not updated: name=%q ticker=%q", name, ticker)
	}
	if executor == nil || *executor != 98000003 {
		t.Errorf("executor = %v, want it updated", executor)
	}
	if creatorID == nil || *creatorID != 90000001 {
		t.Errorf("creator_id was lost: %v", creatorID)
	}
	if creatorCorp == nil || *creatorCorp != 98000001 {
		t.Errorf("creator_corporation_id was lost: %v", creatorCorp)
	}
	if founded == nil || founded.UTC().Year() != 2010 {
		t.Errorf("date_founded was lost: %v", founded)
	}
}

// --- History ---

func TestCharacterHistoryStoresAndMarksSynced(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 8)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
        INSERT INTO characters (character_id, name, corporation_id, updated_at)
        VALUES ($1, 'History Pilot', 98000010, now())`, id); err != nil {
		t.Fatal(err)
	}

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/characters/": jsonHandler(`[
            {"record_id": 100, "corporation_id": 98000001, "start_date": "2015-01-01T00:00:00Z"},
            {"record_id": 200, "corporation_id": 98000010, "start_date": "2020-06-01T00:00:00Z"},
            {"record_id": 150, "corporation_id": 98000005, "start_date": "2018-01-01T00:00:00Z"}
        ]`),
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.CharacterHistory(ctx, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 3 {
		t.Errorf("stored %d rows, want 3", res.Rows)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM character_corporation_history WHERE character_id = $1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("%d history rows in the table", count)
	}

	var fetchedAt *time.Time
	var syncedCorp *int32
	var queuedAt *time.Time
	if err := pool.QueryRow(ctx, `
        SELECT corporation_history_fetched_at, corporation_history_synced_corporation_id,
               corporation_history_queued_at
        FROM characters WHERE character_id = $1`, id).Scan(&fetchedAt, &syncedCorp, &queuedAt); err != nil {
		t.Fatal(err)
	}
	if fetchedAt == nil {
		t.Error("the fetched marker was not set")
	}
	// The marker must be the corporation of the *newest* record, not the last
	// one in the array — the entries arrive unordered.
	if syncedCorp == nil || *syncedCorp != 98000010 {
		t.Errorf("synced corporation = %v, want 98000010 (the newest record)", syncedCorp)
	}
	if queuedAt != nil {
		t.Error("the queued marker was not cleared")
	}
}

// Once synced, an unchanged affiliation means unchanged history — refetching it
// would spend one of only sixty requests a minute to learn nothing.
func TestCharacterHistorySkipsWhenSynced(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 9)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
        INSERT INTO characters (
            character_id, name, corporation_id, updated_at,
            corporation_history_fetched_at, corporation_history_synced_corporation_id
        ) VALUES ($1, 'Synced Pilot', 98000010, now(), now(), 98000010)`, id); err != nil {
		t.Fatal(err)
	}

	var called counter
	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/characters/": func(w http.ResponseWriter, _ *http.Request) {
			called.inc()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		},
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.CharacterHistory(ctx, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if called.get() != 0 {
		t.Errorf("ESI was called %d times despite a current sync marker", called.get())
	}
	if res.Status != 304 {
		t.Errorf("status = %d, want 304 (unchanged)", res.Status)
	}

	// --force must override it.
	if _, err := r.CharacterHistory(ctx, id, true); err != nil {
		t.Fatal(err)
	}
	if called.get() != 1 {
		t.Errorf("--force did not override the sync marker (%d calls)", called.get())
	}
}

// The stretches a corporation spent in no alliance are real history, and must be
// stored as NULL rather than dropped.
func TestCorporationHistoryKeepsAllianceGaps(t *testing.T) {
	pool := testPool(t)
	id := reserve(t, pool, 10)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
        INSERT INTO corporations (corporation_id, name, ticker, alliance_id, updated_at)
        VALUES ($1, 'Gap Corp', 'GAP', 99000001, now())`, id); err != nil {
		t.Fatal(err)
	}

	client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
		"/latest/corporations/": jsonHandler(`[
            {"record_id": 1, "alliance_id": 99000001, "start_date": "2020-01-01T00:00:00Z"},
            {"record_id": 2, "start_date": "2019-01-01T00:00:00Z"},
            {"record_id": 3, "alliance_id": 99000002, "start_date": "2018-01-01T00:00:00Z"}
        ]`),
	})

	r := &Refresher{Pool: pool, ESI: client}
	res, err := r.CorporationHistory(ctx, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows != 3 {
		t.Errorf("stored %d rows, want 3 including the unaffiliated stretch", res.Rows)
	}

	var nullRows int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM corporation_alliance_history
        WHERE corporation_id = $1 AND alliance_id IS NULL`, id).Scan(&nullRows); err != nil {
		t.Fatal(err)
	}
	if nullRows != 1 {
		t.Errorf("%d rows have a null alliance, want exactly 1", nullRows)
	}
}

// A malformed ESI row must not be skipped while the parent is marked fully
// synchronized. That would make the missing history permanent because the
// sync marker suppresses every later fetch.
func TestInvalidHistoryDateDoesNotAdvanceSyncMarker(t *testing.T) {
	t.Run("character", func(t *testing.T) {
		pool := testPool(t)
		id := reserve(t, pool, 11)
		ctx := context.Background()

		if _, err := pool.Exec(ctx, `
            INSERT INTO characters (
                character_id, name, corporation_id, updated_at,
                corporation_history_queued_at
            ) VALUES ($1, 'Invalid History Pilot', 98000010, now(), now())`, id); err != nil {
			t.Fatal(err)
		}

		client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
			"/latest/characters/": jsonHandler(`[
                {"record_id": 1, "corporation_id": 98000010, "start_date": "not-a-date"}
            ]`),
		})
		r := &Refresher{Pool: pool, ESI: client}
		if _, err := r.CharacterHistory(ctx, id, false); err == nil {
			t.Fatal("invalid start_date was accepted")
		}

		var fetchedAt, queuedAt *time.Time
		if err := pool.QueryRow(ctx, `
            SELECT corporation_history_fetched_at, corporation_history_queued_at
            FROM characters WHERE character_id = $1`, id).Scan(&fetchedAt, &queuedAt); err != nil {
			t.Fatal(err)
		}
		if fetchedAt != nil || queuedAt == nil {
			t.Errorf("fetched_at=%v queued_at=%v, want unsynced and still queued",
				fetchedAt, queuedAt)
		}
	})

	t.Run("corporation", func(t *testing.T) {
		pool := testPool(t)
		id := reserve(t, pool, 12)
		ctx := context.Background()

		if _, err := pool.Exec(ctx, `
            INSERT INTO corporations (
                corporation_id, name, ticker, alliance_id, updated_at,
                alliance_history_queued_at
            ) VALUES ($1, 'Invalid History Corp', 'BAD', 99000001, now(), now())`, id); err != nil {
			t.Fatal(err)
		}

		client := fakeESIClient(t, map[string]func(http.ResponseWriter, *http.Request){
			"/latest/corporations/": jsonHandler(`[
                {"record_id": 1, "alliance_id": 99000001, "start_date": "not-a-date"}
            ]`),
		})
		r := &Refresher{Pool: pool, ESI: client}
		if _, err := r.CorporationHistory(ctx, id, false); err == nil {
			t.Fatal("invalid start_date was accepted")
		}

		var fetchedAt, queuedAt *time.Time
		if err := pool.QueryRow(ctx, `
            SELECT alliance_history_fetched_at, alliance_history_queued_at
            FROM corporations WHERE corporation_id = $1`, id).Scan(&fetchedAt, &queuedAt); err != nil {
			t.Fatal(err)
		}
		if fetchedAt != nil || queuedAt == nil {
			t.Errorf("fetched_at=%v queued_at=%v, want unsynced and still queued",
				fetchedAt, queuedAt)
		}
	})
}

// --- Staleness ---

func TestStaleDetection(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	fresh := reserve(t, pool, 20)
	stale := reserve(t, pool, 21)
	moved := reserve(t, pool, 22)
	gone := reserve(t, pool, 23)
	unknown := reserve(t, pool, 24)

	seed := func(id int32, corp, ally int32, age time.Duration, deleted bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
            INSERT INTO characters (character_id, name, corporation_id, alliance_id, updated_at, deleted)
            VALUES ($1, 'Seed', $2, $3, now() - $4::interval, $5)`,
			id, corp, ally, fmt.Sprintf("%d seconds", int(age.Seconds())), deleted); err != nil {
			t.Fatal(err)
		}
	}

	seed(fresh, 98000001, 99000001, time.Hour, false)
	seed(stale, 98000001, 99000001, 30*24*time.Hour, false)
	seed(moved, 98000001, 99000001, time.Hour, false)
	seed(gone, 98000001, 99000001, 30*24*time.Hour, true)

	ref := Referenced{
		// The killmail is newer than the records and says `moved` changed corp.
		KillmailTime: time.Now(),
		Affiliations: []Affiliation{
			{CharacterID: fresh, CorporationID: 98000001, AllianceID: 99000001},
			{CharacterID: stale, CorporationID: 98000001, AllianceID: 99000001},
			{CharacterID: moved, CorporationID: 98009999, AllianceID: 99000001},
			{CharacterID: gone, CorporationID: 98000001, AllianceID: 99000001},
			{CharacterID: unknown, CorporationID: 98000001},
		},
	}

	cascade, err := Stale(ctx, pool, ref)
	if err != nil {
		t.Fatal(err)
	}

	got := map[int32]bool{}
	for _, id := range cascade.Characters {
		got[id] = true
	}

	if got[fresh] {
		t.Error("a character updated an hour ago was marked stale")
	}
	if !got[stale] {
		t.Error("a character updated a month ago was not marked stale")
	}
	if !got[moved] {
		t.Error("a character the killmail proves has moved corp was not marked stale")
	}
	if got[gone] {
		t.Error("a biomassed character was queued for refetching")
	}
	if !got[unknown] {
		t.Error("an unknown character was not queued")
	}
}

// An older killmail says nothing useful about a character's current corp — the
// record may simply be newer than the mail.
func TestStaleIgnoresOlderKillmails(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	id := reserve(t, pool, 25)

	if _, err := pool.Exec(ctx, `
        INSERT INTO characters (character_id, name, corporation_id, alliance_id, updated_at)
        VALUES ($1, 'Recent', 98000001, 99000001, now())`, id); err != nil {
		t.Fatal(err)
	}

	cascade, err := Stale(ctx, pool, Referenced{
		KillmailTime: time.Now().Add(-48 * time.Hour),
		Affiliations: []Affiliation{{CharacterID: id, CorporationID: 98009999}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cascade.Characters) != 0 {
		t.Errorf("a two-day-old killmail triggered a refetch: %v", cascade.Characters)
	}
}

func TestStaleFiltersNPCCorporationsAndDuplicates(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// The player corporation and alliance ids come from the reserved band above
	// anything CCP has issued, so nothing stored by a real ingest can make them
	// look fresh. Using a genuine id here is not merely untidy — it makes the
	// test pass or fail depending on whether the killboard happens to have
	// fetched that corporation recently, which is exactly what happened the
	// first time the live pipeline ran against this database.
	playerCorp := testIDBase + 1
	alliance := testIDBase + 2

	cascade, err := Stale(ctx, pool, Referenced{
		Corporations: []int32{1000045, 1000045, playerCorp, playerCorp, 0},
		Alliances:    []int32{alliance, alliance, 0},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range cascade.Corporations {
		if !IsPlayerCorporation(id) {
			t.Errorf("NPC corporation %d survived the filter", id)
		}
	}
	if len(cascade.Corporations) != 1 {
		t.Errorf("corporations = %v, want one deduplicated player corp", cascade.Corporations)
	}
	if len(cascade.Alliances) != 1 {
		t.Errorf("alliances = %v, want one deduplicated id", cascade.Alliances)
	}
}

func TestCascadeEmpty(t *testing.T) {
	if !(Cascade{}).Empty() {
		t.Error("a zero cascade is empty")
	}
	if (Cascade{Alliances: []int32{1}}).Empty() {
		t.Error("a cascade with work is not empty")
	}
}

type counter struct{ n int }

func (c *counter) inc()     { c.n++ }
func (c *counter) get() int { return c.n }
