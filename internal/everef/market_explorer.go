package everef

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/pgbulk"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	marketOrdersLatestPath = "/market-orders/market-orders-latest.v3.csv.bz2"
	marketOrdersDataset    = "market-orders"
	marketHistoryDataset   = "market-history"
	MarketHistoryDays      = 90
)

var marketOrderColumns = []string{
	"duration", "is_buy_order", "issued", "location_id", "min_volume",
	"order_id", "price", "order_range", "system_id", "type_id",
	"volume_remain", "volume_total", "http_last_modified", "station_id",
	"region_id", "constellation_id", "snapshot_at",
}

var marketExplorerHistoryColumns = []string{
	"type_id", "region_id", "date", "average", "highest", "lowest",
	"order_count", "volume", "http_last_modified",
}

// ImportMarketOrders replaces the current all-region order book when EVE Ref's
// latest object has changed. The replacement and source-file bookmark commit
// together; a failed parse leaves both the old book and old bookmark intact.
func ImportMarketOrders(ctx context.Context, pool *pgxpool.Pool, client *Client, force bool) (Result, error) {
	start := time.Now()
	res := Result{Name: marketOrdersDataset}
	url := client.url(marketOrdersLatestPath)
	meta, err := client.Metadata(ctx, url)
	if err != nil {
		return res, err
	}
	if !force {
		unchanged, err := marketSourceUnchanged(ctx, pool, marketOrdersLatestPath, meta)
		if err != nil {
			return res, err
		}
		if unchanged {
			res.Skipped = 1
			res.Elapsed = time.Since(start).Round(time.Millisecond).String()
			return res, nil
		}
	}

	snapshotAt := time.Now().UTC()
	if meta.LastModified != nil {
		snapshotAt = *meta.LastModified
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const staging = "everef_staging_market_orders"
	if err := pgbulk.StagingTx(ctx, tx, staging, "market_orders"); err != nil {
		return res, err
	}
	w := pgbulk.NewCopier(ctx, tx, staging, marketOrderColumns)
	err = client.Stream(ctx, url, func(r io.Reader) error {
		return parseMarketOrders(r, snapshotAt, w)
	})
	if err != nil {
		return res, err
	}
	if err := w.Flush(); err != nil {
		return res, err
	}
	if w.Written() == 0 {
		return res, fmt.Errorf("market order snapshot is empty — refusing to replace the current book")
	}

	if _, err := tx.Exec(ctx, `TRUNCATE market_orders`); err != nil {
		return res, fmt.Errorf("clear market orders: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO market_orders (`+strings.Join(marketOrderColumns, ", ")+`)
		SELECT DISTINCT ON (order_id) `+strings.Join(marketOrderColumns, ", ")+`
		FROM `+staging+` ORDER BY order_id`); err != nil {
		return res, fmt.Errorf("replace market orders: %w", err)
	}
	if err := recordMarketSource(ctx, tx, marketOrdersLatestPath, marketOrdersDataset, meta, &snapshotAt, w.Written()); err != nil {
		return res, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Seen = w.Written()
	res.Rows = w.Written()
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

func parseMarketOrders(r io.Reader, snapshotAt time.Time, w *pgbulk.Copier) error {
	cr := csv.NewReader(r)
	cr.ReuseRecord = true
	header, err := cr.Read()
	if err != nil {
		return fmt.Errorf("read market order header: %w", err)
	}
	idx := headerIndex(header)
	required := []string{
		"duration", "is_buy_order", "issued", "location_id", "min_volume",
		"order_id", "price", "range", "system_id", "type_id",
		"volume_remain", "volume_total", "region_id",
	}
	if err := requireColumns(idx, required); err != nil {
		return fmt.Errorf("market orders CSV: %w", err)
	}

	for {
		rec, err := cr.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse market order row: %w", err)
		}
		row, ok := marketOrderRow(rec, idx, snapshotAt)
		if !ok {
			continue
		}
		if err := w.Add(row); err != nil {
			return err
		}
	}
}

func marketOrderRow(rec []string, idx map[string]int, snapshotAt time.Time) ([]any, bool) {
	duration, ok := parseInt64Field(rec, idx, "duration")
	if !ok || duration < 0 || duration > 32767 {
		return nil, false
	}
	buy, err := strconv.ParseBool(field(rec, idx, "is_buy_order"))
	if err != nil {
		return nil, false
	}
	issued, err := time.Parse(time.RFC3339, field(rec, idx, "issued"))
	if err != nil {
		return nil, false
	}
	locationID, lok := parseInt64Field(rec, idx, "location_id")
	minVolume, mok := parseInt64Field(rec, idx, "min_volume")
	orderID, ook := parseInt64Field(rec, idx, "order_id")
	systemID, sok := parseInt64Field(rec, idx, "system_id")
	typeID, tok := parseInt64Field(rec, idx, "type_id")
	remain, rok := parseInt64Field(rec, idx, "volume_remain")
	total, vok := parseInt64Field(rec, idx, "volume_total")
	regionID, regok := parseInt64Field(rec, idx, "region_id")
	price, perr := strconv.ParseFloat(field(rec, idx, "price"), 64)
	if !lok || !mok || !ook || !sok || !tok || !rok || !vok || !regok || perr != nil || price <= 0 {
		return nil, false
	}

	return []any{
		int16(duration), buy, issued.UTC(), locationID, minVolume, orderID, price,
		field(rec, idx, "range"), int32(systemID), int32(typeID), remain, total,
		optionalTime(field(rec, idx, "http_last_modified")),
		optionalInt64(field(rec, idx, "station_id")), int32(regionID),
		optionalInt32(field(rec, idx, "constellation_id")), snapshotAt,
	}, true
}

// ImportMarketHistoryDay replaces one all-region day if its object changed.
func ImportMarketHistoryDay(ctx context.Context, pool *pgxpool.Pool, client *Client, date string, force bool) (Result, error) {
	start := time.Now()
	res := Result{Name: date}
	day, err := time.Parse(dateLayout, date)
	if err != nil {
		return res, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
	}
	path := fmt.Sprintf("%s/%d/market-history-%s.csv.bz2", marketHistoryPath, day.Year(), date)
	url := client.url(path)
	meta, err := client.Metadata(ctx, url)
	if errors.Is(err, ErrNotPublished) {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}
	if err != nil {
		return res, err
	}
	if !force {
		unchanged, err := marketSourceUnchanged(ctx, pool, path, meta)
		if err != nil {
			return res, err
		}
		if unchanged {
			res.Skipped = 1
			res.Elapsed = time.Since(start).Round(time.Millisecond).String()
			return res, nil
		}
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return res, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	const staging = "everef_staging_market_history"
	if err := pgbulk.StagingTx(ctx, tx, staging, "market_region_history"); err != nil {
		return res, err
	}
	w := pgbulk.NewCopier(ctx, tx, staging, marketExplorerHistoryColumns)
	err = client.Stream(ctx, url, func(r io.Reader) error {
		return parseMarketExplorerHistory(r, date, w)
	})
	if err != nil {
		return res, err
	}
	if err := w.Flush(); err != nil {
		return res, err
	}
	if w.Written() == 0 {
		return res, fmt.Errorf("market history %s is empty — refusing to replace the stored day", date)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM market_region_history WHERE date = $1`, day); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, pgbulk.MergeSQL(
		"market_region_history", staging, marketExplorerHistoryColumns,
		[]string{"type_id", "region_id", "date"}, pgbulk.DoUpdate,
	)); err != nil {
		return res, fmt.Errorf("replace market history %s: %w", date, err)
	}
	fileTime := day.UTC()
	if err := recordMarketSource(ctx, tx, path, marketHistoryDataset, meta, &fileTime, w.Written()); err != nil {
		return res, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}

	res.Seen = w.Written()
	res.Rows = w.Written()
	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

func parseMarketExplorerHistory(r io.Reader, expectedDate string, w *pgbulk.Copier) error {
	cr := csv.NewReader(r)
	cr.ReuseRecord = true
	header, err := cr.Read()
	if err != nil {
		return fmt.Errorf("read market history header: %w", err)
	}
	idx := headerIndex(header)
	if err := requireColumns(idx, []string{
		"average", "date", "highest", "lowest", "order_count", "volume",
		"http_last_modified", "region_id", "type_id",
	}); err != nil {
		return fmt.Errorf("market history CSV: %w", err)
	}
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse market history row: %w", err)
		}
		if field(rec, idx, "date") != expectedDate {
			continue
		}
		regionID, rok := parseInt64Field(rec, idx, "region_id")
		typeID, tok := parseInt64Field(rec, idx, "type_id")
		if !rok || !tok {
			continue
		}
		if err := w.Add([]any{
			int32(typeID), int32(regionID), expectedDate,
			csvFloat(field(rec, idx, "average")),
			csvFloat(field(rec, idx, "highest")),
			csvFloat(field(rec, idx, "lowest")),
			csvInt(field(rec, idx, "order_count")),
			csvInt(field(rec, idx, "volume")),
			optionalTime(field(rec, idx, "http_last_modified")),
		}); err != nil {
			return err
		}
	}
}

