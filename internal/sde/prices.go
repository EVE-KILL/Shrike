package sde

import (
	"compress/bzip2"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Daily market history from EVE Ref, one bzip2'd CSV per day:
//
//	https://data.everef.net/market-history/<year>/market-history-<date>.csv.bz2
//
// Columns: average,date,highest,lowest,order_count,volume,http_last_modified,region_id,type_id
//
// This is not static data, but it lives here because supercapital valuation
// depends on it: those hulls never trade, so their price is computed from
// blueprint materials priced at market.

const marketHistoryBase = "https://data.everef.net/market-history"

// Only The Forge is stored. It contains Jita, which is the price every
// valuation in this codebase means, and keeping the other ~100 regions would
// multiply the table roughly tenfold for data nothing reads. Production holds
// exactly one region for the same reason.
const regionTheForge = 10000002

// PriceDayResult reports one imported day.
type PriceDayResult struct {
	Date    string `json:"date"`
	Rows    int64  `json:"rows"`
	Elapsed string `json:"elapsed"`
	Missing bool   `json:"missing,omitempty"`
}

// ImportPriceDay loads one day of market history.
//
// A day that is not published yet returns Missing rather than an error: EVE Ref
// publishes on its own schedule and the most recent day is routinely absent for
// several hours.
func ImportPriceDay(ctx context.Context, pool *pgxpool.Pool, day time.Time, userAgent string) (PriceDayResult, error) {
	start := time.Now()
	date := day.Format("2006-01-02")
	res := PriceDayResult{Date: date}

	url := fmt.Sprintf("%s/%d/market-history-%s.csv.bz2", marketHistoryBase, day.Year(), date)

	reqCtx, cancel := context.WithTimeout(ctx, externalTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("User-Agent", userAgent)
	// The body is already bzip2; asking for identity stops any transport-level
	// re-compression that would just be undone again.
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}
	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	// Decompressed and parsed as a stream: a day is ~48 k rows and there is no
	// reason to hold either the archive or the parsed table in memory.
	r := csv.NewReader(bzip2.NewReader(resp.Body))
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		return res, fmt.Errorf("read header: %w", err)
	}
	// Resolve columns by name — the column order has changed before and
	// positional parsing would silently swap values rather than fail.
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}
	for _, required := range []string{"type_id", "region_id", "date", "average", "highest", "lowest", "order_count", "volume"} {
		if _, ok := idx[required]; !ok {
			return res, fmt.Errorf("market history CSV is missing column %q", required)
		}
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	const staging = "sde_staging_prices"
	if _, err := conn.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE %s (LIKE public.prices INCLUDING DEFAULTS) ON COMMIT PRESERVE ROWS`,
		staging)); err != nil {
		return res, fmt.Errorf("create staging: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "DROP TABLE IF EXISTS "+staging)
	}()

	columns := []string{"type_id", "region_id", "date", "average", "highest", "lowest", "order_count", "volume"}
	w := &stagingWriter{ctx: ctx, conn: conn.Conn(), table: staging, columns: columns}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, fmt.Errorf("parse row: %w", err)
		}

		typeID, err1 := strconv.Atoi(rec[idx["type_id"]])
		regionID, err2 := strconv.Atoi(rec[idx["region_id"]])
		if err1 != nil || err2 != nil {
			continue
		}
		if regionID != regionTheForge {
			continue
		}
		if err := w.add([]any{
			int32(typeID), int32(regionID), rec[idx["date"]],
			csvFloat(rec[idx["average"]]), csvFloat(rec[idx["highest"]]), csvFloat(rec[idx["lowest"]]),
			csvInt(rec[idx["order_count"]]), csvInt(rec[idx["volume"]]),
		}); err != nil {
			return res, err
		}
	}
	if err := w.flush(); err != nil {
		return res, err
	}

	tbl := Table{Name: "prices", PK: []string{"type_id", "region_id", "date"}, Columns: columns}
	if _, err := conn.Exec(ctx, mergeSQL(tbl, staging)); err != nil {
		return res, fmt.Errorf("merge prices: %w", err)
	}

	res.Rows = w.written
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// ImportPriceRange loads the given number of days ending yesterday.
//
// Today is skipped: the day is not complete and EVE Ref does not publish it.
func ImportPriceRange(ctx context.Context, pool *pgxpool.Pool, days int, userAgent string, progress func(PriceDayResult)) ([]PriceDayResult, error) {
	var out []PriceDayResult
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i := days; i >= 1; i-- {
		day := today.AddDate(0, 0, -i)
		res, err := ImportPriceDay(ctx, pool, day, userAgent)
		if err != nil {
			return out, err
		}
		out = append(out, res)
		if progress != nil {
			progress(res)
		}
	}
	return out, nil
}

// priceProgressKey matches the key the TypeScript importer already writes, so
// both implementations read and advance the same bookmark rather than tracking
// progress separately.
const priceProgressKey = "everef:prices:last_date"

// RecordPriceProgress stores the most recent day imported.
func RecordPriceProgress(ctx context.Context, pool *pgxpool.Pool, date string) error {
	_, err := pool.Exec(ctx, `
        INSERT INTO config (key, value) VALUES ($1, $2)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
    `, priceProgressKey, date)
	return err
}

// LatestPriceDate reports the most recent day held, for incremental catch-up.
func LatestPriceDate(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var d *time.Time
	if err := pool.QueryRow(ctx, `SELECT max(date) FROM prices`).Scan(&d); err != nil {
		return "", err
	}
	if d == nil {
		return "", nil
	}
	return d.Format("2006-01-02"), nil
}

// csvFloat and csvInt map an empty field to NULL rather than zero — an absent
// price is not a price of zero, and averaging over zeros would drag valuations
// down.
func csvFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}

func csvInt(s string) *int64 {
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// Some counts arrive as floats ("12.0").
		if f, ferr := strconv.ParseFloat(s, 64); ferr == nil {
			v := int64(f)
			return &v
		}
		return nil
	}
	return &n
}
