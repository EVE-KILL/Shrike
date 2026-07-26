// Package eve holds the static-data lookups that every killmail touches.
//
// A killmail arrives from ESI as a bag of numeric IDs. Turning it into a row
// means resolving each of those against the SDE — the ship's group, the group's
// category, the system's region, a module's meta level — and doing so tens of
// times per kill. Going to Postgres for each would make the parser
// latency-bound on data that changes once a fortnight, so the whole working set
// is loaded once and read from memory afterwards.
//
// The set is small: ~53 k types, ~8.5 k systems, ~11 k dogma values, and a few
// thousand rows besides. Under 20 MB resident.
package eve

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Zero means absent throughout this package.
//
// EVE issues no entity with ID 0 — the SDE importer rejects it outright, and
// the TypeScript services coerce with `||` rather than `??` for exactly this
// reason, so a 0 and a NULL have always been the same thing here. Fields that
// can legitimately hold zero (a security status, a parent index) are the
// exception and say so at their declaration.

// Type is the subset of inv_types the killmail path reads.
type Type struct {
	GroupID    int32
	CategoryID int32
	Name       string

	// VariationParentTypeID is the T1 root of a meta/variant family. The
	// fittings extractor collapses T2, faction and storyline variants onto it
	// so one doctrine does not read as a dozen. Zero for types with no parent.
	VariationParentTypeID int32

	// MetaGroupID distinguishes deadspace (5) and officer (6) modules, which
	// is what makes a fit worth announcing.
	MetaGroupID int32

	// MarketGroupID resolves to a `/market/a/b/c` path when a killlist row is
	// published live, so it links the same way a REST-fetched row does.
	MarketGroupID int32
}

// Group is the subset of inv_groups the killmail path reads.
type Group struct {
	CategoryID int32
	Name       string
}

// System is the subset of solar_systems the killmail path reads.
type System struct {
	ConstellationID int32
	RegionID        int32
	Name            string

	// Security is stored as a plain float because the column has no NULLs:
	// every one of the 8,490 systems carries a value, and 0.0 is a real
	// security rating rather than a stand-in for "unknown".
	Security float64
}

// Region is the subset of regions the killmail path reads.
type Region struct {
	Name string
}

// Constellation is the subset of constellations the killmail path reads.
type Constellation struct {
	Name     string
	RegionID int32
}

// DogmaAttributes are the only attributes any killmail computation reads.
//
//	1547 rigSize    — the ship-size exponent behind base points
//	 633 metaLevel  — module quality, weighting the danger factor
//	1211 heatDamage — marks a module as active-fittable, i.e. dangerous
//
// Loading all ~14 M rows of type_dogma_attributes to reach 11 k of them would
// be absurd, so the preload filters. A lookup for any other attribute reports
// absent, which is correct here and would be a bug if this list ever grows
// without the callers being revisited.
var DogmaAttributes = []int32{1547, 633, 1211}

const (
	AttrRigSize    int32 = 1547
	AttrMetaLevel  int32 = 633
	AttrHeatDamage int32 = 1211
)

// DogmaKey identifies one attribute of one type.
type DogmaKey struct {
	TypeID      int32
	AttributeID int32
}

// CacheData is the raw contents of a Cache. Load fills one from Postgres;
// NewCache accepts one directly, which is what tests and fixtures use.
type CacheData struct {
	Types          map[int32]Type
	Groups         map[int32]Group
	Systems        map[int32]System
	Regions        map[int32]Region
	Constellations map[int32]Constellation
	Dogma          map[DogmaKey]float64
	CustomPrices   map[int32]float64
}