// ReconcileMarketHistory checks the rolling window and downloads only new or
// changed objects. It also enforces the database retention boundary.
func ReconcileMarketHistory(ctx context.Context, pool *pgxpool.Pool, client *Client, days int, force bool, progress func(Result)) (Result, error) {
	start := time.Now()
	total := Result{Name: marketHistoryDataset}
	if days < 1 || days > 366 {
		return total, fmt.Errorf("history days must be between 1 and 366")
	}
	today := time.Now().UTC()
	for i := days; i >= 1; i-- {
		date := today.AddDate(0, 0, -i).Format(dateLayout)
		res, err := ImportMarketHistoryDay(ctx, pool, client, date, force)
		if err != nil {
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			res.Failed = 1
			total.Failed++
		} else {
			total.Seen += res.Seen
			total.Rows += res.Rows
			total.Skipped += res.Skipped
			if res.Missing {
				total.Failed++
			}
		}
		if progress != nil {
			progress(res)
		}
	}
	cutoff := today.AddDate(0, 0, -days).Format(dateLayout)
	tag, err := pool.Exec(ctx, `DELETE FROM market_region_history WHERE date < $1::date`, cutoff)
	if err != nil {
		return total, err
	}
	total.Related = tag.RowsAffected()
	total.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return total, nil
}

