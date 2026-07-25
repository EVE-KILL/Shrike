package sde

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dust 514 static data, bundled because CCP removed it from the SDE.
//
// Dust 514 was shut down in 2016 and its data dropped from the export in March
// 2017, but killmails from that period are still in the database and still need
// their type, group and category names to render. Without this seed those kills
// display as bare numeric IDs.
//
// Verified against the 2026-07-23 archive (build 3444265): CCP still ships the
// Dust types, groups and categories, but no longer the market groups. The full
// seed is kept regardless — the market groups were dropped at some point too,
// and the rest may follow.
//
//go:embed data/dust514.json
var dust514Raw []byte

type dustData struct {
	Categories []struct {
		CategoryID int32   `json:"category_id"`
		Name       *string `json:"name"`
		Published  *bool   `json:"published"`
	} `json:"categories"`

	Groups []struct {
		GroupID    int32   `json:"group_id"`
		CategoryID *int32  `json:"category_id"`
		Name       *string `json:"name"`
		Published  *bool   `json:"published"`
		IconID     *int32  `json:"icon_id"`
	} `json:"groups"`

	MarketGroups []struct {
		MarketGroupID int32   `json:"market_group_id"`
		ParentGroupID *int32  `json:"parent_group_id"`
		Name          *string `json:"name"`
		IconID        *int32  `json:"icon_id"`
		HasTypes      *bool   `json:"has_types"`
	} `json:"market_groups"`

	Types []struct {
		TypeID        int32    `json:"type_id"`
		GroupID       *int32   `json:"group_id"`
		Name          *string  `json:"name"`
		Description   *string  `json:"description"`
		Mass          *float64 `json:"mass"`
		Volume        *float64 `json:"volume"`
		Capacity      *float64 `json:"capacity"`
		PortionSize   *int32   `json:"portion_size"`
		Radius        *float64 `json:"radius"`
		Published     *bool    `json:"published"`
		MarketGroupID *int32   `json:"market_group_id"`
		IconID        *int32   `json:"icon_id"`
		GraphicID     *int32   `json:"graphic_id"`
		MetaGroupID   *int32   `json:"meta_group_id"`
		RaceID        *int32   `json:"race_id"`
		BasePrice     *float64 `json:"base_price"`
	} `json:"types"`
}

// SeedResult reports what the Dust 514 seed wrote.
type SeedResult struct {
	Categories   int64  `json:"categories"`
	Groups       int64  `json:"groups"`
	MarketGroups int64  `json:"market_groups"`
	Types        int64  `json:"types"`
	Elapsed      string `json:"elapsed"`
}

// SeedDust514 upserts the bundled data.
//
// Runs after the archive import so that where CCP still ships a record, the
// live archive wins on the first pass and this only fills what is missing —
// both write the same rows, so the order is about provenance, not conflict.
func SeedDust514(ctx context.Context, pool *pgxpool.Pool) (SeedResult, error) {
	start := time.Now()
	var res SeedResult

	var d dustData
	if err := json.Unmarshal(dust514Raw, &d); err != nil {
		return res, fmt.Errorf("parse bundled dust514 data: %w", err)
	}

	// One transaction: the four tables reference each other by category and
	// group, so a partial seed would leave types pointing at absent groups.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	for _, c := range d.Categories {
		if _, err := tx.Exec(ctx, `
            INSERT INTO inv_categories (category_id, name, published, icon_id)
            VALUES ($1, $2, $3, NULL)
            ON CONFLICT (category_id) DO UPDATE
            SET name = EXCLUDED.name, published = EXCLUDED.published
        `, c.CategoryID, c.Name, c.Published); err != nil {
			return res, fmt.Errorf("seed dust category %d: %w", c.CategoryID, err)
		}
		res.Categories++
	}

	for _, g := range d.Groups {
		if _, err := tx.Exec(ctx, `
            INSERT INTO inv_groups (group_id, category_id, name, published, icon_id)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (group_id) DO UPDATE
            SET category_id = EXCLUDED.category_id, name = EXCLUDED.name,
                published = EXCLUDED.published, icon_id = EXCLUDED.icon_id
        `, g.GroupID, g.CategoryID, g.Name, g.Published, g.IconID); err != nil {
			return res, fmt.Errorf("seed dust group %d: %w", g.GroupID, err)
		}
		res.Groups++
	}

	for _, m := range d.MarketGroups {
		if _, err := tx.Exec(ctx, `
            INSERT INTO inv_market_groups
                (market_group_id, parent_group_id, name, description, icon_id, has_types)
            VALUES ($1, $2, $3, NULL, $4, $5)
            ON CONFLICT (market_group_id) DO UPDATE
            SET parent_group_id = EXCLUDED.parent_group_id, name = EXCLUDED.name,
                icon_id = EXCLUDED.icon_id, has_types = EXCLUDED.has_types
        `, m.MarketGroupID, m.ParentGroupID, m.Name, m.IconID, m.HasTypes); err != nil {
			return res, fmt.Errorf("seed dust market group %d: %w", m.MarketGroupID, err)
		}
		res.MarketGroups++
	}

	// 2,527 types, so batched rather than one round trip each.
	batch := &pgx.Batch{}
	for _, t := range d.Types {
		batch.Queue(`
            INSERT INTO inv_types
                (type_id, group_id, name, description, mass, volume, capacity,
                 portion_size, radius, published, market_group_id, icon_id,
                 graphic_id, meta_group_id, race_id, base_price)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
            ON CONFLICT (type_id) DO UPDATE
            SET group_id = EXCLUDED.group_id, name = EXCLUDED.name,
                description = EXCLUDED.description, mass = EXCLUDED.mass,
                volume = EXCLUDED.volume, capacity = EXCLUDED.capacity,
                portion_size = EXCLUDED.portion_size, radius = EXCLUDED.radius,
                published = EXCLUDED.published,
                market_group_id = EXCLUDED.market_group_id,
                icon_id = EXCLUDED.icon_id, graphic_id = EXCLUDED.graphic_id,
                meta_group_id = EXCLUDED.meta_group_id, race_id = EXCLUDED.race_id,
                base_price = EXCLUDED.base_price
        `, t.TypeID, t.GroupID, t.Name, t.Description, t.Mass, t.Volume,
			t.Capacity, t.PortionSize, t.Radius, t.Published, t.MarketGroupID,
			t.IconID, t.GraphicID, t.MetaGroupID, t.RaceID, t.BasePrice)
	}

	br := tx.SendBatch(ctx, batch)
	for range d.Types {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return res, fmt.Errorf("seed dust types: %w", err)
		}
		res.Types++
	}
	if err := br.Close(); err != nil {
		return res, err
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}
