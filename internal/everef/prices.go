package everef

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Daily market history, one bzip2'd CSV per day:
//
//	https://data.everef.net/market-history/<year>/market-history-<date>.csv.bz2
//
// Columns: average,date,highest,lowest,order_count,volume,http_last_modified,region_id,type_id

const marketHistoryPath = "/market-history"

// RegionTheForge is the only region stored.
//
// It contains Jita, which is the price every valuation in this codebase means.
// Keeping the other hundred-odd regions would multiply a 23-million-row table
// tenfold for data nothing reads.
const RegionTheForge = 10000002

var priceColumns = []string{
	"type_id", "region_id", "date", "average", "highest", "lowest", "order_count", "volume",
}

// ImportPriceDay loads one day of market history.
//
// A day EVE Ref has not published yet comes back with Missing set rather than
// an error — the most recent day is routinely absent for several hours.
func ImportPriceDay(ctx context.Context, pool *pgxpool.Pool, client *Client, date string) (Result, error) {
	start := time.Now()
	res := Result{Name: date}

	day, err := time.Parse(dateLayout, date)
	if err != nil {
		return res, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
	}
	url := client.url(fmt.Sprintf("%s/%d/market-history-%s.csv.bz2", marketHistoryPath, day.Year(), date))

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()

	const staging = "everef_staging_prices"
	drop, err := pgbulk.Staging(ctx, conn.Conn(), staging, "prices")
	if err != nil {
		return res, err
	}
	defer drop()

	w := pgbulk.NewCopier(ctx, conn.Conn(), staging, priceColumns)

	err = client.Stream(ctx, url, func(r io.Reader) error {
		cr := csv.NewReader(r)
		cr.ReuseRecord = true

		header, err := cr.Read()
		if err != nil {
			return fmt.Errorf("read header: %w", err)
		}
		// Resolved by name: the column order has changed before, and positional
		// parsing would silently swap values rather than fail.
		idx := map[string]int{}
		for i, h := range header {
			idx[h] = i
		}
		for _, required := range priceColumns {
			if _, ok := idx[required]; !ok {
				return fmt.Errorf("market history CSV is missing column %q", required)
			}
		}

		for {
			rec, err := cr.Read()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("parse row: %w", err)
			}

			regionID, err := strconv.Atoi(rec[idx["region_id"]])
			if err != nil || regionID != RegionTheForge {
				continue
			}
			typeID, err := strconv.Atoi(rec[idx["type_id"]])
			if err != nil {
				continue
			}

			// A row with no average is a day the type was listed but never
			// traded. Storing it would put a zero into the price history, and
			// "latest average at or before this date" would then value the item
			// at nothing. Production holds no such row.
			average := csvFloat(rec[idx["average"]])
			if average == nil || *average <= 0 {
				continue
			}

			if err := w.Add([]any{
				int32(typeID), int32(regionID), rec[idx["date"]],
				*average,
				csvFloat(rec[idx["highest"]]), csvFloat(rec[idx["lowest"]]),
				csvInt(rec[idx["order_count"]]), csvInt(rec[idx["volume"]]),
			}); err != nil {
				return err
			}
		}
	})
	if errors.Is(err, ErrNotPublished) {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}
	if err != nil {
		return res, err
	}

	if err := w.Flush(); err != nil {
		return res, err
	}

	// DoNothing, matching the TypeScript. EVE Ref occasionally republishes a
	// corrected day, and DoUpdate would pick that up — but it would also
	// silently reprice history on any re-run, and every stored killmail value
	// was computed from what was held at the time. Keeping the first version
	// read is what makes a re-import idempotent.
	if _, err := conn.Exec(ctx, pgbulk.MergeSQL("prices", staging, priceColumns,
		[]string{"type_id", "region_id", "date"}, pgbulk.DoNothing)); err != nil {
		return res, fmt.Errorf("merge prices: %w", err)
	}

	res.Rows = w.Written()
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// ImportPrices loads each of the given days, reporting progress as it goes.
func ImportPrices(ctx context.Context, pool *pgxpool.Pool, client *Client, dates []string, progress func(Result)) (Result, error) {
	start := time.Now()
	total := Result{Name: "prices"}

	for _, date := range dates {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		r, err := ImportPriceDay(ctx, pool, client, date)
		if err != nil {
			// The TypeScript command treats one unavailable or malformed day as
			// a per-date failure and continues through the requested range.
			// Keep the bookmark where it was so a later run can repair the gap.
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			r.Failed = 1
			total.Failed++
			if progress != nil {
				progress(r)
			}
			continue
		}
		total.Rows += r.Rows
		if r.Missing {
			total.Failed++
		} else if err := configstore.Set(ctx, pool, configstore.KeyPricesLastDate, date); err != nil {
			return total, err
		}
		if progress != nil {
			progress(r)
		}
	}

	total.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return total, nil
}

// LatestPriceDate reports the most recent day held, which is where a backfill
// resumes from.
func LatestPriceDate(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var d *time.Time
	if err := pool.QueryRow(ctx, `SELECT max(date) FROM prices`).Scan(&d); err != nil {
		return "", err
	}
	if d == nil {
		return "", nil
	}
	return d.Format(dateLayout), nil
}

// csvFloat and csvInt map an empty or zero field to NULL.
//
// Zero is folded into NULL rather than stored because the TypeScript coerces
// with `||` and production consequently holds neither: an absent price is not a
// price of zero, and a highest-price of zero is a parse artefact.
func csvFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f == 0 {
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
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return nil
		}
		n = int64(f)
	}
	if n == 0 {
		return nil
	}
	return &n
}
