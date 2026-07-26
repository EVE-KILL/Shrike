package eve

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Valuation, in the order the price of a type is decided:
//
//  1. custom_prices — the override for hulls the market cannot price
//  2. a per-group rule — skins, capsules and two ore types with no real market
//  3. the Jita average on the kill date, or the most recent one before it
//  4. failing all of that, 0.01 ISK
//
// Step 4 is not a placeholder for "unknown". It is what the killboard has
// always recorded, and changing it would silently reprice history, so it stays.

// FallbackPrice is the value of a type with no market history at or before the
// kill date.
const FallbackPrice = 0.01

// Per-group overrides, kept in the order the TypeScript checks them.
const (
	categorySKIN     int32 = 91   // cosmetic, never worth more than the floor
	groupCapsule     int32 = 29   // pods are priced by convention, not market
	groupMordunium   int32 = 4513 // ore introduced without a market presence
	groupGriemeer    int32 = 4759 // likewise
	capsulePrice           = 10_000.0
	newOreTypesPrice       = 200.0
)

// groupPrice reports the fixed price for a type whose market cannot price it,
// and false when the market should decide.
func (c *Cache) groupPrice(typeID int32) (float64, bool) {
	t, ok := c.types[typeID]
	if !ok || t.GroupID == 0 {
		return 0, false
	}
	switch {
	case t.CategoryID == categorySKIN:
		return FallbackPrice, true
	case t.GroupID == groupCapsule:
		return capsulePrice, true
	case t.GroupID == groupMordunium, t.GroupID == groupGriemeer:
		return newOreTypesPrice, true
	}
	return 0, false
}

// Prices resolves market values for a given day.
//
// Killmails are valued a whole mail at a time — one ship plus everything fitted
// and in the hold — so the lookup is built around a batch: collect every type
// on the mail, resolve them in one query, then value from memory. Fetching each
// type separately would turn a 200-item Rorqual loss into 200 round trips.
type Prices struct {
	pool  *pgxpool.Pool
	cache *Cache

	mu sync.Mutex
	// memo holds resolved market averages for memoDate only. Backfills run in
	// date order, so dropping the whole map when the date changes keeps this
	// bounded by the number of types (~50 k) rather than by types × days, which
	// over eighteen years of history would not fit in memory.
	memoDate string
	memo     map[int32]float64
	// memoComplete means memo holds every priced type for memoDate, so a miss
	// is genuinely "never traded at or before this date" rather than "not asked
	// for yet". Set by Snapshot.
	memoComplete bool
}

// NewPrices builds a price resolver over a loaded cache.
func NewPrices(pool *pgxpool.Pool, cache *Cache) *Prices {
	return &Prices{pool: pool, cache: cache, memo: map[int32]float64{}}
}

// Day is the resolved price of every requested type on one date.
type Day struct {
	date   string
	prices *Prices
	market map[int32]float64
}

// On resolves the market price of each type as of date (YYYY-MM-DD).
//
// Types settled by a custom price or a group rule are not queried — those
// answers do not depend on the date, so asking the market about them would be
// wasted work.
func (p *Prices) On(ctx context.Context, date string, typeIDs []int32) (*Day, error) {
	d := &Day{date: date, prices: p, market: make(map[int32]float64, len(typeIDs))}

	p.mu.Lock()
	if p.memoDate != date {
		p.memoDate = date
		p.memo = make(map[int32]float64, len(typeIDs))
		p.memoComplete = false
	}

	var missing []int32
	seen := make(map[int32]bool, len(typeIDs))
	for _, id := range typeIDs {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		if _, ok := p.cache.CustomPrice(id); ok {
			continue
		}
		if _, ok := p.cache.groupPrice(id); ok {
			continue
		}
		if v, ok := p.memo[id]; ok {
			d.market[id] = v
			continue
		}
		if p.memoComplete {
			// The snapshot holds every type that has ever traded by this date,
			// so a miss needs no query — it resolves to the floor.
			continue
		}
		missing = append(missing, id)
	}
	p.mu.Unlock()

	if len(missing) == 0 {
		return d, nil
	}

	// One row per type: the most recent average at or before the kill date.
	// The region is not filtered because the importer only ever stores The
	// Forge — the table is Jita by construction.
	rows, err := p.pool.Query(ctx, `
        SELECT DISTINCT ON (type_id) type_id, average
        FROM prices
        WHERE type_id = ANY($1::int[]) AND date <= $2::date
        ORDER BY type_id, date DESC`, missing, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[int32]float64, len(missing))
	for rows.Next() {
		var id int32
		var avg *float64
		if err := rows.Scan(&id, &avg); err != nil {
			return nil, err
		}
		// A row whose average is NULL is a day the type was listed but never
		// traded. That is not a price, so it falls through to the floor.
		if avg == nil {
			continue
		}
		found[id] = *avg
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	for _, id := range missing {
		v, ok := found[id]
		if !ok {
			v = FallbackPrice
		}
		// Guard against a concurrent On having rotated the date underneath us;
		// writing these into a different day's memo would misprice that day.
		if p.memoDate == date {
			p.memo[id] = v
		}
		d.market[id] = v
	}
	p.mu.Unlock()

	return d, nil
}

// Of returns the price of a type, applying the full order of precedence.
//
// A type that was not passed to On resolves to the floor rather than reaching
// for the database, so a missed type shows up as a wrong value in a diff rather
// than as an unexplained query in a hot loop.
func (d *Day) Of(typeID int32) float64 {
	if v, ok := d.prices.cache.CustomPrice(typeID); ok {
		return v
	}
	if v, ok := d.prices.cache.groupPrice(typeID); ok {
		return v
	}
	if v, ok := d.market[typeID]; ok {
		return v
	}
	return FallbackPrice
}

// Snapshot resolves every type's price as of date in a single query.
//
// It exists for the archive importers, which parse a whole day of killmails at
// once. Asking per killmail would be twenty thousand queries for a day that has
// exactly one answer per type; asking once up front is one. After it returns,
// On() for that date is answered entirely from memory, including for types that
// have never traded.
func (p *Prices) Snapshot(ctx context.Context, date string) (int, error) {
	rows, err := p.pool.Query(ctx, `
        SELECT DISTINCT ON (type_id) type_id, average
        FROM prices
        WHERE date <= $1::date AND average > 0
        ORDER BY type_id, date DESC`, date)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	memo := make(map[int32]float64, 32768)
	for rows.Next() {
		var id int32
		var avg float64
		if err := rows.Scan(&id, &avg); err != nil {
			return 0, err
		}
		memo[id] = avg
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	p.mu.Lock()
	p.memoDate = date
	p.memo = memo
	p.memoComplete = true
	p.mu.Unlock()

	return len(memo), nil
}

// Seed installs a day's prices directly, without a database.
//
// It exists so a price snapshot can come from somewhere other than Postgres —
// a test fixture, or a warm start from a file. After it returns, On() for that
// date is answered entirely from memory and a type absent from the snapshot
// resolves to the floor rather than triggering a query, exactly as it would
// after Snapshot.
func (p *Prices) Seed(date string, market map[int32]float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.memoDate = date
	p.memo = market
	p.memoComplete = true
}