// NewCache builds a cache from data already in hand. The maps are adopted, not
// copied — the caller must not retain a reference and mutate them, which would
// break the no-locking guarantee.
func NewCache(d CacheData) *Cache {
	c := &Cache{
		types:          d.Types,
		groups:         d.Groups,
		systems:        d.Systems,
		regions:        d.Regions,
		constellations: d.Constellations,
		dogma:          d.Dogma,
		customPrices:   d.CustomPrices,
	}
	// Nil maps read fine but would panic on the writes Load performs, so a
	// partially specified cache is completed rather than trusted.
	if c.types == nil {
		c.types = map[int32]Type{}
	}
	if c.groups == nil {
		c.groups = map[int32]Group{}
	}
	if c.systems == nil {
		c.systems = map[int32]System{}
	}
	if c.regions == nil {
		c.regions = map[int32]Region{}
	}
	if c.constellations == nil {
		c.constellations = map[int32]Constellation{}
	}
	if c.dogma == nil {
		c.dogma = map[DogmaKey]float64{}
	}
	if c.customPrices == nil {
		c.customPrices = map[int32]float64{}
	}
	return c
}

// Cache is an immutable snapshot of the static data.
//
// Nothing writes to it after Load returns, so it needs no locking and can be
// shared across every goroutine in the process. Picking up a new SDE build
// means calling Load again and swapping the pointer — deliberately not an
// in-place refresh, which would reintroduce the locking this design exists to
// avoid.
type Cache struct {
	types          map[int32]Type
	groups         map[int32]Group
	systems        map[int32]System
	regions        map[int32]Region
	constellations map[int32]Constellation
	dogma          map[DogmaKey]float64
	customPrices   map[int32]float64
}

