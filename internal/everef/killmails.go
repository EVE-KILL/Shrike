package everef

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/configstore"
	"github.com/eve-kill/shrike/internal/eve"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/jackc/pgx/v5/pgxpool"
)

// One tar.bz2 per day, each holding fifteen to twenty-five thousand killmails
// as individual JSON files:
//
//	https://data.everef.net/killmails/<year>/killmails-<date>.tar.bz2
//
// The archive carries the raw ESI documents, so every mail goes through the
// same parser a live one does. That is the point: an imported kill and a
// streamed kill must be indistinguishable once stored.

const killmailsPath = "/killmails"

// killmailBatch is how many parsed mails accumulate before a COPY. Large enough
// that the round trip disappears against the parse, small enough that a day's
// worth of parsed rows is never all in memory at once.
const killmailBatch = 2000

var killmailArchive = regexp.MustCompile(`killmails-(\d{4}-\d{2}-\d{2})\.tar\.bz2$`)

// DiscoverKillmailDays lists the archives published between two dates.
//
// EVE Ref has gaps, so the listing is authoritative rather than a generated
// date range — asking for a day that was never published is a wasted request
// and a misleading failure count.
func DiscoverKillmailDays(ctx context.Context, client *Client, from, to string) ([]string, error) {
	fromYear, err := strconv.Atoi(from[:4])
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q", from)
	}
	toYear, err := strconv.Atoi(to[:4])
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q", to)
	}

	var out []string
	for year := fromYear; year <= toYear; year++ {
		dates, err := client.List(ctx, client.url(fmt.Sprintf("%s/%d/", killmailsPath, year)), killmailArchive)
		if err != nil {
			if errors.Is(err, ErrNotPublished) {
				continue
			}
			return nil, err
		}
		for _, d := range dates {
			if d >= from && d <= to {
				out = append(out, d)
			}
		}
	}
	return out, nil
}

// KillmailImport carries what every day of the import needs.
type KillmailImport struct {
	Pool   *pgxpool.Pool
	Client *Client
	Cache  *eve.Cache
	Prices *eve.Prices

	// SkipExisting reads the IDs already stored for the day and skips parsing
	// them. The insert ignores conflicts either way, so this only saves the
	// parse — worth one indexed query when re-running a day that is mostly
	// duplicates, wasted work when the day is mostly new.
	SkipExisting bool
}

// ImportKillmailDay imports one daily archive.
func (k *KillmailImport) ImportKillmailDay(ctx context.Context, date string) (Result, error) {
	start := time.Now()
	res := Result{Name: date}

	// Every mail in the archive shares a kill date, and a price is resolved as
	// of that date — so one snapshot answers the whole day's valuations. Doing
	// it per killmail would be twenty thousand queries with one answer between
	// them.
	if _, err := k.Prices.Snapshot(ctx, date); err != nil {
		return res, fmt.Errorf("price snapshot for %s: %w", date, err)
	}

	var existing map[int64]bool
	if k.SkipExisting {
		var err error
		existing, err = storedKillmailIDs(ctx, k.Pool, date)
		if err != nil {
			return res, err
		}
	}

	url := k.Client.url(fmt.Sprintf("%s/%s/killmails-%s.tar.bz2", killmailsPath, date[:4], date))

	var batch []*killmail.Parsed
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		out, err := killmail.InsertBatch(ctx, k.Pool, batch)
		if err != nil {
			return err
		}
		res.Rows += out.Killmails
		batch = batch[:0]
		return nil
	}

	err := k.Client.WalkArchive(ctx, url, func(_ string, data []byte) error {
		var km killmail.ESIKillmail
		// A member that will not decode, or that decodes to something without
		// an ID, is counted and skipped: one corrupt file out of twenty
		// thousand is not a reason to abandon the day.
		if !decodeMember(data, &km) || km.KillmailID == 0 {
			res.Failed++
			return nil
		}
		res.Seen++

		if existing[km.KillmailID] {
			res.Skipped++
			return nil
		}

		// The archives carry killmail_hash on the document; ESI's own response
		// does not, because there the hash is half the request.
		parsed, err := killmail.Parse(ctx, k.Cache, k.Prices, &km, km.KillmailHash, km.WarID)
		if err != nil {
			// One document that cannot be valued or resolved must not strand
			// the entire daily archive. The TypeScript importer counts that
			// document as failed and keeps the successfully parsed mails.
			res.Failed++
			return nil
		}
		batch = append(batch, parsed)
		if len(batch) >= killmailBatch {
			return flush()
		}
		return nil
	})
	if errors.Is(err, ErrNotPublished) {
		res.Missing = true
		res.Elapsed = time.Since(start).Round(time.Millisecond).String()
		return res, nil
	}
	if err != nil {
		return res, err
	}
	if err := flush(); err != nil {
		return res, err
	}

	res.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return res, nil
}

// ImportKillmails imports each of the given days.
func (k *KillmailImport) ImportKillmails(ctx context.Context, dates []string, progress func(Result)) (Result, error) {
	start := time.Now()
	total := Result{Name: "killmails"}

	for _, date := range dates {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		r, err := k.ImportKillmailDay(ctx, date)
		if err != nil {
			return total, err
		}
		total.Seen += r.Seen
		total.Rows += r.Rows
		total.Skipped += r.Skipped
		total.Failed += r.Failed
		if progress != nil {
			progress(r)
		}
		// The bookmark advances only after a day finishes. If a batch or
		// download fails, this assignment is never reached and the failed day
		// remains the next day selected by the CLI.
		if err := configstore.Set(ctx, k.Pool, configstore.KeyKillmailsLastDate, date); err != nil {
			return total, err
		}
	}

	total.Elapsed = time.Since(start).Round(time.Millisecond).String()
	return total, nil
}

// storedKillmailIDs reads the IDs already held for one day.
//
// Bounded by killmail_time rather than by the ID list from the archive, so it
// is one indexed range scan and needs nothing from the download.
func storedKillmailIDs(ctx context.Context, pool *pgxpool.Pool, date string) (map[int64]bool, error) {
	rows, err := pool.Query(ctx, `
        SELECT killmail_id FROM killmails
        WHERE killmail_time >= $1::date AND killmail_time < ($1::date + interval '1 day')`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]bool, 32768)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// LatestKillmailDate reports the most recent stored kill, which is where a
// backfill resumes from.
func LatestKillmailDate(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var d *time.Time
	if err := pool.QueryRow(ctx, `SELECT max(killmail_time) FROM killmails`).Scan(&d); err != nil {
		return "", err
	}
	if d == nil {
		return "", nil
	}
	return d.UTC().Format(dateLayout), nil
}