func marketSourceUnchanged(ctx context.Context, pool *pgxpool.Pool, path string, meta FileMetadata) (bool, error) {
	var etag *string
	var size int64
	var modified *time.Time
	err := pool.QueryRow(ctx, `
		SELECT etag, size_bytes, source_last_modified
		FROM market_source_files WHERE source_path = $1`, path).Scan(&etag, &size, &modified)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	storedETag := ""
	if etag != nil {
		storedETag = *etag
	}
	return storedETag == meta.ETag && size == meta.Size && sameTime(modified, meta.LastModified), nil
}

func recordMarketSource(ctx context.Context, tx pgx.Tx, path, dataset string, meta FileMetadata, fileTime *time.Time, rows int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO market_source_files (
			source_path, dataset, etag, size_bytes, source_last_modified,
			file_time, rows_imported, imported_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,now())
		ON CONFLICT (source_path) DO UPDATE SET
			dataset=EXCLUDED.dataset, etag=EXCLUDED.etag,
			size_bytes=EXCLUDED.size_bytes,
			source_last_modified=EXCLUDED.source_last_modified,
			file_time=EXCLUDED.file_time, rows_imported=EXCLUDED.rows_imported,
			imported_at=now()`,
		path, dataset, nullString(meta.ETag), meta.Size, meta.LastModified, fileTime, rows)
	return err
}

func headerIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, name := range header {
		idx[strings.TrimSpace(name)] = i
	}
	return idx
}

func requireColumns(idx map[string]int, names []string) error {
	for _, name := range names {
		if _, ok := idx[name]; !ok {
			return fmt.Errorf("missing column %q", name)
		}
	}
	return nil
}

func field(rec []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func parseInt64Field(rec []string, idx map[string]int, name string) (int64, bool) {
	v, err := strconv.ParseInt(field(rec, idx, name), 10, 64)
	return v, err == nil
}

func optionalInt64(value string) any {
	if value == "" {
		return nil
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil || v == 0 {
		return nil
	}
	return v
}

func optionalInt32(value string) any {
	v := optionalInt64(value)
	if v == nil {
		return nil
	}
	return int32(v.(int64))
}

func optionalTime(value string) any {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