// Load reads every lookup table into memory.
//
// The queries run in sequence rather than concurrently: each is a bulk scan
// where transfer dominates and round trips do not, and serialising them keeps
// one connection busy instead of occupying most of a pool sized for a pooler.
func Load(ctx context.Context, pool *pgxpool.Pool) (*Cache, error) {
	c := &Cache{
		types:          make(map[int32]Type, 60000),
		groups:         make(map[int32]Group, 1600),
		systems:        make(map[int32]System, 9000),
		regions:        make(map[int32]Region, 120),
		constellations: make(map[int32]Constellation, 1200),
		dogma:          make(map[DogmaKey]float64, 12000),
		customPrices:   make(map[int32]float64, 128),
	}

	if err := load(ctx, pool, `
        SELECT type_id,
               coalesce(group_id, 0), coalesce(category_id, 0), coalesce(name, ''),
               coalesce(variation_parent_type_id, 0), coalesce(meta_group_id, 0),
               coalesce(market_group_id, 0)
        FROM inv_types`,
		func(scan scanner) error {
			var id int32
			var t Type
			if err := scan(&id, &t.GroupID, &t.CategoryID, &t.Name,
				&t.VariationParentTypeID, &t.MetaGroupID, &t.MarketGroupID); err != nil {
				return err
			}
			c.types[id] = t
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load inv_types: %w", err)
	}

	if err := load(ctx, pool, `
        SELECT group_id, coalesce(category_id, 0), coalesce(name, '') FROM inv_groups`,
		func(scan scanner) error {
			var id int32
			var g Group
			if err := scan(&id, &g.CategoryID, &g.Name); err != nil {
				return err
			}
			c.groups[id] = g
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load inv_groups: %w", err)
	}

	if err := load(ctx, pool, `
        SELECT solar_system_id, coalesce(constellation_id, 0), coalesce(region_id, 0),
               coalesce(system_name, ''), coalesce(security, 0)
        FROM solar_systems`,
		func(scan scanner) error {
			var id int32
			var s System
			if err := scan(&id, &s.ConstellationID, &s.RegionID, &s.Name, &s.Security); err != nil {
				return err
			}
			c.systems[id] = s
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load solar_systems: %w", err)
	}

	if err := load(ctx, pool, `SELECT region_id, coalesce(name, '') FROM regions`,
		func(scan scanner) error {
			var id int32
			var r Region
			if err := scan(&id, &r.Name); err != nil {
				return err
			}
			c.regions[id] = r
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load regions: %w", err)
	}

	if err := load(ctx, pool, `
        SELECT constellation_id, coalesce(constellation_name, ''), coalesce(region_id, 0)
        FROM constellations`,
		func(scan scanner) error {
			var id int32
			var cn Constellation
			if err := scan(&id, &cn.Name, &cn.RegionID); err != nil {
				return err
			}
			c.constellations[id] = cn
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load constellations: %w", err)
	}

	if err := load(ctx, pool, `
        SELECT type_id, attribute_id, value
        FROM type_dogma_attributes
        WHERE attribute_id = ANY($1::int[])`,
		func(scan scanner) error {
			var k DogmaKey
			var v float64
			if err := scan(&k.TypeID, &k.AttributeID, &v); err != nil {
				return err
			}
			c.dogma[k] = v
			return nil
		}, DogmaAttributes); err != nil {
		return nil, fmt.Errorf("load type_dogma_attributes: %w", err)
	}

	// One row per type: `date` on custom_prices is a valid-until sentinel
	// (every current row reads 9999-12-31, i.e. "applies forever"), not a
	// time series. Filtering it by the kill date would exclude everything, so
	// take the latest row per type — the same rule BattleDetector and the
	// battle endpoints already apply.
	if err := load(ctx, pool, `
        SELECT DISTINCT ON (type_id) type_id, price
        FROM custom_prices
        ORDER BY type_id, date DESC`,
		func(scan scanner) error {
			var id int32
			var price float64
			if err := scan(&id, &price); err != nil {
				return err
			}
			c.customPrices[id] = price
			return nil
		}); err != nil {
		return nil, fmt.Errorf("load custom_prices: %w", err)
	}

	return c, nil
}

// scanner is the row-scan closure handed to each load callback.
type scanner func(dest ...any) error

func load(ctx context.Context, pool *pgxpool.Pool, query string, fn func(scanner) error, args ...any) error {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := fn(rows.Scan); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Type looks up an inventory type.
func (c *Cache) Type(id int32) (Type, bool) {
	t, ok := c.types[id]
	return t, ok
}

// Group looks up an inventory group.
func (c *Cache) Group(id int32) (Group, bool) {
	g, ok := c.groups[id]
	return g, ok
}

// System looks up a solar system.
func (c *Cache) System(id int32) (System, bool) {
	s, ok := c.systems[id]
	return s, ok
}

// Region looks up a region.
func (c *Cache) Region(id int32) (Region, bool) {
	r, ok := c.regions[id]
	return r, ok
}

// Constellation looks up a constellation.
func (c *Cache) Constellation(id int32) (Constellation, bool) {
	cn, ok := c.constellations[id]
	return cn, ok
}

// Dogma looks up one attribute of one type. Only the attributes in
// DogmaAttributes are held; anything else always reports absent.
func (c *Cache) Dogma(typeID, attrID int32) (float64, bool) {
	v, ok := c.dogma[DogmaKey{TypeID: typeID, AttributeID: attrID}]
	return v, ok
}

// CustomPrice returns the hand-maintained or generated price for a type, which
// overrides the market for hulls that never trade.
func (c *Cache) CustomPrice(typeID int32) (float64, bool) {
	p, ok := c.customPrices[typeID]
	return p, ok
}

// CategoryOfType resolves a type straight to its category, which is the
// question most callers actually have. It reads inv_types.category_id — already
// denormalised by the SDE importer — rather than hopping through the group.
func (c *Cache) CategoryOfType(typeID int32) int32 {
	return c.types[typeID].CategoryID
}

// CountsByName reports how many rows of each table are held, which is what
// makes an empty or half-loaded cache visible before it silently produces
// killmails full of nulls.
func (c *Cache) CountsByName() map[string]int {
	return map[string]int{
		"inv_types":             len(c.types),
		"inv_groups":            len(c.groups),
		"solar_systems":         len(c.systems),
		"regions":               len(c.regions),
		"constellations":        len(c.constellations),
		"type_dogma_attributes": len(c.dogma),
		"custom_prices":         len(c.customPrices),
	}
}
