package eve

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func priceCache() *Cache {
	return NewCache(CacheData{
		Types: map[int32]Type{
			587:   {GroupID: 25, CategoryID: 6},    // an ordinary hull
			670:   {GroupID: 29, CategoryID: 6},    // capsule
			49738: {GroupID: 4513, CategoryID: 25}, // Mordunium
			52306: {GroupID: 4759, CategoryID: 25}, // Griemeer
			42678: {GroupID: 1950, CategoryID: 91}, // a SKIN
			671:   {GroupID: 30, CategoryID: 6},    // titan, priced by override
			// A type the SDE knows but that has no group — the market decides.
			99999: {CategoryID: 91},
		},
		CustomPrices: map[int32]float64{
			671: 95_000_000_000,
			587: 1, // deliberately absurd: an override must beat the market
		},
	})
}

// The group rules exist because these types have no usable market. Getting one
// wrong misprices every killmail carrying it, in both directions.
func TestGroupPrice(t *testing.T) {
	c := priceCache()
	cases := []struct {
		name   string
		typeID int32
		want   float64
		ok     bool
	}{
		{"skin", 42678, FallbackPrice, true},
		{"capsule", 670, capsulePrice, true},
		{"mordunium", 49738, newOreTypesPrice, true},
		{"griemeer", 52306, newOreTypesPrice, true},
		{"ordinary hull", 587, 0, false},
		{"unknown type", 123456, 0, false},
		// A type with no group is not classifiable, so the market decides even
		// though its category says SKIN.
		{"grouped nowhere", 99999, 0, false},
	}
	for _, c2 := range cases {
		got, ok := c.groupPrice(c2.typeID)
		if ok != c2.ok || got != c2.want {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", c2.name, got, ok, c2.want, c2.ok)
		}
	}
}

// Precedence is the whole design: a custom price beats a group rule, a group
// rule beats the market, and anything unpriced lands on the floor rather than
// on zero.
func TestDayPrecedence(t *testing.T) {
	c := priceCache()
	p := NewPrices(nil, c)
	day := &Day{
		date:   "2026-07-25",
		prices: p,
		market: map[int32]float64{587: 5_000_000, 670: 42},
	}

	if got := day.Of(671); got != 95_000_000_000 {
		t.Errorf("titan: got %v, want the custom price", got)
	}
	if got := day.Of(587); got != 1 {
		t.Errorf("custom price did not beat the market: got %v", got)
	}
	// The capsule has a market entry here and must still be priced by rule.
	if got := day.Of(670); got != capsulePrice {
		t.Errorf("capsule: got %v, want the group rule to beat the market", got)
	}
	if got := day.Of(42678); got != FallbackPrice {
		t.Errorf("skin: got %v, want %v", got, FallbackPrice)
	}
	// A type nobody asked about resolves to the floor rather than reaching for
	// the database behind the caller's back.
	if got := day.Of(123456); got != FallbackPrice {
		t.Errorf("unrequested type: got %v, want %v", got, FallbackPrice)
	}
}

// A cache assembled without every map must still be safe to read.
func TestNewCacheTolerAtesMissingMaps(t *testing.T) {
	c := NewCache(CacheData{})
	if _, ok := c.Type(587); ok {
		t.Error("empty cache reported a type")
	}
	if _, ok := c.Dogma(587, AttrRigSize); ok {
		t.Error("empty cache reported a dogma value")
	}
	if got := c.CountsByName()["inv_types"]; got != 0 {
		t.Errorf("empty cache counted %d types", got)
	}
}

// The TypeScript parser always prices against The Forge (Jita). The table can
// hold more regions, so omitting that predicate silently values a kill from
// whichever region happened to have the newest row.
func TestPricesUseTheForgeAndPreserveZeroAverages(t *testing.T) {
	pool := isolatedPricePool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		CREATE TEMP TABLE prices (
			type_id integer NOT NULL,
			region_id integer NOT NULL,
			date date NOT NULL,
			average double precision
		) ON COMMIT PRESERVE ROWS;
		INSERT INTO prices (type_id, region_id, date, average) VALUES
			(42, 10000002, '2026-07-20', 100),
			(42, 10000043, '2026-07-21', 999),
			(43, 10000002, '2026-07-20', 0);
	`); err != nil {
		t.Fatalf("seed temporary prices: %v", err)
	}

	cache := NewCache(CacheData{})
	prices := NewPrices(pool, cache)
	day, err := prices.On(ctx, "2026-07-21", []int32{42, 43})
	if err != nil {
		t.Fatalf("On: %v", err)
	}
	if got := day.Of(42); got != 100 {
		t.Errorf("type 42 price = %v, want The Forge value 100", got)
	}
	if got := day.Of(43); got != 0 {
		t.Errorf("type 43 price = %v, want stored zero average", got)
	}

	snapshotPrices := NewPrices(pool, cache)
	if _, err := snapshotPrices.Snapshot(ctx, "2026-07-21"); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	snapshot, err := snapshotPrices.On(ctx, "2026-07-21", []int32{42, 43})
	if err != nil {
		t.Fatalf("On after Snapshot: %v", err)
	}
	if got := snapshot.Of(42); got != 100 {
		t.Errorf("snapshot type 42 price = %v, want The Forge value 100", got)
	}
	if got := snapshot.Of(43); got != 0 {
		t.Errorf("snapshot type 43 price = %v, want stored zero average", got)
	}
}

func isolatedPricePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	// A temp table is session-local. One connection guarantees the setup and
	// the production query use that same session without touching public data.
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
