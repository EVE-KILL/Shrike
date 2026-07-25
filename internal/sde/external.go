package sde

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Data the SDE archive does not carry, sourced from EVE Ref plus two bundled
// price lists. The TypeScript importer folds these into the same command, so
// they are here too rather than in a separate one.

const (
	insuranceURL  = "https://data.everef.net/insurance-prices/insurance-prices-latest.json"
	structuresURL = "https://data.everef.net/structures/structures-latest.v2.json"

	// externalTimeout covers the structures feed, which is the larger of the two.
	externalTimeout = 5 * time.Minute

	// customPriceDate is a valid-until sentinel rather than a real date: these
	// prices apply indefinitely. Killmail valuation picks the row whose date is
	// the earliest one still in the future.
	customPriceDate = "9999-12-31"
)

// ExternalResult reports one external feed.
type ExternalResult struct {
	Name    string `json:"name"`
	Rows    int64  `json:"rows"`
	Elapsed string `json:"elapsed"`
}

func fetchJSON(ctx context.Context, url, userAgent string, into any) error {
	ctx, cancel := context.WithTimeout(ctx, externalTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	// Streamed rather than read whole: the structures feed is several MB.
	return json.NewDecoder(resp.Body).Decode(into)
}

// ImportInsurancePrices loads ship insurance costs and payouts.
//
// The feed nests six levels per type — Basic through Platinum — which flatten
// into one row each.
func ImportInsurancePrices(ctx context.Context, pool *pgxpool.Pool, userAgent string) (ExternalResult, error) {
	start := time.Now()
	res := ExternalResult{Name: "insurance_prices"}

	var feed []struct {
		TypeID int32 `json:"type_id"`
		Levels []struct {
			Name   string  `json:"name"`
			Cost   float64 `json:"cost"`
			Payout float64 `json:"payout"`
		} `json:"levels"`
	}
	if err := fetchJSON(ctx, insuranceURL, userAgent, &feed); err != nil {
		return res, fmt.Errorf("fetch insurance prices: %w", err)
	}

	batch := &pgx.Batch{}
	for _, t := range feed {
		for _, l := range t.Levels {
			batch.Queue(`
                INSERT INTO insurance_prices (type_id, level_name, cost, payout)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (type_id, level_name) DO UPDATE
                SET cost = EXCLUDED.cost, payout = EXCLUDED.payout
            `, t.TypeID, l.Name, l.Cost, l.Payout)
			res.Rows++
		}
	}

	if err := sendBatch(ctx, pool, batch, int(res.Rows)); err != nil {
		return res, fmt.Errorf("write insurance prices: %w", err)
	}
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// ImportStructures loads player-owned structures observed by EVE Ref.
//
// Unlike everything else here this is not static data — structures are built and
// destroyed constantly — but killmails reference them as locations, so the names
// have to come from somewhere.
func ImportStructures(ctx context.Context, pool *pgxpool.Pool, userAgent string) (ExternalResult, error) {
	start := time.Now()
	res := ExternalResult{Name: "structures"}

	var feed map[string]struct {
		StructureID   int64   `json:"structure_id"`
		Name          *string `json:"name"`
		OwnerID       *int32  `json:"owner_id"`
		SolarSystemID *int32  `json:"solar_system_id"`
		TypeID        *int32  `json:"type_id"`
		Position      *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"position"`
		IsPublic *bool `json:"is_public_structure"`
		IsMarket *bool `json:"has_market"`
	}
	if err := fetchJSON(ctx, structuresURL, userAgent, &feed); err != nil {
		return res, fmt.Errorf("fetch structures: %w", err)
	}

	batch := &pgx.Batch{}
	for _, s := range feed {
		if s.StructureID == 0 {
			continue
		}
		var x, y, z *float64
		if s.Position != nil {
			x, y, z = &s.Position.X, &s.Position.Y, &s.Position.Z
		}
		// first_seen is set once and never moved; last_seen advances every run,
		// which is what makes "when did this structure disappear" answerable.
		batch.Queue(`
            INSERT INTO structures
                (structure_id, name, owner_id, solar_system_id, type_id,
                 x, y, z, is_public, is_market, first_seen, last_seen)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now(), now())
            ON CONFLICT (structure_id) DO UPDATE
            SET name = EXCLUDED.name, owner_id = EXCLUDED.owner_id,
                solar_system_id = EXCLUDED.solar_system_id, type_id = EXCLUDED.type_id,
                x = EXCLUDED.x, y = EXCLUDED.y, z = EXCLUDED.z,
                is_public = EXCLUDED.is_public, is_market = EXCLUDED.is_market,
                last_seen = now()
        `, s.StructureID, s.Name, s.OwnerID, s.SolarSystemID, s.TypeID,
			x, y, z, s.IsPublic, s.IsMarket)
		res.Rows++
	}

	if err := sendBatch(ctx, pool, batch, int(res.Rows)); err != nil {
		return res, fmt.Errorf("write structures: %w", err)
	}
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// manualPrice is a hand-set value for a hull the market cannot price — unique
// ships, event rewards, and things that have never been sold.
type manualPrice struct {
	TypeID int32
	// double precision, not an integer: several hulls are pinned to 0.01 ISK
	// because their market is manipulated or they only drop from sites.
	Price float64
}

var manualPrices = []manualPrice{
	{12478, 0.01}, // Khumaak
	{34559, 0.01}, // Conflux Element
	{44265, 0.01}, // Victory Firework
	{34558, 0.01},
	{34556, 0.01},
	{34560, 0.01},
	{36902, 0.01},
	{34557, 0.01},
	{44264, 0.01},     // Market manipulation override
	{55511, 30000000}, // Rare ships — conditional prices (using current/default values)
	{88001, 10000000000},
	{3514, 250000000000},   // Revenant
	{42241, 650000000000},  // Molok
	{11940, 3400000000000}, // Gold Magnate
	{45645, 35000000000},   // Loggerhead
	{87381, 45000000000},   // Sarathiel
	{42124, 45000000000},
	{42243, 70000000000},   // Chemosh
	{2834, 80000000000},    // Utu
	{3516, 80000000000},    // Malice
	{11375, 80000000000},   // Freki
	{3518, 100000000000},   // Vangel
	{32788, 100000000000},  // Cambion
	{32790, 100000000000},  // Etana
	{32209, 100000000000},  // Mimir
	{11942, 100000000000},  // Silver Magnate
	{33673, 100000000000},  // Whiptail
	{35779, 120000000000},  // Imp
	{42125, 120000000000},  // Vendetta
	{42246, 120000000000},  // Caedes
	{74141, 120000000000},  // Geri
	{2836, 150000000000},   // Adrestia
	{33675, 150000000000},  // Chameleon
	{35781, 150000000000},  // Fiend
	{45530, 150000000000},  // Virtuoso
	{48636, 150000000000},  // Hydra
	{60765, 150000000000},  // Raiju
	{74316, 150000000000},  // Bestla
	{78414, 150000000000},  // Shapash
	{33397, 200000000000},  // Chremoas
	{42245, 200000000000},  // Rabisu
	{85062, 200000000000},  // Sidewinder
	{45531, 230000000000},  // Victor
	{89808, 230000000000},  // Skua
	{48635, 230000000000},  // Tiamat
	{60764, 230000000000},  // Laelaps
	{77726, 230000000000},  // Cybele
	{85229, 250000000000},  // Cobra
	{47512, 60000000000},   // 'Moreau' Fortizar
	{45647, 60000000000},   // Caiman
	{89807, 450000000000},  // Anhinga
	{45649, 550000000000},  // Komodo
	{42126, 650000000000},  // Vanquisher
	{9860, 1000000000000},  // Polaris
	{11019, 1000000000000}, // Cockroach
	{85236, 1250000000000}, // Python
	{635, 500000000000},    // Opux Luxury Yacht
	{11011, 500000000000},  // Guardian-Vexor
	{25560, 500000000000},  // Opux Dragoon Yacht
	{33395, 500000000000},  // Moracha
	{13202, 750000000000},  // Megathron Federate Issue
	{11936, 750000000000},  // Apocalypse Imperial Issue
	{11938, 750000000000},  // Armageddon Imperial Issue
	{26842, 750000000000},  // Tempest Tribal Issue
	{78576, 750000000000},  // Azariel (Angel Titan)
	{26840, 2500000000000}, // Raven State Issue
	{47514, 60000000000},   // 'Horizon' Fortizar
	{42242, 60000000000},   // Dagon
}

// SeedManualCustomPrices writes the hand-maintained price list.
func SeedManualCustomPrices(ctx context.Context, pool *pgxpool.Pool) (ExternalResult, error) {
	start := time.Now()
	res := ExternalResult{Name: "custom_prices (manual)"}

	batch := &pgx.Batch{}
	for _, p := range manualPrices {
		batch.Queue(`
            INSERT INTO custom_prices (type_id, date, price)
            VALUES ($1, $2::date, $3)
            ON CONFLICT (type_id, date) DO UPDATE SET price = EXCLUDED.price
        `, p.TypeID, customPriceDate, p.Price)
		res.Rows++
	}
	if err := sendBatch(ctx, pool, batch, len(manualPrices)); err != nil {
		return res, err
	}
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// Supercapital hulls: titans (30) and supercarriers (659). They are never sold
// on the open market, so their value is computed from what they cost to build.
var supercapGroups = []int32{30, 659}

// buildCostMarkup approximates the difference between raw material cost and
// what a hull actually changes hands for.
const buildCostMarkup = 1.08

// GenerateSupercapPrices derives supercapital values from blueprint materials.
//
// Requires the prices table, which is populated by the market-history importer
// rather than by the SDE. With no prices loaded every hull costs zero and
// nothing is written — that is the correct outcome, not a failure, so this
// reports zero rows instead of erroring.
func GenerateSupercapPrices(ctx context.Context, pool *pgxpool.Pool) (ExternalResult, error) {
	start := time.Now()
	res := ExternalResult{Name: "custom_prices (supercapitals)"}

	// Latest price per type. No region filter is needed: the importer only ever
	// stores The Forge, so the table is Jita by construction.
	//
	// average > 0 stays — it drops days where a type was listed but never
	// traded, which would otherwise price a titan's worth of minerals at zero.
	latest := map[int32]float64{}
	rows, err := pool.Query(ctx, `
        SELECT DISTINCT ON (type_id) type_id, average
        FROM prices
        WHERE average > 0
        ORDER BY type_id, date DESC`)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id int32
		var avg *float64
		if err := rows.Scan(&id, &avg); err != nil {
			rows.Close()
			return res, err
		}
		if avg != nil {
			latest[id] = *avg
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return res, rows.Err()
	}

	// product type -> the blueprint that makes it, and how many it yields.
	//
	// The output quantity is essential: a reaction that produces 200 units of a
	// composite from one run must divide its material cost by 200. Omitting it
	// does not merely inflate the answer, it compounds at every level of the
	// recursion — a titan came out roughly 16,000x too expensive.
	type bpKey struct {
		blueprint int32
		activity  string
	}
	type product struct {
		bpKey
		quantity float64
	}
	producedBy := map[int32]product{}
	if err := scanRows(ctx, pool, `
        SELECT product_type_id, blueprint_type_id, activity, quantity
        FROM blueprint_activity_products
        WHERE activity IN ('manufacturing', 'reaction')`, func(s pgx.Rows) error {
		var productID, bp int32
		var activity string
		var qty *int32
		if err := s.Scan(&productID, &bp, &activity, &qty); err != nil {
			return err
		}
		q := 1.0
		if qty != nil && *qty > 0 {
			q = float64(*qty)
		}
		// A type can come from both a reaction and a manufacturing job;
		// manufacturing is the one that reflects build cost.
		existing, ok := producedBy[productID]
		if !ok || (existing.activity == "reaction" && activity == "manufacturing") {
			producedBy[productID] = product{bpKey{bp, activity}, q}
		}
		return nil
	}); err != nil {
		return res, err
	}

	type material struct {
		typeID   int32
		quantity int32
	}
	materials := map[bpKey][]material{}
	if err := scanRows(ctx, pool, `
        SELECT blueprint_type_id, activity, material_type_id, quantity
        FROM blueprint_activity_materials`, func(s pgx.Rows) error {
		var bp, mat int32
		var activity string
		var qty *int32
		if err := s.Scan(&bp, &activity, &mat, &qty); err != nil {
			return err
		}
		q := int32(0)
		if qty != nil {
			q = *qty
		}
		k := bpKey{bp, activity}
		materials[k] = append(materials[k], material{mat, q})
		return nil
	}); err != nil {
		return res, err
	}

	// Recursive descent with memoisation. The visited set breaks cycles — some
	// reaction chains consume their own output — by falling back to the market
	// price for the type that closes the loop.
	cache := map[int32]float64{}
	var unitCost func(typeID int32, visited map[int32]bool) float64
	unitCost = func(typeID int32, visited map[int32]bool) float64 {
		if c, ok := cache[typeID]; ok {
			return c
		}
		if visited[typeID] {
			return latest[typeID]
		}
		bp, ok := producedBy[typeID]
		if !ok {
			c := latest[typeID]
			cache[typeID] = c
			return c
		}
		mats := materials[bp.bpKey]
		if len(mats) == 0 {
			c := latest[typeID]
			cache[typeID] = c
			return c
		}
		next := make(map[int32]bool, len(visited)+1)
		for k := range visited {
			next[k] = true
		}
		next[typeID] = true

		total := 0.0
		for _, m := range mats {
			total += unitCost(m.typeID, next) * float64(m.quantity)
		}
		// Cost of one unit, not of one production run.
		perUnit := total / bp.quantity
		cache[typeID] = perUnit
		return perUnit
	}

	var supercaps []int32
	if err := scanRows(ctx, pool,
		`SELECT type_id FROM inv_types WHERE group_id = ANY($1)`,
		func(s pgx.Rows) error {
			var id int32
			if err := s.Scan(&id); err != nil {
				return err
			}
			supercaps = append(supercaps, id)
			return nil
		}, supercapGroups); err != nil {
		return res, err
	}

	batch := &pgx.Batch{}
	queued := 0
	for _, id := range supercaps {
		cost := unitCost(id, map[int32]bool{})
		if cost <= 0 {
			continue
		}
		batch.Queue(`
            INSERT INTO custom_prices (type_id, date, price)
            VALUES ($1, $2::date, $3)
            ON CONFLICT (type_id, date) DO UPDATE SET price = EXCLUDED.price
        `, id, customPriceDate, math.Round(cost*buildCostMarkup))
		queued++
	}
	if err := sendBatch(ctx, pool, batch, queued); err != nil {
		return res, err
	}
	res.Rows = int64(queued)
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

func scanRows(ctx context.Context, pool *pgxpool.Pool, sql string, fn func(pgx.Rows) error, args ...any) error {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func sendBatch(ctx context.Context, pool *pgxpool.Pool, batch *pgx.Batch, n int) error {
	if n == 0 {
		return nil
	}
	br := pool.SendBatch(ctx, batch)
	for i := 0; i < n; i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	return br.Close()
}
